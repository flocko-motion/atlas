/**
 * package: render / tests
 * type:    test
 * job:     pin that an idle sample window reports null rather than a truthful, misleading 0
 * limits:  headless; frame-stats.ts touches no Sigma and no DOM
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';

import { frameStats } from './frame-stats.ts';

test('an idle window (no frames drawn) reports fps and frameMs as null, not 0', () => {
  const { fps, frameMs, stallMs } = frameStats(0, 500, 0);
  assert.equal(fps, null, 'an idle canvas reported a numeric fps');
  assert.equal(frameMs, null, 'an idle canvas reported a numeric frame cost');
  assert.equal(stallMs, null, 'a window with no gap reported one anyway');
});

test('a drawing window reports the measured fps, frame cost and stall', () => {
  const { fps, frameMs, stallMs } = frameStats(30, 500, 42);
  assert.equal(fps, 60, `30 frames in 500 ms should read 60 fps, got ${fps}`);
  assert.equal(frameMs, 500 / 30);
  assert.equal(stallMs, 42);
});

test('a stalled thread with drawn frames still reports its gap', () => {
  const { stallMs } = frameStats(5, 500, 480);
  assert.equal(stallMs, 480, 'a real stall was not reported');
});
