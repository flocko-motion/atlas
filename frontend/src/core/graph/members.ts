/**
 * package: core / graph
 * type:    data
 * job:     hold the claim ids a scope contains, as the source reported them
 * limits:  storage only; which ids those are is the query engine's answer (-> ranke-go)
 *
 * Membership is not computed here. A scoped query with `output.detail: id` returns the
 * identities in that scope, and this holds the answer so the renderer can intersect it
 * with what is cached. Module scope, like the graph itself: an id set of a large branch is
 * graph data, and graph data never enters React state.
 */

import type { ClaimId } from '../mock/model.ts';

const members = new Map<string, Set<ClaimId>>();

/** setMembers records the ids a scope reported, replacing any earlier answer. */
export function setMembers(scope: string, ids: Iterable<ClaimId>): Set<ClaimId> {
  const set = new Set(ids);
  members.set(scope, set);
  return set;
}

/** membersOf returns a scope's reported ids, or null when it has not been asked. */
export function membersOf(scope: string): Set<ClaimId> | null {
  return members.get(scope) ?? null;
}

/** forgetMembers drops every answer — a session reset, or a source change. */
export function forgetMembers(): void {
  members.clear();
}
