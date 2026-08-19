/**
 * package: core / layout
 * type:    logic
 * job:     position calculators — take a graph, return coordinates
 * limits:  headless; they draw nothing (-> render/renderer)
 *
 * `history` is the default the measurements argued for: 21 ms at 50k, deterministic, one
 * contribution per row. ForceAtlas2 suits drill-downs of a few thousand claims
 * (13–130 ms/iter); at archive scale it is ~1 500 ms/iter.
 */

import forceAtlas2 from 'graphology-layout-forceatlas2';
// Explicit extension: the package has no exports map, so Node's ESM resolver
// needs the file name. Vite resolves it either way.
import FA2Layout from 'graphology-layout-forceatlas2/worker.js';
import { circlepack, circular, random } from 'graphology-layout';
import type { DirectedGraph } from 'graphology';
import { requireContribution } from '../graph/shape.ts';
import { hashString } from '../hash.ts';

export type LayoutName =
  | 'timeline'
  | 'history'
  | 'layered'
  | 'random'
  | 'circular'
  | 'circlepack'
  | 'fa2-worker';

export const LAYOUT_LABELS: Record<LayoutName, string> = {
  timeline: 'time, in strata by class',
  history: 'rows by contribution',
  layered: 'layers by provenance depth',
  random: 'random',
  circular: 'circular',
  circlepack: 'circle-packed by class',
  'fa2-worker': 'ForceAtlas2 (worker)',
};

/**
 * STRATA are the classes a claim may belong to, top of the picture first when reversed for
 * display (-> ui/panes/ViewPane). Every class keeps its own colour (-> core/graph/build.ts
 * CLASS_COLOR) and its own filter toggle; which vertical band it draws in is a separate
 * question BANDS answers below — entity and relation are two classes sharing one band.
 */
export const STRATA = ['contribution', 'source', 'derivation', 'entity', 'relation'] as const;

/**
 * BANDS are the timeline layout's vertical strata, bottom of the screen first. Sigma maps a
 * larger graph y higher up (`y: (1 - y) * height / 2`), so the array index *is* the y ordering.
 *
 * The order follows the ADT: bookkeeping, then what entered, what was concluded, then the
 * semantic layer built from it. It is monotone with provenance, so an edge points down or
 * sideways and almost never up — provenance becomes a direction rather than something to
 * trace. Contribution sits at the edge because it is the band the class filter drops, and
 * dropping the bottom band leaves the picture whole.
 *
 * Entity and relation share the top band as one *semantic layer* — entities and the relations
 * between them are one reading of the graph, not two — while each keeps its own colour and its
 * own filter toggle (-> STRATA); only the height they are allotted is pooled. Left separate, a
 * relation is rare enough in most archives that its own band sits almost empty while the four
 * content bands beneath it are cramped for room they are not given.
 */
const BANDS = ['contribution', 'source', 'derivation', 'semantic'] as const;

/** bandOf names the band a class draws in — entity and relation share one, semantic. */
function bandOf(cls: string): (typeof BANDS)[number] {
  if (cls === 'entity' || cls === 'relation') return 'semantic';
  return (BANDS as readonly string[]).includes(cls) ? (cls as (typeof BANDS)[number]) : 'source';
}

/** stratumOf is the band index a claim class draws in; an unknown class sits with the sources. */
export function stratumOf(cls: string): number {
  return BANDS.indexOf(bandOf(cls));
}

export interface TimelineContext {
  /** Where each claim's instant sits on the axis. */
  toX: (at: number) => number;
  /** The instant a claim carries, in epoch ms. */
  createdAt: (node: string) => number;
  /** The claim's class — the band it sits in. */
  classOf: (node: string) => string;
  /** The claim's subtype — which subband of its band it sits in (-> HEAD_SUBBAND_FRACTION). */
  subOf: (node: string) => string;
  /**
   * How far the strata are stretched, 1 being the height below. It scales the finished
   * positions rather than the height they are laid out in: a band keeps its lanes and its
   * neighbours, and only the room between them grows.
   */
  yStretch?: number;
}

