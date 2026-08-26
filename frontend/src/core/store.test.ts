/**
 * package: core / tests
 * type:    test
 * job:     pin the invariant the tab-strip refactor rests on — a CBOR tab is never mistaken
 *          for a graph view by the selector everything else reads through
 * limits:  headless; the store alone, no session, no pane, no canvas
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';

import { activeView, defaultView, useExplorer } from './store.ts';
import type { CborTabState } from './store.ts';

function reset() {
  useExplorer.setState({ tabs: [], activeTabId: null });
}

const cborTab: CborTabState = { kind: 'cbor', id: 'cbor-1', label: 'claim-a', claimId: 'claim-a', scope: null };

test('activeView reads null while a CBOR tab is active, not the last graph view', () => {
  reset();
  const view = defaultView('v1', 'v1');
  useExplorer.getState().addTab(view);
  useExplorer.getState().addTab(cborTab);
  useExplorer.getState().activateTab('cbor-1');

  const state = useExplorer.getState();
  assert.equal(state.activeTabId, 'cbor-1');
  assert.equal(activeView(state), null, 'activeView returned a graph view while a CBOR tab was active');
});

// render/renderer.ts, render/camera.ts and both time widgets all read the active tab only
// through activeView — nothing else derives it (store.ts's own note on the selector). A view
// settings patch reaching a CBOR tab regardless would be exactly the class of bug that null
// return exists to prevent, so patchView's own kind guard is pinned directly here too.
test('patchView leaves a CBOR tab untouched — the graph-only guard the reducers rely on', () => {
  reset();
  useExplorer.getState().addTab(cborTab);
  useExplorer.getState().patchView('cbor-1', { layout: 'circular' });

  const tab = useExplorer.getState().tabs.find((t) => t.id === 'cbor-1');
  assert.deepEqual(tab, cborTab, 'patchView changed a CBOR tab addressed by a graph view id');
});
