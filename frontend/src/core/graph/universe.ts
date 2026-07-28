/**
 * The union graph — every claim loaded this session, in one graphology instance.
 *
 * It lives at module scope, deliberately outside any framework state: React holds
 * a handle to it and never the data. A claim's id is its content address, so a
 * claim reached by two different queries merges to one node with no
 * reconciliation, and the store's size is the union of what was fetched rather
 * than the sum.
 */

import { DirectedGraph } from 'graphology';
import type { MockClaim } from '../mock/model.ts';
import { CLASS_COLOR, CLASS_SIZE, addClaims, addClaimsProgressively } from './build.ts';
import type { ProgressReport } from './build.ts';

/** The single graphology instance the renderer reads from. */
const universe = new DirectedGraph({ allowSelfLoops: false });

/** Contribution counter across everything merged in, for view ranges. */
let contributions = 0;

export interface MergeResult {
  addedNodes: number;
  addedEdges: number;
  /** Claims already present, which is the accumulation working as intended. */
  duplicateClaims: number;
  danglingRefs: number;
  mergeMs: number;
}

/** graph exposes the union for the renderer and core operations to read. */
export function graph(): DirectedGraph {
  return universe;
}

/** totalContributions is the highest contribution index merged so far. */
export function totalContributions(): number {
  return contributions;
}

/**
 * mergeClaims folds a page of claims into the union — what a query response does,
 * whether it came from a generator or a server.
 */
export function mergeClaims(claims: MockClaim[], contributionCount: number): MergeResult {
  const result = addClaims(universe, claims);
  contributions = Math.max(contributions, contributionCount);
  return result;
}

/**
 * mergeClaimsProgressively is `mergeClaims` in batches that yield to the UI, so a
 * large page reports progress instead of freezing the tab.
 */
export async function mergeClaimsProgressively(
  claims: MockClaim[],
  contributionCount: number,
  onProgress?: (report: ProgressReport) => void,
): Promise<MergeResult> {
  const result = await addClaimsProgressively(universe, claims, {}, onProgress);
  contributions = Math.max(contributions, contributionCount);
  return result;
}

/** clear empties the union — a session reset, not something a query does. */
export function clear(): void {
  universe.clear();
  contributions = 0;
}

export { CLASS_COLOR, CLASS_SIZE };
