/**
 * package: ui / tests
 * type:    test
 * job:     pin how many of the axis's precomputed candidates a given zoom keeps, and that they
 *          never crowd each other
 * limits:  render-time selection only; the candidates themselves — which instant, at what
 *          granularity — are core's own behaviour (-> core/layout/timescale.test.ts)
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';

import { timeTicks, wordFor } from './ticks.ts';

const SECOND = 1000;
const MINUTE = 60 * SECOND;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

/** linear scatters instants evenly across a 1000 px axis, one candidate per instant, all
 * worded the same unit — the simple case for exercising the stride/spacing logic alone. */
function linearPositions(instants: number[], unit: 'day' | 'year' = 'day') {
  const from = instants[0];
  const to = instants[instants.length - 1];
  const span = to - from || 1;
  const xOf = (at: number) => ((at - from) / span) * 1000;
  const positions = instants.map((at) => ({ axisX: xOf(at), at, unit }));
  return { positions, xOf, from, to };
}

test('no two labels come closer than the minimum gap', () => {
  const instants = Array.from({ length: 40 }, (_, i) => Date.parse('2024-01-01T00:00:00Z') + i * DAY);
  const { positions, xOf, from, to } = linearPositions(instants);
  const ticks = timeTicks({ from, to, xOf, minGap: 90, positions });
  for (let i = 1; i < ticks.length; i++) {
    assert.ok(
      ticks[i].x - ticks[i - 1].x >= 90 - 1e-6,
      `${ticks[i - 1].label}→${ticks[i].label} are ${(ticks[i].x - ticks[i - 1].x).toFixed(0)} apart`,
    );
  }
});

test('a wider minimum gap keeps strictly fewer labels, never more', () => {
  const instants = Array.from({ length: 200 }, (_, i) => Date.parse('2024-01-01T00:00:00Z') + i * HOUR);
  const { positions, xOf, from, to } = linearPositions(instants);
  const loose = timeTicks({ from, to, xOf, minGap: 20, positions });
  const tight = timeTicks({ from, to, xOf, minGap: 200, positions });
  assert.ok(loose.length > 1, 'expected more than one label at a loose gap');
  assert.ok(tight.length < loose.length, `tight (${tight.length}) did not thin below loose (${loose.length})`);
});

// A candidate array is built once over the whole archive; a reader can still be looking at
// only a slice of it. The binary search into [from, to] must not leak a label from outside.
test('only candidates inside the visible range are considered', () => {
  const instants = Array.from({ length: 10 }, (_, i) => Date.parse('2024-01-01T00:00:00Z') + i * DAY);
  const { positions, xOf } = linearPositions(instants);
  const midFrom = instants[3];
  const midTo = instants[6];
  const ticks = timeTicks({ from: midFrom, to: midTo, xOf, minGap: 1, positions });
  const [xLo, xHi] = [xOf(midFrom), xOf(midTo)];
  for (const tick of ticks) {
    assert.ok(tick.x >= xLo - 1e-6 && tick.x <= xHi + 1e-6, `a tick at x=${tick.x} fell outside [${xLo}, ${xHi}]`);
  }
});

// A narrow, zoomed-in view can hold exactly one real candidate — a stride aligned to a fixed
// global origin (so the picture does not jitter as a reader zooms) can step clean over the
// only one there, and that must not read as "nothing to show here".
test('a visible range holding exactly one candidate still shows it', () => {
  const instants = Array.from({ length: 500 }, (_, i) => Date.parse('2024-01-01T00:00:00Z') + i * MINUTE);
  const { positions, xOf } = linearPositions(instants);
  // A minGap demanding a stride far larger than the whole visible slice holds.
  const midFrom = instants[250];
  const midTo = instants[251];
  const ticks = timeTicks({ from: midFrom, to: midTo, xOf, minGap: 5000, positions });
  assert.ok(ticks.length >= 1, 'a visible candidate went unlabelled');
});

test('no candidates in the visible range falls back to labelling its own ends', () => {
  const instants = [Date.parse('2024-01-01T00:00:00Z'), Date.parse('2025-01-01T00:00:00Z')];
  const { positions } = linearPositions(instants);
  // A window strictly between the two candidates, touching neither, and not a whole number of
  // days apart — so the two endpoint labels (worded to the hour) do not coincidentally match.
  // Its own xOf, scaled to fill 1000 px with just this window — a reader zoomed in this far
  // sees the window fill the screen, not compressed to the sliver the full-archive scale
  // (linearPositions' own xOf, spanning the whole year) would draw it at.
  const from = instants[0] + 10 * DAY + 3 * HOUR;
  const to = instants[0] + 20 * DAY + 7 * HOUR;
  const xOf = (at: number) => ((at - from) / (to - from)) * 1000;
  const ticks = timeTicks({ from, to, xOf, minGap: 60, positions });
  assert.ok(ticks.length > 0, 'an empty-of-candidates window produced no labels at all');
  assert.notEqual(ticks[0]?.label, ticks[ticks.length - 1]?.label, 'both ends read the same');
});

test('empty positions still falls back to endpoint labels', () => {
  const ticks = timeTicks({ from: 0, to: 10 * DAY, xOf: (at) => at / DAY, minGap: 60, positions: [] });
  assert.ok(ticks.length > 0, 'no candidates at all produced no labels');
});

test('an instant span yields no crowd of labels', () => {
  const at = Date.parse('2024-03-04T05:06:07Z');
  const ticks = timeTicks({ from: at, to: at, xOf: () => 0, minGap: 60, positions: [] });
  assert.ok(ticks.length <= 2, `a zero span produced ${ticks.length} labels`);
  assert.deepEqual(timeTicks({ from: 10, to: 0, xOf: () => 0, minGap: 60, positions: [] }), []);
});

// Two units competing for the same spot: the coarser one keeps its place regardless of which
// index order they arrive in, and `select`'s tie-break is unit rank, not array order.
test('a coarser unit outranks a finer one at the same position', () => {
  const at = Date.parse('2024-06-15T00:00:00Z');
  const positions = [
    { axisX: 500, at, unit: 'day' as const },
    { axisX: 500, at, unit: 'year' as const },
  ];
  const ticks = timeTicks({ from: at, to: at, xOf: () => 500, minGap: 60, positions });
  assert.equal(ticks.length, 1);
  assert.equal(ticks[0].unit, 'year');
});

test('a boundary is worded by the part that distinguishes it', () => {
  const at = Date.parse('2024-03-04T05:06:07.008Z');
  assert.equal(wordFor(at, 'year'), '2024');
  assert.equal(wordFor(at, 'month'), 'Mar');
  assert.equal(wordFor(at, 'day'), 'Mar 4');
  assert.equal(wordFor(at, 'hour'), '05:06');
  assert.equal(wordFor(at, 'second'), '05:06:07');
});
