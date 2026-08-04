/**
 * package: core / graph
 * type:    logic
 * job:     measure the graph's shape — height, layers, contribution rows
 * limits:  headless; it measures, it does not position (-> core/layout)
 *
 * Two views of "tall" disagree, and that decided the default layout: provenance depth
 * piles content into a few huge layers, while contribution order gives one row each.
 */

import type { DirectedGraph } from 'graphology';

export interface DepthStats {
  /** Longest directed path in claims — the archive's height. */
  height: number;
  meanDepth: number;
  p95Depth: number;
  /** Distinct depths, i.e. how many layers a layered layout would draw. */
  layers: number;
  /** Claims in the widest layer. */
  widestLayer: number;
  /** height ÷ widestLayer — how ribbon-shaped the picture is. */
  aspect: number;
  computeMs: number;
}

export interface HistoryStats {
  rows: number;
  widestRow: number;
  meanRow: number;
  aspect: number;
}

/**
 * requireContribution fails loudly on a missing contribution attribute. Without it
 * a graph built without that attribute reports every claim in row 0 — which looks
 * like a measurement rather than a mistake, and did once.
 */
export function requireContribution(contributionOf: (node: string) => number, node: string): number {
  const c = contributionOf(node);
  if (typeof c !== 'number' || !Number.isFinite(c)) {
    throw new Error(`node ${node} has no numeric 'contribution' attribute — build the graph with it`);
  }
  return c;
}

/** contributionOf reads the contribution index a claim was added in. */
export const contributionOf = (graph: DirectedGraph) => (node: string) =>
  graph.getNodeAttribute(node, 'contribution') as number;

/**
 * depths computes each claim's provenance depth in one pass.
 *
 * No topological sort is needed: claims are inserted in creation order and an edge
 * always points at an *earlier* claim, so every reference already has its depth when
 * we reach the claim citing it. That makes height an O(V+E) property — which is why
 * a layered layout is nearly free.
 */
export function depths(graph: DirectedGraph): { depth: Map<string, number>; stats: DepthStats } {
  const t0 = performance.now();
  const depth = new Map<string, number>();
  const perLayer = new Map<number, number>();
  let height = 0;
  let sum = 0;

  graph.forEachNode((node) => {
    let d = 0;
    graph.forEachOutNeighbor(node, (ref) => {
      const rd = depth.get(ref);
      if (rd !== undefined && rd + 1 > d) d = rd + 1;
    });
    depth.set(node, d);
    perLayer.set(d, (perLayer.get(d) ?? 0) + 1);
    if (d > height) height = d;
    sum += d;
  });

  const sorted = [...depth.values()].sort((a, b) => a - b);
  const widestLayer = Math.max(...perLayer.values(), 0);
  return {
    depth,
    stats: {
      height,
      meanDepth: sum / (graph.order || 1),
      p95Depth: sorted[Math.min(sorted.length - 1, Math.floor(sorted.length * 0.95))] ?? 0,
      layers: perLayer.size,
      widestLayer,
      aspect: height / (widestLayer || 1),
      computeMs: performance.now() - t0,
    },
  };
}

/** historyStats describes the shape contribution order gives the picture. */
export function historyStats(
  graph: DirectedGraph,
  contribution: (node: string) => number,
): HistoryStats {
  const perRow = new Map<number, number>();
  graph.forEachNode((node) => {
    const c = requireContribution(contribution, node);
    perRow.set(c, (perRow.get(c) ?? 0) + 1);
  });
  const widestRow = Math.max(...perRow.values(), 0);
  const rows = perRow.size;
  return { rows, widestRow, meanRow: graph.order / (rows || 1), aspect: rows / (widestRow || 1) };
}
