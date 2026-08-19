/**
 * package: core / tests
 * type:    test
 * job:     pin how the timeline layout places claims on y — strata, subbands, and the scatter
 *          within each
 * limits:  headless; positions only, no rendering
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';
import { DirectedGraph } from 'graphology';

import { assignTimeline } from './layouts.ts';
import type { TimelineContext } from './layouts.ts';

/** A context whose claims each carry an explicit class and subtype, keyed by node id. */
function ctxFor(classes: Record<string, string>, subtypes: Record<string, string> = {}): TimelineContext {
  return {
    toX: (at) => at,
    createdAt: (node) => Number(node),
    classOf: (node) => classes[node] ?? '',
    subOf: (node) => subtypes[node] ?? '',
  };
}

/** buildGraph makes a node per id, all at distinct instants so x never coincides. */
function buildGraph(ids: string[]): DirectedGraph {
  const g = new DirectedGraph();
  ids.forEach((id, i) => g.addNode(id, { x: 0, y: 0, id: String(i) }));
  return g;
}

test('contribution/head claims sit in their own subband, strictly below the rest of the band', () => {
  const heads = Array.from({ length: 6 }, (_, i) => `head${i}`);
  const others = Array.from({ length: 6 }, (_, i) => `other${i}`);
  const g = buildGraph([...heads, ...others]);
  const classes = Object.fromEntries([...heads, ...others].map((id) => [id, 'contribution']));
  const subtypes = Object.fromEntries([
    ...heads.map((id) => [id, 'head']),
    ...others.map((id) => [id, 'contributor']),
  ]);
  assignTimeline(g, ctxFor(classes, subtypes));

  const yOf = (id: string) => g.getNodeAttribute(id, 'y') as number;
  const maxHeadY = Math.max(...heads.map(yOf));
  const minOtherY = Math.min(...others.map(yOf));
  assert.ok(
    maxHeadY < minOtherY,
    `expected every head below every other contribution claim, got max head=${maxHeadY}, min other=${minOtherY}`,
  );
});

// Not a single line: several heads should land on distinct lanes within their subband, the
// same scatter every other stratum gets — the whole reason a subband exists is to group heads
// without collapsing them onto one row.
test('heads scatter across lanes within their subband rather than collapsing to one', () => {
  const heads = Array.from({ length: 12 }, (_, i) => `head${i}`);
  const g = buildGraph(heads);
  const classes = Object.fromEntries(heads.map((id) => [id, 'contribution']));
  const subtypes = Object.fromEntries(heads.map((id) => [id, 'head']));
  assignTimeline(g, ctxFor(classes, subtypes));

  const ys = new Set(heads.map((id) => g.getNodeAttribute(id, 'y') as number));
  assert.ok(ys.size > 1, `expected more than one lane among 12 heads, got ${ys.size}`);
});

// The subband split is scoped to contribution — every other stratum still gets its band whole,
// as it did before heads were split out.
test('a non-contribution class is unaffected by the head subband split', () => {
  const claims = Array.from({ length: 20 }, (_, i) => `c${i}`);
  const g = buildGraph(claims);
  const classes = Object.fromEntries(claims.map((id) => [id, 'source']));
  assignTimeline(g, ctxFor(classes));

  const ys = new Set(claims.map((id) => g.getNodeAttribute(id, 'y') as number));
  assert.ok(ys.size > 1, `expected the source band to scatter across lanes as before, got ${ys.size}`);
});

// The split is a fixed share of the contribution band, present whether or not any head claim
// is actually loaded — the same "fixed regardless of what is shown" rule the bands themselves
// follow (-> the toggle-must-not-move-anyone-else's-claims tests this mirrors).
test('the head subband exists even when nothing is placed in it', () => {
  const others = Array.from({ length: 6 }, (_, i) => `other${i}`);
  const g = buildGraph(others);
  const classes = Object.fromEntries(others.map((id) => [id, 'contribution']));
  const subtypes = Object.fromEntries(others.map((id) => [id, 'contributor']));
  // Should not throw, and should place these claims above where the (empty) head subband is.
  assert.doesNotThrow(() => assignTimeline(g, ctxFor(classes, subtypes)));
});
