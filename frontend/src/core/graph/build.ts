/**
 * Claims → graphology. Headless: no renderer, no DOM.
 *
 * `addClaims` merges into an existing graph, because the explorer accumulates —
 * query after query folds into one union keyed by content address. `buildGraph`
 * is the one-shot form the benches use.
 */

import { DirectedGraph } from 'graphology';
import type { MockArchive, MockClaim } from '../mock/model.ts';
import { classOf } from '../mock/model.ts';
import { yieldToPaint } from '../scheduler.ts';

/** CLASS_COLOR maps a node class to its paint colour (Okabe–Ito, colour-safe). */
export const CLASS_COLOR: Record<string, string> = {
  source: '#0072b2',
  derivation: '#e69f00',
  entity: '#009e73',
  relation: '#cc79a7',
  contribution: '#7f7f7f',
};

/** CLASS_SIZE gives each class a base radius — hubs get scaled up by degree. */
export const CLASS_SIZE: Record<string, number> = {
  source: 2,
  derivation: 1.5,
  entity: 3,
  relation: 2,
  contribution: 4,
};

export interface BuildOptions {
  /** `full` also stores the claim metadata a detail pane needs. */
  attrs?: 'lean' | 'full';
  /**
   * Edge types to leave out, matched by prefix. Every claim carries a
   * `contribution/contributor` edge, so a contributor node's degree equals the
   * number of claims it signed — a star that dominates layout and the picture.
   */
  dropEdgeTypes?: string[];
}

export interface MergeStats {
  addedNodes: number;
  addedEdges: number;
  duplicateClaims: number;
  /** References to claims outside the delivered set — real reads have these. */
  danglingRefs: number;
  mergeMs: number;
}

/**
 * addClaims folds claims into a graph, skipping any already present. Node keys are
 * claim ids, so a claim delivered twice merges rather than duplicating.
 */
export function addClaims(
  graph: DirectedGraph,
  claims: MockClaim[],
  opts: BuildOptions = {},
): MergeStats {
  const t0 = performance.now();
  const nodes = addNodes(graph, claims, opts);
  const edges = addEdges(graph, claims, opts);
  return {
    addedNodes: nodes.addedNodes,
    duplicateClaims: nodes.duplicateClaims,
    addedEdges: edges.addedEdges,
    danglingRefs: edges.danglingRefs,
    mergeMs: performance.now() - t0,
  };
}

/** Claims merged between yields: small enough to keep a frame, large enough to be cheap. */
const MERGE_BATCH = 4000;

export interface ProgressReport {
  stage: string;
  done: number;
  total: number;
}

/**
 * addClaimsProgressively is `addClaims` with the loop broken into batches that yield
 * to the UI between them. The work is identical; only the scheduling differs — and
 * that is the whole difference between a progress indicator and an apparent hang.
 */
export async function addClaimsProgressively(
  graph: DirectedGraph,
  claims: MockClaim[],
  opts: BuildOptions = {},
  onProgress?: (report: ProgressReport) => void,
): Promise<MergeStats> {
  const t0 = performance.now();
  const totals: MergeStats = {
    addedNodes: 0,
    addedEdges: 0,
    duplicateClaims: 0,
    danglingRefs: 0,
    mergeMs: 0,
  };

  // Claims first, in batches; an edge can only be added once both ends exist.
  for (let from = 0; from < claims.length; from += MERGE_BATCH) {
    const stats = addNodes(graph, claims.slice(from, from + MERGE_BATCH), opts);
    totals.addedNodes += stats.addedNodes;
    totals.duplicateClaims += stats.duplicateClaims;
    const done = Math.min(from + MERGE_BATCH, claims.length);
    onProgress?.({ stage: 'merging claims', done, total: claims.length });
    if (done < claims.length) await yieldToPaint();
  }

  for (let from = 0; from < claims.length; from += MERGE_BATCH) {
    const stats = addEdges(graph, claims.slice(from, from + MERGE_BATCH), opts);
    totals.addedEdges += stats.addedEdges;
    totals.danglingRefs += stats.danglingRefs;
    const done = Math.min(from + MERGE_BATCH, claims.length);
    onProgress?.({ stage: 'merging edges', done, total: claims.length });
    if (done < claims.length) await yieldToPaint();
  }

  totals.mergeMs = performance.now() - t0;
  return totals;
}

/** addNodes merges the claims themselves, skipping any already present. */
function addNodes(
  graph: DirectedGraph,
  claims: MockClaim[],
  opts: BuildOptions,
): { addedNodes: number; duplicateClaims: number } {
  const attrs = opts.attrs ?? 'full';
  let addedNodes = 0;
  let duplicateClaims = 0;

  for (const claim of claims) {
    if (graph.hasNode(claim.id)) {
      duplicateClaims++;
      continue;
    }
    const cls = classOf(claim.type);
    const base = {
      x: 0,
      y: 0,
      size: CLASS_SIZE[cls] ?? 2,
      color: CLASS_COLOR[cls] ?? '#999999',
      // `contribution` is in even the lean profile: it is the layout's axis, not
      // metadata. `claimType`, never `type` — Sigma reads `type` as the name of
      // the rendering program to use.
      contribution: claim.contribution,
      cls,
    };
    graph.addNode(
      claim.id,
      attrs === 'lean'
        ? base
        : {
            ...base,
            label: claim.label,
            claimType: claim.type,
            createdAt: claim.created_at,
            contentSize: claim.content_size,
            encoding: claim.encoding,
          },
    );
    addedNodes++;
  }
  return { addedNodes, duplicateClaims };
}

/** addEdges merges each claim's typed references, once both ends are present. */
function addEdges(
  graph: DirectedGraph,
  claims: MockClaim[],
  opts: BuildOptions,
): { addedEdges: number; danglingRefs: number } {
  const drop = opts.dropEdgeTypes ?? [];
  let addedEdges = 0;
  let danglingRefs = 0;
  for (const claim of claims) {
    for (const edge of claim.edges) {
      if (drop.some((prefix) => edge.type.startsWith(prefix))) continue;
      if (!graph.hasNode(edge.reference)) {
        danglingRefs++;
        continue;
      }
      const before = graph.size;
      graph.mergeDirectedEdge(claim.id, edge.reference, { claimType: edge.type });
      if (graph.size > before) addedEdges++;
    }
  }
  return { addedEdges, danglingRefs };
}

export interface BuildResult {
  graph: DirectedGraph;
  buildMs: number;
  mergedEdges: number;
  danglingRefs: number;
}

/** buildGraph turns one archive into a fresh graph — the benches' entry point. */
export function buildGraph(archive: MockArchive, opts: BuildOptions = {}): BuildResult {
  const graph = new DirectedGraph({ allowSelfLoops: false });
  const stats = addClaims(graph, archive.claims, opts);
  return {
    graph,
    buildMs: stats.mergeMs,
    mergedEdges: stats.duplicateClaims,
    danglingRefs: stats.danglingRefs,
  };
}

export interface DegreeStats {
  max: number;
  mean: number;
  p50: number;
  p99: number;
  /** Nodes with degree ≥ 100 — the hubs that dominate force layout cost. */
  hubs: number;
}

/** degreeStats summarises the degree distribution a layout has to cope with. */
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
