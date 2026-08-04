/**
 * package: core / query
 * type:    data
 * job:     hold what to read from the active source
 * limits:  the request, not the archive it runs against (-> core/connections)
 *
 * A connection identifies an archive; a query is a request against it, so the same query
 * against two connections is meaningful. Shaped to grow into the REST contract's
 * `select`/`where`/`order`/`limit`.
 */

import { create } from 'zustand';

/** Claim classes a query may restrict itself to; empty means every class. */
export const CLAIM_CLASSES = ['source', 'derivation', 'entity', 'relation', 'contribution'];

export interface Query {
  /**
   * Scope to read — `select.branch` in the contract — or null until one is known.
   *
   * There is deliberately no default. A branch name is a fact about the archive in front
   * of the explorer, so guessing one is wrong whenever it is wrong and silently so; every
   * name comes from the branch listing instead, which makes discovery precede any
   * scope-confined read.
   */
  branch: string | null;
  /** Cap on claims returned — `limit.results`. */
  limit: number;
  /** Restrict to these classes, empty for all. Applied as the view's selection. */
  classes: string[];
}

export const DEFAULT_QUERY: Query = { branch: null, limit: 100000, classes: [] };

interface QueryState {
  query: Query;
  patchQuery: (patch: Partial<Query>) => void;
  toggleClass: (cls: string) => void;
}

export const useQuery = create<QueryState>((set) => ({
  query: { ...DEFAULT_QUERY },
  patchQuery: (patch) => set((s) => ({ query: { ...s.query, ...patch } })),
  toggleClass: (cls) =>
    set((s) => ({
      query: {
        ...s.query,
        classes: s.query.classes.includes(cls)
          ? s.query.classes.filter((c) => c !== cls)
          : [...s.query.classes, cls],
      },
    })),
}));

/** describeQuery summarises a query for a view's tab label. */
export function describeQuery(query: Query): string {
  const scope = query.classes.length === 0 ? 'all classes' : query.classes.join('+');
  return `${query.limit >= 1000 ? `${Math.round(query.limit / 1000)}k` : query.limit} · ${scope}`;
}
