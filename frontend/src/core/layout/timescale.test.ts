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
const HOUR = 60 * MINUTE;
const DAY = 86_400 * SECOND;
const YEAR = 365 * DAY;

/**
 * maxGapFraction is the widest stretch of the axis between consecutive candidates (or from an
 * end to the nearest one), as a fraction of the whole width — the shape a raw candidate count
 * cannot see: a count can stay small and healthy while every candidate piles into one corner
 * and the rest of the axis goes unlabelled, which is this ticket's bug exactly.
 */
function maxGapFraction(scale: { tickPositions: readonly { axisX: number }[]; width: number }): number {
  if (scale.width <= 0) return 0;
  const xs = scale.tickPositions.map((p) => p.axisX).sort((a, b) => a - b);
  let gap = xs[0] ?? scale.width;
  for (let i = 1; i < xs.length; i++) gap = Math.max(gap, xs[i] - xs[i - 1]);
  gap = Math.max(gap, scale.width - (xs[xs.length - 1] ?? 0));
  return gap / scale.width;
}

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

// sd-b56636: a region's share of candidates follows its own claims, not how wide some other
// region of the axis happens to be — a dense run's sheer number of small gaps can otherwise sum
// to nearly the whole axis width and leave a sparser tail with almost none of it to claim.
test('a sparse tail after a dense burst keeps candidates of its own', () => {
  const dense = Array.from({ length: 3000 }, (_, i) => i * 4 * HOUR); // ~1.4 years, every 4h
  const tailStart = dense[dense.length - 1] + 20 * DAY;
  const tail = Array.from({ length: 8 }, (_, i) => tailStart + i * 30 * DAY); // ~8 months, sparse
  const scale = timeScale([...dense, ...tail]);

  const inTail = scale.tickPositions.filter((p) => p.at >= tailStart);
  assert.ok(
    inTail.length >= 2,
    `expected more than a single candidate across the sparse tail, got ${inTail.length}`,
  );
  // The stronger claim: candidates of its own means SPREAD across it, not just two endpoints
  // (which any implementation trivially keeps) — assert the tail's own coverage, not just count.
  // Positions are made relative to the tail's own start, since maxGapFraction reads them as
  // an offset from an axis beginning at 0, not the global axis this tail sits far along.
  const tailOrigin = scale.toX(tailStart);
  const tailWidth = scale.toX(tail[tail.length - 1]) - tailOrigin;
  const relative = inTail.map((p) => ({ axisX: p.axisX - tailOrigin }));
  const fraction = maxGapFraction({ tickPositions: relative, width: tailWidth });
  assert.ok(fraction <= 0.6, `widest unlabelled stretch within the tail was ${(fraction * 100).toFixed(1)}%`);
});

// A single ordinary pause (a weekend, a maintenance window) tags the same unit as the tail
// itself. Pooling by unit name alone re-merges the two into one pool and the pause drags the
// tail's threshold back up to what starved it before; a contiguous RUN keeps the pause as its
// own small run instead, since the dense claims on either side break it off from the tail.
test('a same-scale pause inside the dense run does not re-starve the tail', () => {
  const dense = Array.from({ length: 3000 }, (_, i) => i * 4 * HOUR);
  const midpoint = 1500;
  for (let i = midpoint; i < dense.length; i++) dense[i] += 3 * DAY; // one day-scale pause
  const tailStart = dense[dense.length - 1] + 20 * DAY;
  const tail = Array.from({ length: 8 }, (_, i) => tailStart + i * 30 * DAY);
  const scale = timeScale([...dense, ...tail]);

  const inTail = scale.tickPositions.filter((p) => p.at >= tailStart);
  assert.ok(
    inTail.length >= 2,
    `expected the pause to not re-starve the tail, got ${inTail.length} candidates`,
  );
});

// The mirror of the tail test, from the other side: a dense, substantial region must keep its
// own candidates when an unrelated region made of MANY small runs is appended elsewhere, AND
// that appended region must itself stay covered — a fix that only protects the side named in a
// repro (as the previous round's did) moves the bug rather than closing it.
test('neither side starves when a many-run region is appended to a dense one', () => {
  const dense = Array.from({ length: 3000 }, (_, i) => i * 4 * HOUR); // one uniform run
  const alone = timeScale(dense).tickPositions.length;

  let t = dense[dense.length - 1];
  const wiggly: number[] = [];
  for (let i = 0; i < 60; i++) {
    t += i % 2 === 0 ? 10 * SECOND : 5 * DAY; // alternates gap scale every step: 60 tiny runs
    wiggly.push(t);
  }
  const scale = timeScale([...dense, ...wiggly]);
  const denseSide = scale.tickPositions.filter((p) => p.at <= dense[dense.length - 1]).length;
  const wigglySide = scale.tickPositions.filter((p) => p.at > dense[dense.length - 1]).length;
  assert.ok(denseSide >= alone * 0.8, `dense region had ${alone} candidates alone, only ${denseSide} once appended`);
  assert.ok(wigglySide >= 4, `the appended region itself was left with only ${wigglySide} candidates`);
});

