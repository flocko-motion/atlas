/**
 * The query — what to read from the active source, as opposed to which source it is.
 *
 * The split matters: a *connection* identifies an archive (a URL and credentials, or
 * a generator's seed and granularity), while a *query* is a request against it. The
 * same query run against two connections is a meaningful thing to do; a query that
 * carried its own archive identity would not be.
 *
 * Today it holds a result limit and an optional class filter, which is all the mock
 * source can honour. It is shaped to grow into the REST contract's query — `select`
 * (branch, root claim, traversal), `where`, `order`, `limit` — without the UI or the
 * store changing shape around it.
 */

import { create } from 'zustand';

/** Claim classes a query may restrict itself to; empty means every class. */
export const CLAIM_CLASSES = ['source', 'derivation', 'entity', 'relation', 'contribution'];

export interface Query {
  /** Branch to read — `select.branch` in the contract. Ignored by mock sources. */
  branch: string;
  /** Cap on claims returned — `limit.results`. */
  limit: number;
  /** Restrict to these classes, empty for all. Applied as the view's selection. */
  classes: string[];
}

export const DEFAULT_QUERY: Query = { branch: 'main', limit: 100000, classes: [] };

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