/** How tall the whole picture is, in graph units — the extent the renderer normalises against. */
export const TIMELINE_HEIGHT = 1000;

/** Room one lane wants, which sets how many a band can hold. */
const LANE_UNITS = 16;

/**
 * Room left below the bottom band and above the top one — asymmetric, and more asymmetric than
 * equal canvas-pixel spacing would suggest. The ruler is a hard visual landmark right below
 * graph-y 0 (its own border and background, not just empty pane), so the same gap that reads as
 * spacious above the top band — nothing there but the pane's edge — reads as tight next to it;
 * the bottom margin has to overcompensate for that anchor, not just match the top one's pixels.
 * Sigma draws a larger graph y higher up the screen, so it is graph-y 0 that sits by the ruler.
 */
const TIMELINE_MARGIN_BOTTOM = LANE_UNITS * 2.5;
const TIMELINE_MARGIN_TOP = LANE_UNITS / 8;

/**
 * Share of the contribution band given to `contribution/head` claims — the bottom of the band,
 * every other contribution subtype (contributor, branches, delete, expiry) taking the rest
 * above it. A head is one per currently-open line of work, not one per claim the way a
 * contributor or a branch-table revision is, so it is a small, distinctive slice of the band's
 * traffic — set apart in its own strip rather than scattered in among the rest of the
 * bookkeeping, it reads as what it is instead of one more dot among many.
 */
const HEAD_SUBBAND_FRACTION = 0.3;

/** One vertical slice a claim may be placed in: where it starts, and how many lanes it holds. */
interface Slot {
  base: number;
  lanes: number;
  laneGap: number;
}

/** slotOf builds the lane geometry for a height, floored at one lane so a slice never vanishes. */
function slotOf(base: number, height: number): Slot {
  const lanes = Math.max(1, Math.floor(height / LANE_UNITS));
  return { base, lanes, laneGap: height / lanes };
}

/**
 * assignTimeline puts time on x and class strata on y.
 *
 * Claims sharing an instant share an x, so a stack means "simultaneous" and nothing else.
 * Within a slot, a claim's lane is a hash of its id — not its position in time order. A
 * round-robin by time-sorted index put neighbours in time on consecutive lanes, so as x
 * advanced the lane advanced in lockstep with it: a staircase, not a scatter, and captions
 * collided along the diagonal it drew. A hash gives every claim a lane independent of when its
 * neighbours fall, which is what the round-robin was reaching for — spreading claims apart —
 * without the diagonal that came along with it.
 *
 * Every one of the four bands gets its fixed share of the height, whether or not any of its
 * classes is currently shown: the View tab's stratum toggles hide claims (-> render/renderer
 * admits), never move them, so the band heights are set from the potential four, never from how
 * many happen to be switched on right now — a toggle must not shift anyone else's claims. The
 * contribution band alone further splits into two slots by subtype (-> HEAD_SUBBAND_FRACTION),
 * the same reasoning one level down: a subtype's slot is fixed whether or not that subtype
 * happens to be present in what is loaded.
 */
export function assignTimeline(graph: DirectedGraph, ctx: TimelineContext): void {
  const bandHeight = (TIMELINE_HEIGHT - TIMELINE_MARGIN_BOTTOM - TIMELINE_MARGIN_TOP) / BANDS.length;
  // Bottom of the picture first, matching BANDS — Sigma draws a larger y higher up.
  const slotsOf = new Map<string, { rest: Slot; head?: Slot }>(
    BANDS.map((band, i) => {
      const bandBase = TIMELINE_MARGIN_BOTTOM + i * bandHeight;
      if (band !== 'contribution') return [band, { rest: slotOf(bandBase, bandHeight) }];
      const headHeight = bandHeight * HEAD_SUBBAND_FRACTION;
      return [
        band,
        { head: slotOf(bandBase, headHeight), rest: slotOf(bandBase + headHeight, bandHeight - headHeight) },
      ];
    }),
  );

  const stretch = ctx.yStretch ?? 1;
  const placed = new Map<string, { x: number; y: number }>();
  graph.forEachNode((node) => {
    const band = BANDS[stratumOf(ctx.classOf(node))];
    const slots = slotsOf.get(band);
    const slot = (slots?.head && ctx.subOf(node) === 'head' ? slots.head : slots?.rest) ?? slotOf(0, 0);
    const lane = hashString(node) % slot.lanes;
    const x = ctx.toX(ctx.createdAt(node));
    const y = (slot.base + lane * slot.laneGap) * stretch;
    placed.set(node, { x, y });
  });

  graph.updateEachNodeAttributes((node, attr) => {
    const at = placed.get(node);
    if (at) {
      attr.x = at.x;
      attr.y = at.y;
    }
    return attr;
  });
}

