/**
 * package: core / scope
 * type:    data
 * job:     name the scopes a view can sit in
 * limits:  vocabulary only; membership is computed in core/graph (-> core/graph/closure)
 *
 * A scope is browsable exactly when it has a head, because a view predicate is a closure
 * test and a closure needs a root. That single rule decides the whole set: each named
 * branch has a head, the archive has the branch-table head, and `$universe` has none —
 * which is why RankeQL refuses to read there without an explicit head. So `$universe` is
 * absent by the rule rather than by a special case.
 */

import type { ClaimId } from './mock/model.ts';

/** The reserved name of the archive-wide scope, as grants and RankeQL spell it. */
export const ARCHIVE_SCOPE = '$archive';

/** A scope a view can be confined to: a name, and the head whose closure it is. */
export interface Scope {
  /** `$archive`, or a branch name. */
  name: string;
  /** The claim id this scope's closure is rooted at. */
  head: ClaimId;
}

/** isArchive reports whether a scope is the archive-wide one rather than a branch. */
export function isArchive(scope: Scope): boolean {
  return scope.name === ARCHIVE_SCOPE;
}

/** scopeLabel is how a scope reads in the picker. */
export function scopeLabel(scope: Scope): string {
  return isArchive(scope) ? 'whole archive' : scope.name;
}

/** shortHead abbreviates a head id for a label, where the full id would not fit. */
export function shortHead(head: ClaimId): string {
  return head.length > 12 ? `${head.slice(0, 12)}…` : head;
}

/** One entry of the scope picker. The empty value lifts the confinement. */
export interface ScopeOption {
  /** Scope name, or '' for everything loaded. */
  value: string;
  label: string;
  selected: boolean;
}

/**
 * scopeOptions is what the picker offers: everything loaded, then each scope with the head
 * its closure is rooted at. Composed here rather than in the view, so what the picker
 * offers is decided where the rule lives and can be tested without a browser.
 */
export function scopeOptions(scopes: Scope[], selected: Scope | null): ScopeOption[] {
  return [
    { value: '', label: 'everything loaded', selected: selected === null },
    ...scopes.map((scope) => ({
      value: scope.name,
      label: `${scopeLabel(scope)} · ${shortHead(scope.head)}`,
      selected: selected?.name === scope.name,
    })),
  ];
}
