/**
 * package: core / tests
 * type:    test
 * job:     pin the trail a reader leaves — that back returns, forward returns to where back came
 *          from, and asking about something new forks the trail rather than branching it
 * limits:  headless; the store alone, no pane and no canvas
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';

import { useExplorer } from './store.ts';

/** at is what the store currently answers about — the pair a visit records. */
function at() {
  const { selected, selectedEdge } = useExplorer.getState().selection;
  return { node: selected, edge: selectedEdge };
}

/** fresh starts a reader with no trail behind them. */
function fresh() {
  useExplorer.setState({
    selection: { selected: null, selectedEdge: null, hovered: null },
    history: { visits: [], at: -1 },
  });
  return useExplorer.getState();
}

test('back returns to what was being looked at, and forward returns from it', () => {
  const store = fresh();
  store.select('a');
  store.select('b');
  store.selectEdge('e1');

  store.stepHistory(-1);
  assert.deepEqual(at(), { node: 'b', edge: null });
  store.stepHistory(-1);
  assert.deepEqual(at(), { node: 'a', edge: null });
  store.stepHistory(1);
  assert.deepEqual(at(), { node: 'b', edge: null });
  store.stepHistory(1);
  assert.deepEqual(at(), { node: null, edge: 'e1' }, 'an edge is a place too');
});

test('a step past either end of the trail changes nothing', () => {
  const store = fresh();
  store.select('a');
  store.stepHistory(-1);
  assert.deepEqual(at(), { node: 'a', edge: null }, 'there was nowhere behind it');
  store.stepHistory(1);
  assert.deepEqual(at(), { node: 'a', edge: null }, 'nor anywhere ahead');
});

test('asking about something new from part-way back drops what was ahead', () => {
  const store = fresh();
  store.select('a');
  store.select('b');
  store.select('c');
  store.stepHistory(-1);
  store.stepHistory(-1);
  assert.deepEqual(at(), { node: 'a', edge: null });

  store.select('d');
  assert.deepEqual(at(), { node: 'd', edge: null });
  assert.deepEqual(
    useExplorer.getState().history,
    { visits: [{ node: 'a', edge: null }, { node: 'd', edge: null }], at: 1 },
    'b and c were ahead of where the reader turned off',
  );
});

test('asking twice about the same claim leaves one step, not two', () => {
  const store = fresh();
  store.select('a');
  store.select('a');
  assert.equal(useExplorer.getState().history.visits.length, 1);
});