/**
 * assignHistory positions claims by the contribution that added them: y is the
 * contribution, x is the position within it.
 */
export function assignHistory(
  graph: DirectedGraph,
  contribution: (node: string) => number,
  gap = { x: 24, y: 18 },
): void {
  const perRow = new Map<number, number>();
  graph.forEachNode((node) => {
    const c = requireContribution(contribution, node);
    perRow.set(c, (perRow.get(c) ?? 0) + 1);
  });
  const cursor = new Map<number, number>();
  graph.updateEachNodeAttributes((node, attr) => {
    const c = requireContribution(contribution, node);
    const i = cursor.get(c) ?? 0;
    cursor.set(c, i + 1);
    const width = perRow.get(c) ?? 1;
    attr.x = (i - (width - 1) / 2) * gap.x;
    attr.y = c * gap.y;
    return attr;
  });
}

/** assignLayered positions claims by provenance depth: y is the layer. */
export function assignLayered(
  graph: DirectedGraph,
  depth: Map<string, number>,
  gap = { x: 24, y: 12 },
): void {
  const perLayer = new Map<number, number>();
  graph.forEachNode((node) => {
    const d = depth.get(node) ?? 0;
    perLayer.set(d, (perLayer.get(d) ?? 0) + 1);
  });
  const cursor = new Map<number, number>();
  graph.updateEachNodeAttributes((node, attr) => {
    const d = depth.get(node) ?? 0;
    const i = cursor.get(d) ?? 0;
    cursor.set(d, i + 1);
    const width = perLayer.get(d) ?? 1;
    attr.x = (i - (width - 1) / 2) * gap.x;
    attr.y = d * gap.y;
    return attr;
  });
}

export interface LayoutContext {
  depth: Map<string, number>;
  contribution: (node: string) => number;
  /** What the timeline layout needs; absent, it falls back to provenance depth. */
  timeline?: TimelineContext;
  /** Wall-clock budget for the worker layout, in ms. */
  fa2Ms?: number;
}

/** apply runs a named layout and returns how long it took. */
export async function apply(
  graph: DirectedGraph,
  layout: LayoutName,
  ctx: LayoutContext,
): Promise<number> {
  const t0 = performance.now();
  switch (layout) {
    case 'timeline':
      // Without a scale there is no axis, so depth stands in rather than piling every claim
      // at x = 0.
      if (ctx.timeline) assignTimeline(graph, ctx.timeline);
      else assignLayered(graph, ctx.depth);
      break;
    case 'history':
      assignHistory(graph, ctx.contribution);
      break;
    case 'layered':
      assignLayered(graph, ctx.depth);
      break;
    case 'random':
      random.assign(graph, { scale: 1000 });
      break;
    case 'circular':
      circular.assign(graph, { scale: 1000 });
      break;
    case 'circlepack':
      circlepack.assign(graph, { hierarchyAttributes: ['cls'] });
      break;
    case 'fa2-worker': {
      // Off the main thread, per the design's rule that a blocked main thread is
      // 0 fps. Seeded cheaply first, then settled for a budget.
      random.assign(graph, { scale: 1000 });
      const supervisor = new FA2Layout(graph, {
        settings: { ...forceAtlas2.inferSettings(graph), barnesHutOptimize: true },
      });
      supervisor.start();
      await new Promise((resolve) => setTimeout(resolve, ctx.fa2Ms ?? 5000));
      supervisor.kill();
      break;
    }
  }
  return performance.now() - t0;
}
