/**
 * package: core / tests
 * type:    test
 * job:     pin that a claim's CBOR opens one tab per claim, not one per click
 * limits:  headless; the store and session actions alone, no pane and no canvas
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';

import { openClaimCbor } from './session.ts';
import { useExplorer } from './store.ts';

function reset() {
  useExplorer.setState({ tabs: [], activeTabId: null });
}

test('opening the same claim twice brings the one tab forward rather than adding a second', () => {
  reset();
  openClaimCbor('claim-a');
  const afterFirst = useExplorer.getState();
  assert.equal(afterFirst.tabs.length, 1);
  const tabId = afterFirst.tabs[0].id;
  assert.equal(afterFirst.activeTabId, tabId);

  // A tab in between, then back to the claim already open — the earlier tab is found and
  // activated rather than shadowed by a newer one for the same claim.
  openClaimCbor('claim-b');
  assert.equal(useExplorer.getState().tabs.length, 2);
  openClaimCbor('claim-a');
  const afterReopen = useExplorer.getState();
  assert.equal(afterReopen.tabs.length, 2, 'the same claim opened a second tab');
  assert.equal(afterReopen.activeTabId, tabId, 'reopening did not bring the original tab forward');
});

test('each open tab carries only its own claim id and nothing a graph view would', () => {
  reset();
  openClaimCbor('claim-a');
  const tab = useExplorer.getState().tabs[0];
  assert.equal(tab.kind, 'cbor');
  if (tab.kind !== 'cbor') return;
  assert.equal(tab.claimId, 'claim-a');

  // A graph view's own fields — `scope` is the one field the two kinds share.
  const graphOnly = ['contributionRange', 'classes', 'layout', 'xStretch', 'yStretch', 'edges', 'edgesOnMove', 'labels', 'labelsOnMove', 'sizeByDegree'];
  for (const key of graphOnly) {
    assert.ok(!(key in tab), `a CBOR tab carried the graph-view field '${key}'`);
  }
});
