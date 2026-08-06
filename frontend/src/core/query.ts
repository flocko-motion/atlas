/**
 * package: core / query
 * type:    data
 * job:     hold what to read from the active source
 * limits:  the form a reader fills in, not the query it becomes (-> core/data/source)
 *
 * A connection identifies an archive; this says what to read from it, so the same settings
 * against two connections are meaningful. It is deliberately *not* a RankeQL query: that type
 * is the library's, generated from the released schema, and `core/data/source` builds one from
 * these settings at request time. `classes` is why the two differ — a class selection is what
 * the view draws, not what the read asks for.
 */

import { create } from 'zustand';

export interface QueryForm {
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

export const DEFAULT_QUERY: QueryForm = { branch: null, limit: 100000, classes: [] };

interface QueryState {
  query: QueryForm;
  patchQuery: (patch: Partial<QueryForm>) => void;
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

/** describeQuery summarises the settings for a view's tab label. */
export function describeQuery(query: QueryForm): string {
  const scope = query.classes.length === 0 ? 'all classes' : query.classes.join('+');
  return `${query.limit >= 1000 ? `${Math.round(query.limit / 1000)}k` : query.limit} · ${scope}`;
}
