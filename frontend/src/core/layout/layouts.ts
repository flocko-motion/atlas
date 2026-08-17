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
 * STRATA are the bands, bottom of the screen first. Sigma maps a larger graph y higher up
 * (`y: (1 - y) * height / 2`), so the array index *is* the y ordering.
 *
 * The order follows the ADT: bookkeeping, then what entered, what was concluded, the things
 * distilled, and the links between them. It is monotone with provenance, so an edge points
 * down or sideways and almost never up — provenance becomes a direction rather than
 * something to trace. Contribution sits at the edge because it is the band the class filter
 * drops, and dropping the bottom band leaves the picture whole.
 */
export const STRATA = ['contribution', 'source', 'derivation', 'entity', 'relation'] as const;

/** stratumOf is the band a claim class belongs to; an unknown class sits with the sources. */
export function stratumOf(cls: string): number {
  const at = STRATA.indexOf(cls as (typeof STRATA)[number]);
  return at === -1 ? STRATA.indexOf('source') : at;
}

export interface TimelineContext {
  /** Where each claim's instant sits on the axis. */
  toX: (at: number) => number;
  /** The instant a claim carries, in epoch ms. */
  createdAt: (node: string) => number;
  /** The claim's class — the band it sits in. */
  classOf: (node: string) => string;
  /**
   * The strata to give room to, or empty for all of them. A hidden stratum takes no height,
   * so the bands that remain fill the picture rather than leaving a gap where it was.
   */
  visible?: readonly string[];
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
 * assignTimeline puts time on x and class strata on y.
 *
 * Claims sharing an instant share an x, so a stack means "simultaneous" and nothing else.
 * Within a band, lanes are handed out round-robin in time order: consecutive claims land on
 * different lanes, so neighbours in time do not draw over each other even where the axis puts
 * them close together.
 *
 * The visible bands divide the whole height between them, so the picture is always full and
 * hiding a stratum gives its room to the others rather than leaving a hole.
 */
export function assignTimeline(graph: DirectedGraph, ctx: TimelineContext): void {
  const shown = ctx.visible && ctx.visible.length > 0 ? ctx.visible : STRATA;
  const bands = STRATA.filter((stratum) => shown.includes(stratum));
  if (bands.length === 0) return;

  const bandHeight = TIMELINE_HEIGHT / bands.length;
  const lanes = Math.max(1, Math.floor(bandHeight / LANE_UNITS));
  const laneGap = bandHeight / lanes;
  // Bottom of the picture first, matching STRATA — Sigma draws a larger y higher up.
  const baseOf = new Map<string, number>(bands.map((stratum, i) => [stratum, i * bandHeight]));

  // Time order per band, so the round robin walks the claims as a reader would.
  const byBand = new Map<string, string[]>();
  graph.forEachNode((node) => {
    const stratum = STRATA[stratumOf(ctx.classOf(node))];
    const bucket = byBand.get(stratum);
    if (bucket) bucket.push(node);
    else byBand.set(stratum, [node]);
  });

  const stretch = ctx.yStretch ?? 1;
  const placed = new Map<string, { x: number; y: number }>();
  for (const [stratum, nodes] of byBand) {
    const base = baseOf.get(stratum);
    nodes.sort((a, b) => ctx.createdAt(a) - ctx.createdAt(b));
    nodes.forEach((node, i) => {
      const x = ctx.toX(ctx.createdAt(node));
      // A hidden band has no base; its claims keep a position so the reducer, not the
      // layout, is what hides them.
      const y = ((base ?? 0) + (i % lanes) * laneGap) * stretch;
      placed.set(node, { x, y });
    });
  }

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
