/**
 * package: core / tests
 * type:    test
 * job:     pin that loading while a CBOR tab is active reuses the existing graph tab
 * limits:  headless; store and session actions alone, no pane and no canvas
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';

import { activeView, useExplorer } from './store.ts';
import { load, openClaimCbor } from './session.ts';
import { useConnections } from './connections.ts';
import { clear } from './graph/universe.ts';
import { forgetMembers } from './graph/members.ts';

/** smallMock points the built-in generator at an archive a test can afford. */
function smallMock() {
  const mock = useConnections.getState().connections.find((c) => c.kind === 'mock')!;
  useConnections.getState().updateConnection(mock.id, {
    mock: { claims: 200, seed: 3, claimsPerContribution: 10 },
  });
  useConnections.getState().activate(mock.id);
}

function reset() {
  clear();
  forgetMembers();
  useExplorer.setState({
    tabs: [],
    activeTabId: null,
    notice: null,
    scopes: { state: 'unknown', scopes: [], selected: null, error: null },
  });
}

test('loading with a CBOR tab active brings the existing graph tab forward, not a fresh one', async () => {
  reset();
  smallMock();

  await load({ limit: 50 });
  const afterFirstLoad = useExplorer.getState().tabs.filter((t) => t.kind === 'graph');
  assert.equal(afterFirstLoad.length, 1, 'the first load did not settle on exactly one graph tab');
  const graphTabId = afterFirstLoad[0].id;

  openClaimCbor('some-claim-id');
  assert.equal(activeView(useExplorer.getState()), null, 'a CBOR tab read as a graph view');

  await load({ limit: 50 });
  const afterSecondLoad = useExplorer.getState().tabs.filter((t) => t.kind === 'graph');
  assert.equal(afterSecondLoad.length, 1, 'a load with a CBOR tab active made a second graph tab');
  assert.equal(afterSecondLoad[0].id, graphTabId, 'a load with a CBOR tab active replaced the graph tab rather than reusing it');

  openClaimCbor('another-claim-id');
  await load({ limit: 50 });
  const afterThirdLoad = useExplorer.getState().tabs.filter((t) => t.kind === 'graph');
  assert.equal(afterThirdLoad.length, 1, 'repeated loads behind a CBOR tab kept piling up graph tabs');
});