// Uniform spacing is the one shape a flat per-run share and a width-proportional one agree on
// (a single run); a realistic archive with many small runs — a burst every day, say — is where
// splitting the SAME fixed budget equally across runs instead of by width regressed: R runs of
// (`floor(TICK_BUDGET/R)` or a floor) each is linear in R, not bounded by it.
test('tickPositions count does not grow with how many runs feed it either', () => {
  const daily = (days: number) => {
    const times: number[] = [];
    let day = 0;
    for (let d = 0; d < days; d++) {
      for (let i = 0; i < 5; i++) times.push(day + i * MINUTE); // a 5-claim burst, once a day
      day += DAY;
    }
    return times;
  };
  const few = timeScale(daily(100)).tickPositions.length;
  const many = timeScale(daily(3000)).tickPositions.length;
  assert.ok(many < few * 4, `100 days produced ${few} candidates, 3000 days produced ${many} — not bounded`);
});

// A count staying small proves nothing about WHERE the candidates landed: this ticket's actual
// bug survived a round where every test above passed, because the whole budget piled into one
// contiguous block of the axis and the rest — most of it — went unlabelled. No claim-covered
// stretch may be left further than a modest share of the axis from its nearest candidate,
// across every shape a realistic archive's density can take.
test('no wide stretch of a claim-covered axis goes without a candidate', () => {
  const daily = (days: number) => {
    const times: number[] = [];
    let day = 0;
    for (let d = 0; d < days; d++) {
      for (let i = 0; i < 5; i++) times.push(day + i * MINUTE);
      day += DAY;
    }
    return times;
  };
  const alternating = (n: number) => {
    const times: number[] = [];
    let t = 0;
    for (let i = 0; i < n; i++) {
      times.push(t);
      t += i % 2 === 0 ? 10 * SECOND : 5 * DAY;
    }
    return times;
  };
  const mixed = (uniformCount: number, alternatingCount: number) => {
    const uniform = Array.from({ length: uniformCount }, (_, i) => i * 4 * HOUR);
    let t = uniform[uniform.length - 1];
    const alt: number[] = [];
    for (let i = 0; i < alternatingCount; i++) {
      t += i % 2 === 0 ? 10 * SECOND : 5 * DAY;
      alt.push(t);
    }
    return [...uniform, ...alt];
  };

  const shapes: Record<string, number[]> = {
    'daily(100)': daily(100),
    'daily(1000)': daily(1000),
    'daily(3000)': daily(3000),
    'alternating(2000)': alternating(2000),
    'uniform(300)+alternating(600)': mixed(300, 600),
  };
  for (const [name, times] of Object.entries(shapes)) {
    const fraction = maxGapFraction(timeScale(times));
    assert.ok(fraction <= 0.2, `${name}: widest unlabelled stretch was ${(fraction * 100).toFixed(1)}% of the axis`);
  }
});

/**
 * jitter is a deterministic, non-uniform offset — exact arithmetic sequences are the one shape
 * where floating-point shares tie exactly, which hid budgetsFor's original floor-pool bug for
 * five review rounds: a real archive's `created_at` values never land on a grid.
 */
function jitter(i: number, spreadMs: number): number {
  return (i * 2654435761) % spreadMs;
}

// The failure that survived every prior test: an exact grid where the deciles a reader would
// zoom into hold near-equal shares, so a share-ranked (rather than position-ranked) floor pool
// happened to spread evenly by coincidence. One millisecond of jitter breaks the coincidence.
// Both the whole-axis view (maxGapFraction) and what a reader sees zoomed into any one tenth of
// the axis (per-decile share vs. per-decile candidates) have to hold, on data with no exact ties.
test('a jittered, unevenly-paused archive still covers every decile it has claims in', () => {
  const hourlyWithPauses = (claims: number, pauseEvery: number) => {
    const times: number[] = [];
    let t = 0;
    for (let i = 0; i < claims; i++) {
      times.push(t + jitter(i, HOUR));
      t += HOUR;
      if ((i + 1) % pauseEvery === 0) t += 5 * DAY;
    }
    return times;
  };
  const dailyJittered = (days: number) => {
    const times: number[] = [];
    let day = 0;
    for (let d = 0; d < days; d++) {
      for (let i = 0; i < 5; i++) times.push(day + i * MINUTE + jitter(d * 5 + i, MINUTE));
      day += DAY;
    }
    return times;
  };

  for (const [name, times] of Object.entries({
    'hourly, 5-day pause every 50': hourlyWithPauses(3000, 50),
    'daily burst of 5, jittered': dailyJittered(3000),
  })) {
    const scale = timeScale(times);
    assert.ok(
      maxGapFraction(scale) <= 0.2,
      `${name}: widest unlabelled stretch was ${(maxGapFraction(scale) * 100).toFixed(1)}%`,
    );

    // The stronger, decile-local claim: a tenth of the axis holding a meaningful share of the
    // claims must hold more than a token candidate of its own — a healthy whole-axis figure
    // can still hide one thin decile once select() only draws ~15 labels regardless of zoom.
    const claimDeciles = new Array(10).fill(0);
    for (const at of times) claimDeciles[Math.min(9, Math.floor((scale.toX(at) / scale.width) * 10))]++;
    const candidateDeciles = new Array(10).fill(0);
    for (const p of scale.tickPositions) candidateDeciles[Math.min(9, Math.floor((p.axisX / scale.width) * 10))]++;
    for (let d = 0; d < 10; d++) {
      if (claimDeciles[d] < times.length * 0.01) continue; // negligible share: nothing to cover
      assert.ok(
        candidateDeciles[d] >= 3,
        `${name}: decile ${d} holds ${claimDeciles[d]} claims but only ${candidateDeciles[d]} candidates`,
      );
    }
  }
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
