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

export type LayoutName = 'history' | 'layered' | 'random' | 'circular' | 'circlepack' | 'fa2-worker';

export const LAYOUT_LABELS: Record<LayoutName, string> = {
  history: 'rows by contribution',
  layered: 'layers by provenance depth',
  random: 'random',
  circular: 'circular',
  circlepack: 'circle-packed by class',
  'fa2-worker': 'ForceAtlas2 (worker)',
};

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
