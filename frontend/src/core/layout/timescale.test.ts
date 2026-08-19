/**
 * package: core / tests
 * type:    test
 * job:     pin the time axis — ordering, simultaneity, and the range it has to span
 * limits:  headless; formatting and ticks are the ruler's (-> ui)
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';

import { gapWidth, granularityFor, timeScale } from './timescale.ts';

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

// A remote claim (a generator's bad timestamp, say) must stay fully on the axis — atX/toX
// still reach it — while the dense range a first look opens on trims past it.
test('a lone remote outlier is trimmed from the dense range but stays on the axis', () => {
  const outlier = 0;
  const cluster = Array.from({ length: 5 }, (_, i) => 50 * YEAR + i * MINUTE);
  const scale = timeScale([outlier, ...cluster]);

  assert.equal(scale.toX(outlier), 0, 'the outlier fell off the axis');
  assert.equal(scale.atX(0), outlier, 'and is no longer reachable back off it');
  assert.ok(
    scale.denseExtent.x0 > 0,
    `expected the dense range to start past the outlier, got x0=${scale.denseExtent.x0}`,
  );
  assert.equal(scale.denseExtent.x1, scale.width, 'the dense range should still reach the cluster end');
});

test('an archive with no lone outlier has a dense range spanning the whole axis', () => {
  const times = Array.from({ length: 10 }, (_, i) => i * DAY);
  const scale = timeScale(times);
  assert.equal(scale.denseExtent.x0, 0);
  assert.equal(scale.denseExtent.x1, scale.width);
});

// The outlier at the *end* of the axis exercises the other half of the trim — a widest gap
// sitting last in the window under consideration, which a previous version of denseRange
// looped on forever rather than shrinking past.
test('a lone remote outlier at the end trims from the right without hanging', { timeout: 2000 }, () => {
  const cluster = Array.from({ length: 5 }, (_, i) => i * MINUTE);
  const outlier = 50 * YEAR;
  const scale = timeScale([...cluster, outlier]);

  assert.equal(scale.denseExtent.x0, 0);
  assert.ok(
    scale.denseExtent.x1 < scale.width,
    `expected the dense range to stop short of the outlier, got x1=${scale.denseExtent.x1}`,
  );
});

// granularityFor is the rule a candidate's wording follows: the coarsest unit that still
// distinguishes it from a neighbour this far away.
test('granularityFor buckets a gap to the coarsest unit that still shows it', () => {
  assert.equal(granularityFor(45 * SECOND), 'second');
  // A year is 365.25 days internally (calendar leap years average out); comfortably past that,
  // not the test's own 365-day YEAR constant, which is a hair short of it.
  assert.equal(granularityFor(366 * DAY), 'year');
  assert.equal(granularityFor(DAY), 'day');
  assert.equal(granularityFor(0), 'ms');
  assert.equal(granularityFor(500), 'ms');
});

// tickPositions is the build-time walk a reader's "sweep left to right, grab a claim every so
// often" idea becomes: density should follow where the claims actually are, not the calendar.
test('tickPositions is denser where claims are dense, sparser across a silence', () => {
  const burstStart = 10 * YEAR;
  const burst = Array.from({ length: 200 }, (_, i) => burstStart + i * SECOND); // ~200 s of claims
  const after = burstStart + 5 * YEAR; // five years of silence, then one more claim
  const scale = timeScale([...burst, after]);

  const inBurst = scale.tickPositions.filter((p) => p.at >= burstStart && p.at < burstStart + 200 * SECOND);
  const inSilence = scale.tickPositions.filter((p) => p.at > burstStart + 200 * SECOND && p.at < after);
  assert.ok(inBurst.length > 1, `expected more than one candidate inside the burst, got ${inBurst.length}`);
  assert.ok(inSilence.length === 0, `expected no candidate in the silence, got ${inSilence.length}`);
});

// The true ends of the loaded data are never silently absent, even where the build threshold
// would otherwise have skipped straight past them.
test('tickPositions always keeps the first and last instant', () => {
  const times = Array.from({ length: 500 }, (_, i) => i * SECOND); // tightly packed, well under threshold
  const scale = timeScale(times);
  const ats = scale.tickPositions.map((p) => p.at);
  assert.equal(ats[0], times[0], 'the first instant was not kept');
  assert.equal(ats[ats.length - 1], times[times.length - 1], 'the last instant was not kept');
});

// The count tickPositions produces is bounded by the axis's own width divided by the build
// threshold, not by how many distinct instants fed it — a reference-view-full archive should
// hold roughly the same number of candidates whether it has hundreds of claims or hundreds of
// thousands, since a finer packing only makes the threshold skip more of them, not fewer.
test('tickPositions count does not grow with how many claims fed it', () => {
  const span = 10 * YEAR;
  const few = timeScale(Array.from({ length: 50 }, (_, i) => (i * span) / 49)).tickPositions.length;
  const many = timeScale(Array.from({ length: 50_000 }, (_, i) => (i * span) / 49_999)).tickPositions.length;
  assert.ok(many < few * 4, `50,000 claims produced ${many} candidates against ${few} for 50 — not bounded`);
});

// The one property the user's algorithm asks for by name: a candidate's granularity comes from
// its distance to the instant immediately before it, not from a calendar or a global unit — a
// claim moments after a long silence still reads coarse, one moments after a burst reads fine.
test('a candidate is worded by its distance to its own left neighbour', () => {
  const longSilenceThenBurst = [0, 20 * YEAR, 20 * YEAR + 2 * SECOND, 20 * YEAR + 4 * SECOND];
  const scale = timeScale(longSilenceThenBurst);
  const at = (t: number) => scale.tickPositions.find((p) => p.at === t);

  // The claim right after the long silence: coarse, since its left neighbour is 20 years away.
  const afterSilence = at(20 * YEAR);
  assert.ok(afterSilence, 'the claim after the silence was not a candidate at all');
  assert.equal(afterSilence?.unit, 'year');

  // A claim a couple of seconds after ITS left neighbour: fine, regardless of how coarse the
  // claim before *that* one read.
  const inBurst = at(20 * YEAR + 4 * SECOND);
  if (inBurst) assert.equal(inBurst.unit, 'second');
});
