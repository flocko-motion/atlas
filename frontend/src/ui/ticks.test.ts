/**
 * package: ui / tests
 * type:    test
 * job:     pin which dates get labelled, and that a coarser unit wins when they crowd
 * limits:  placement only; no DOM and no camera
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';

import { boundaries, timeTicks, wordFor } from './ticks.ts';


/** linear is the simple case: an axis with no compression, 1000 px wide. */
function linear(from: number, to: number) {
  return (instant: number) => ((instant - from) / (to - from)) * 1000;
}

test('a new year is always labelled', () => {
  const from = Date.parse('2023-06-01T00:00:00Z');
  const to = Date.parse('2025-06-01T00:00:00Z');
  const ticks = timeTicks({ from, to, xOf: linear(from, to), minGap: 60 });

  const years = ticks.filter((t) => t.major).map((t) => t.label);
  assert.deepEqual(years, ['2024', '2025'], `years = ${JSON.stringify(years)}`);
  for (const year of ticks.filter((t) => t.major)) {
    assert.equal(year.unit, 'year');
  }
});

test('a coarser unit wins when a finer one would crowd', () => {
  const from = Date.parse('2024-01-01T00:00:00Z');
  const to = Date.parse('2024-04-01T00:00:00Z'); // three months across 1000 px

  // Room enough for months, nothing like enough for days.
  const roomy = timeTicks({ from, to, xOf: linear(from, to), minGap: 60 });
  assert.ok(
    roomy.some((t) => t.unit === 'month'),
    `wanted months, got ${JSON.stringify(roomy.map((t) => `${t.unit}:${t.label}`))}`,
  );
  assert.ok(!roomy.some((t) => t.unit === 'day'), 'days should not fit in three months of 1000 px');

  // Demand more room per label and even months give way to the year.
  const tight = timeTicks({ from, to, xOf: linear(from, to), minGap: 400 });
  assert.ok(!tight.some((t) => t.unit === 'month'), 'months survived a 400 px minimum gap');
});

test('a finer unit is used when there is room for it', () => {
  const from = Date.parse('2024-03-04T05:00:00Z');
  const to = Date.parse('2024-03-04T05:10:00Z'); // ten minutes across 1000 px
  const ticks = timeTicks({ from, to, xOf: linear(from, to), minGap: 60 });
  assert.ok(
    ticks.some((t) => t.unit === 'minute' || t.unit === 'second'),
    `wanted minutes, got ${JSON.stringify(ticks.map((t) => t.unit))}`,
  );
});

test('no two labels come closer than the minimum gap', () => {
  const from = Date.parse('2024-01-01T00:00:00Z');
  const to = Date.parse('2024-12-31T00:00:00Z');
  const ticks = timeTicks({ from, to, xOf: linear(from, to), minGap: 90 });
  for (let i = 1; i < ticks.length; i++) {
    assert.ok(
      ticks[i].x - ticks[i - 1].x >= 90 - 1e-6,
      `${ticks[i - 1].label}→${ticks[i].label} are ${(ticks[i].x - ticks[i - 1].x).toFixed(0)} apart`,
    );
  }
});

// The axis compresses silences, so boundaries are not evenly spaced even at one unit. One
// crushed stretch must not push the whole ruler up a unit.
test('a compressed stretch is thinned rather than coarsening the whole ruler', () => {
  const from = Date.parse('2024-01-01T00:00:00Z');
  const to = Date.parse('2024-07-01T00:00:00Z');
  // Half the months land in the first 4% of the width; the rest are spread over the remainder.
  const crushed = (width: number) => (instant: number) => {
    const head = width * 0.04;
    const through = (instant - from) / (to - from);
    return through < 0.5 ? through * 2 * head : head + (through - 0.5) * 2 * (width - head);
  };
  // 1000 px draws the axis across the whole canvas; 600 is as far as the wheel compresses it,
  // which is 40% past what used to be reachable.
  for (const width of [1000, 600]) {
    const ticks = timeTicks({ from, to, xOf: crushed(width), minGap: 60 });
    assert.ok(
      ticks.some((t) => t.unit === 'month'),
      `the crushed half coarsened the whole ruler at ${width} px`,
    );
    for (let i = 1; i < ticks.length; i++) {
      assert.ok(ticks[i].x - ticks[i - 1].x >= 60 - 1e-6, `thinning left a collision at ${width} px`);
    }
  }
});

test('boundaries land on the calendar, not on offsets from the start', () => {
  const from = Date.parse('2024-03-04T05:06:07Z');
  const to = Date.parse('2024-03-06T00:00:00Z');
  for (const at of boundaries(from, to, 'day')) {
    assert.equal(new Date(at).toISOString().slice(11), '00:00:00.000Z', 'a day tick is not midnight');
  }
  const months = boundaries(Date.parse('2024-01-15T00:00:00Z'), Date.parse('2024-04-02T00:00:00Z'), 'month');
  assert.deepEqual(
    months.map((m) => new Date(m).toISOString().slice(0, 10)),
    ['2024-02-01', '2024-03-01', '2024-04-01'],
  );
});

test('a boundary is worded by the part that distinguishes it', () => {
  const at = Date.parse('2024-03-04T05:06:07.008Z');
  assert.equal(wordFor(at, 'year'), '2024');
  assert.equal(wordFor(at, 'month'), 'Mar');
  assert.equal(wordFor(at, 'day'), 'Mar 4');
  assert.equal(wordFor(at, 'hour'), '05:06');
  assert.equal(wordFor(at, 'second'), '05:06:07');
});

test('an instant span yields no crowd of labels', () => {
  const at = Date.parse('2024-03-04T05:06:07Z');
  const ticks = timeTicks({ from: at, to: at, xOf: () => 0, minGap: 60 });
  assert.ok(ticks.length <= 2, `a zero span produced ${ticks.length} labels`);
  assert.deepEqual(timeTicks({ from: 10, to: 0, xOf: () => 0, minGap: 60 }), []);
});

// A large archive spanning hours contains no year, month or day boundary, and finer units are
// crowded — which left the ruler empty until it was zoomed right in.
test('a span with no calendar boundary inside it still gets labels', () => {
  const from = Date.parse('2024-03-04T05:06:07Z');
  const to = Date.parse('2024-03-04T09:52:41Z'); // under five hours, mid-day, no midnight
  const ticks = timeTicks({ from, to, xOf: linear(from, to), minGap: 400 });

  assert.ok(ticks.length > 0, 'no labels at all for a mid-day span');
  for (const tick of ticks) {
    assert.ok(tick.label.length > 0, 'a label with no text');
  }
});

test('a span shorter than any boundary labels its own ends', () => {
  const from = Date.parse('2024-03-04T05:06:07.100Z');
  const to = Date.parse('2024-03-04T05:06:07.900Z'); // 800 ms, no second boundary inside
  const ticks = timeTicks({ from, to, xOf: linear(from, to), minGap: 300 });
  assert.ok(ticks.length >= 1, 'a sub-second span produced no labels');
  assert.notEqual(ticks[0].label, ticks[ticks.length - 1]?.label, 'both ends read the same');
});
