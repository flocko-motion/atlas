/**
 * package: core / tests
 * type:    test
 * job:     pin the time axis — ordering, simultaneity, and the range it has to span
 * limits:  headless; formatting and ticks are the ruler's (-> ui)
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';

import { gapWidth, timeScale } from './timescale.ts';

const SECOND = 1000;
const MINUTE = 60 * SECOND;
const DAY = 86_400 * SECOND;
const YEAR = 365 * DAY;

// The whole reason for a logarithm: eleven orders of magnitude have to fit on one axis.
test('a gap spans milliseconds to centuries within a bounded distance', () => {
  const widths = [1, SECOND, MINUTE, DAY, YEAR, 100 * YEAR].map((ms) => gapWidth(ms));
  for (let i = 1; i < widths.length; i++) {
    assert.ok(widths[i] > widths[i - 1], `a longer gap must be wider: ${widths}`);
  }
  const range = widths[widths.length - 1] / widths[0];
  assert.ok(range < 100, `a century is ${range.toFixed(0)}× a millisecond, want a bounded range`);
  // And a century stays distinguishable from a year, which a hard cap would lose.
  assert.ok(gapWidth(100 * YEAR) > gapWidth(YEAR) * 1.1);
});

// Simultaneity is the one thing a stack is allowed to mean.
test('only identical instants coincide', () => {
  assert.equal(gapWidth(0), 0);
  const scale = timeScale([1000, 1000, 1000]);
  assert.equal(scale.width, 0);
  assert.equal(scale.instants, 1);

  // A gap that exists is at least a node wide, however small it is.
  assert.ok(gapWidth(0.1) >= 8, 'a tenth of a millisecond collapsed to nothing');
  assert.ok(gapWidth(1) >= 8);
});

test('the axis is monotonic and spans its instants', () => {
  const times = [0, SECOND, SECOND + 1, 10 * YEAR, 10 * YEAR + MINUTE];
  const scale = timeScale(times);
  const xs = times.map((t) => scale.toX(t));
  for (let i = 1; i < xs.length; i++) {
    assert.ok(xs[i] > xs[i - 1], `x must climb with time: ${xs}`);
  }
  assert.equal(xs[0], 0);
  assert.equal(xs[xs.length - 1], scale.width);
  assert.equal(scale.from, times[0]);
  assert.equal(scale.to, times[times.length - 1]);
});

// The ruler reads dates out of the axis, so the inverse has to land back on the instant.
test('atX inverts toX', () => {
  const times = [0, 5 * SECOND, 5 * SECOND + 2, 3 * YEAR];
  const scale = timeScale(times);
  for (const at of times) {
    const back = scale.atX(scale.toX(at));
    assert.ok(Math.abs(back - at) < 1, `round trip of ${at} gave ${back}`);
  }
  // Outside the axis, both ends clamp rather than extrapolating.
  assert.equal(scale.atX(-10), scale.from);
  assert.equal(scale.atX(scale.width + 10), scale.to);
  assert.equal(scale.toX(-1), 0);
  assert.equal(scale.toX(4 * YEAR), scale.width);
});

// A burst inside a long span still has to be legible, which is what the floor buys.
test('a burst inside a decade is not crushed to a point', () => {
  const burst = Array.from({ length: 20 }, (_, i) => 10 * YEAR + i);
  const scale = timeScale([0, ...burst]);
  const first = scale.toX(burst[0]);
  const last = scale.toX(burst[burst.length - 1]);
  assert.ok(last - first >= 19 * 8, `the burst spans ${(last - first).toFixed(0)}, want a node per claim`);
});

test('an empty archive has a point for an axis', () => {
  const scale = timeScale([]);
  assert.equal(scale.width, 0);
  assert.equal(scale.toX(12345), 0);
  assert.equal(scale.instants, 0);
});

// The cursor snaps to a claim, so the axis has to answer which claim is nearest.
test('nearestInstant snaps to an instant that carries claims', () => {
  const times = [0, 10 * SECOND, 10 * SECOND + 5, 3 * YEAR];
  const scale = timeScale(times);

  for (const at of times) {
    assert.equal(scale.nearestInstant(at), at, 'an instant should snap to itself');
  }
  // Nearer the earlier one, nearer the later one, and the tie goes to the earlier.
  assert.equal(scale.nearestInstant(4 * SECOND), 0);
  assert.equal(scale.nearestInstant(8 * SECOND), 10 * SECOND);
  assert.equal(scale.nearestInstant(5 * SECOND), 0);
  // Beyond either end it clamps rather than inventing an instant.
  assert.equal(scale.nearestInstant(-1000), 0);
  assert.equal(scale.nearestInstant(99 * YEAR), 3 * YEAR);
  // Snapping is in time, so a compressed gap changes nothing about the answer.
  assert.equal(scale.nearestInstant(10 * SECOND + 1), 10 * SECOND);
  assert.equal(scale.nearestInstant(10 * SECOND + 4), 10 * SECOND + 5);
});

// Stepping has to move, which is what distinguishes it from snapping to the nearest.
test('stepInstant moves to the neighbouring claim', () => {
  const times = [0, 10 * SECOND, 10 * SECOND + 5, 3 * YEAR];
  const scale = timeScale(times);

  assert.equal(scale.stepInstant(0, 1), 10 * SECOND);
  assert.equal(scale.stepInstant(10 * SECOND, 1), 10 * SECOND + 5);
  assert.equal(scale.stepInstant(10 * SECOND + 5, -1), 10 * SECOND);
  assert.equal(scale.stepInstant(3 * YEAR, -1), 10 * SECOND + 5);

  // From between two instants it lands on the one ahead, or the one behind.
  assert.equal(scale.stepInstant(5 * SECOND, 1), 10 * SECOND);
  assert.equal(scale.stepInstant(5 * SECOND, -1), 0);

  // Standing on an instant, a step moves — except at the end it is being asked to pass, where
  // holding still is the only honest answer.
  for (const at of times) {
    if (at !== scale.to) assert.notEqual(scale.stepInstant(at, 1), at, `forward from ${at} stood still`);
    if (at !== scale.from) assert.notEqual(scale.stepInstant(at, -1), at, `back from ${at} stood still`);
  }
  assert.equal(scale.stepInstant(scale.from, -1), scale.from, 'stepping before the first claim invented one');
  assert.equal(scale.stepInstant(scale.to, 1), scale.to, 'stepping past the last claim invented one');

  // Either end holds rather than running off.
  assert.equal(scale.stepInstant(3 * YEAR + 1, 1), 3 * YEAR);
  assert.equal(scale.stepInstant(-1, -1), 0);
});
