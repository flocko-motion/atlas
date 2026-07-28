/**
 * Claims → graphology. This is the step a real explorer performs on every read,
 * so it is measured separately from generation: a query returns claims, the
 * client turns them into a renderable graph.
 *
 * Two attribute profiles are built so the spike can price display metadata.
 * `lean` carries only what Sigma reads to paint a node (x, y, size, color);
 * `full` also carries the claim metadata an inspector panel needs. At 100k
 * claims the difference is the difference between a comfortable heap and a
 * hostile one, so it is a measurement, not a guess.
 */

import { DirectedGraph } from 'graphology';
import type { MockArchive } from './model.ts';
import { classOf } from './model.ts';

/** CLASS_COLOR maps a node class to its paint colour (Okabe–Ito, colour-safe). */
export const CLASS_COLOR: Record<string, string> = {
  source: '#0072b2',
  derivation: '#e69f00',
  entity: '#009e73',
  relation: '#cc79a7',
  contribution: '#7f7f7f',
};

/** CLASS_SIZE gives each class a base radius — hubs get scaled up by degree. */
const CLASS_SIZE: Record<string, number> = {
  source: 2,
  derivation: 1.5,
  entity: 3,
  relation: 2,
  contribution: 4,
};

export interface BuildOptions {
  /** `full` also stores claim metadata on each node. */
  attrs?: 'lean' | 'full';
  /**
   * Edge types to leave out, matched by prefix. A renderer needs this: every
   * claim carries a `contribution/contributor` edge, so a contributor node's
   * degree equals the number of claims it signed — a star that dominates both
   * force layout and the picture.
   */
  dropEdgeTypes?: string[];
}

export interface BuildResult {
  graph: DirectedGraph;
  buildMs: number;
  /** Parallel edges collapsed by merge (same pair, second type). */
  mergedEdges: number;
  /** References to claims outside the delivered set — real reads have these. */
  danglingRefs: number;
}

/** build turns generated claims into a graphology DirectedGraph. */
export function build(archive: MockArchive, opts: BuildOptions = {}): BuildResult {
  const attrs = opts.attrs ?? 'lean';
  const t0 = performance.now();
  const graph = new DirectedGraph({ allowSelfLoops: false });

  for (const claim of archive.claims) {
    const cls = classOf(claim.type);
    if (attrs === 'lean') {
      graph.addNode(claim.id, {
        x: 0,
        y: 0,
        size: CLASS_SIZE[cls] ?? 2,
        color: CLASS_COLOR[cls] ?? '#999999',
      });
    } else {
      graph.addNode(claim.id, {
        x: 0,
        y: 0,
        size: CLASS_SIZE[cls] ?? 2,
        color: CLASS_COLOR[cls] ?? '#999999',
        label: claim.label,
        cls,
        type: claim.type,
        createdAt: claim.created_at,
        contentSize: claim.content_size,
        encoding: claim.encoding,
      });
    }
  }

  let mergedEdges = 0;
  let danglingRefs = 0;
  const before = () => graph.size;

  const drop = opts.dropEdgeTypes ?? [];
  for (const claim of archive.claims) {
    for (const edge of claim.edges) {
      if (drop.some((prefix) => edge.type.startsWith(prefix))) continue;
      if (!graph.hasNode(edge.reference)) {
        danglingRefs++;
        continue;
      }
      const size = before();
      graph.mergeDirectedEdge(claim.id, edge.reference, { type: edge.type });
      if (graph.size === size) mergedEdges++;
    }
  }

  return { graph, buildMs: performance.now() - t0, mergedEdges, danglingRefs };
}

export interface DegreeStats {
  max: number;
  mean: number;
  p50: number;
  p99: number;
  /** Nodes with degree ≥ 100 — the hubs that dominate force layout cost. */
  hubs: number;
}

/** degreeStats summarises the degree distribution that layout has to cope with. */
export function degreeStats(graph: DirectedGraph): DegreeStats {
  const degrees = new Int32Array(graph.order);
  let i = 0;
  let sum = 0;
  let max = 0;
  let hubs = 0;
  graph.forEachNode((node) => {
    const d = graph.degree(node);
    degrees[i++] = d;
    sum += d;
    if (d > max) max = d;
    if (d >= 100) hubs++;
  });
  degrees.sort();
  const at = (q: number) => degrees[Math.min(degrees.length - 1, Math.floor(degrees.length * q))];
  return { max, mean: sum / (graph.order || 1), p50: at(0.5), p99: at(0.99), hubs };
}

/** sizeByDegree scales node radius by degree so hubs read as hubs. */
export function sizeByDegree(graph: DirectedGraph, minSize = 1.5, maxSize = 14): void {
  let max = 1;
  graph.forEachNode((node) => {
    const d = graph.degree(node);
    if (d > max) max = d;
  });
  const logMax = Math.log1p(max);
  graph.updateEachNodeAttributes((node, attr) => {
    const t = Math.log1p(graph.degree(node)) / logMax;
    attr.size = minSize + t * (maxSize - minSize);
    return attr;
  });
}
