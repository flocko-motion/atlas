/**
 * package: core / tests
 * type:    test
 * job:     pin what a stretch leaves behind — that every graph a reader may be shown next is
 *          laid out to the stretch the view reports
 * limits:  headless; no DOM, no Sigma, no camera
 *
 * The bound the camera is held to is measured from the view's stretch against the pinned extent
 * (-> render/hold). So a graph laid out to a different stretch than the view reports is a graph
 * the bound is measured wrongly for — the picture can then travel far out of the viewport before
 * anything stops it, which is what a lens used to leave behind.
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';
import type { DirectedGraph } from 'graphology';

import { generate } from './mock/generate.ts';
import { clear, graph, mergeClaims } from './graph/universe.ts';
import { lensOf } from './graph/lens.ts';
import { assignTimeline } from './layout/layouts.ts';
import { settleUnion, stretchY, timelineContext } from './timeline.ts';
import { defaultView, useExplorer } from './store.ts';

/** ySpan is how tall the drawn claims are, in graph units — what the extent is measured against. */
function ySpan(g: DirectedGraph): number {
  let lo = Infinity;
  let hi = -Infinity;
  g.forEachNode((_node, attr) => {
    lo = Math.min(lo, Number(attr.y));
    hi = Math.max(hi, Number(attr.y));
  });
  return hi - lo;
}

/** archive lays a generated archive out on the timeline, as a load does. */
function archive(): number {
  clear();
  useExplorer.setState({ tabs: [], activeTabId: null });
  const view = defaultView('v1', 'v1');
  useExplorer.getState().addTab(view);
  mergeClaims(generate(400, 7).claims, 40);
  assignTimeline(graph(), timelineContext());
  return ySpan(graph());
}

test('a stretch over a lens leaves the union laid out to the same stretch', () => {
  const rest = archive();
  assert.ok(rest > 0, 'the timeline laid nothing out');

  // A lens copies the union's positions, so what it is stretched to is what the reader sees.
  const lens = lensOf(graph(), { x0: -Infinity, x1: Infinity }).graph;
  const { applied } = stretchY(4, lens);
  assert.equal(applied, 4);
  assert.ok(Math.abs(ySpan(lens) - rest * 4) < 1e-6, 'the lens was not stretched');

  // The union is what the reader is shown the moment the lens is dropped, and every measurement
  // of the picture is taken against the view's stretch — so it may not stay at the old one.
  settleUnion();
  assert.ok(
    Math.abs(ySpan(graph()) - rest * 4) < 1e-6,
    `the union is still at ${ySpan(graph())}, where the view reports ${rest * 4}`,
  );
});

test('a stretch over the union alone leaves nothing to settle', () => {
  archive();
  stretchY(2);
  assert.equal(settleUnion(), false, 'the union was laid out twice for one stretch');
});
