/**
 * package: render / tests
 * type:    test
 * job:     pin the one bound — that neither zoom nor pan can leave the viewport off the graph
 * limits:  arithmetic only; no camera and no DOM
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';

import {
  MIN_COVER,
  covered,
  fillRatio,
  fillStretch,
  hold,
  holdRange,
  needed,
  ratioCeiling,
  stretchFloor,
} from './bounds.ts';

/** A 1000 px viewport, so a share reads directly as pixels. */
const VIEWPORT = 1000;

// Two rules over one measurement: how small a zoom may draw the graph, which is what leaves
// padding — and where a pan may then put it, which may add none of its own. Both axes are held
// the same way; only how far each may be zoomed out differs, and that is the ceiling below.
test('a pan adds no padding: a graph past the viewport covers it, whichever way it is pushed', () => {
  const size = 2000; // larger than the viewport, as a stretched axis or a fitted band is
  const { min, max } = holdRange(size, VIEWPORT);
  assert.deepEqual({ min, max }, { min: VIEWPORT - size, max: 0 });
  for (const start of [min, max, (min + max) / 2]) {
    assert.equal(covered(start, size, VIEWPORT), VIEWPORT, `a near edge at ${start} left a gap`);
  }
  // A step beyond either end opens one, which is what makes these the ends.
  assert.ok(covered(min - 1, size, VIEWPORT) < VIEWPORT);
  assert.ok(covered(max + 1, size, VIEWPORT) < VIEWPORT);
});

// Where a fit lands: filling the viewport exactly, there is nowhere else for it to be.
test('a graph the size of the viewport is held to it', () => {
  assert.deepEqual(holdRange(VIEWPORT, VIEWPORT), { min: 0, max: 0 });
  assert.equal(hold(270, VIEWPORT, VIEWPORT), -270, 'a picture pushed past the far edge');
  assert.equal(hold(-270, VIEWPORT, VIEWPORT), 270, 'a picture pushed past the near one');
});

test('a graph smaller than the viewport is only asked to stay wholly in view', () => {
  const size = 400; // padding a zoom left, which a pan keeps rather than adding to
  assert.equal(needed(size, VIEWPORT), size);
  const { min, max } = holdRange(size, VIEWPORT);
  assert.equal(min, 0, 'its near edge may not go past the near edge of the viewport');
  assert.equal(max, VIEWPORT - size, 'nor its far edge past the far one');
});

// Taking `needed` at whichever of its two branches applies, the range never inverts: the
// containment branch leaves the slack the graph does not fill, the coverage branch leaves what
// hangs off each edge, and at one size they meet. So hold has nothing to defend against.
test('the range holds an edge rather than inverting, at any size against any viewport', () => {
  for (const viewport of [1, 250, 1000]) {
    for (const size of [0.01, 0.5, MIN_COVER, 0.99, 1, 2, 50].map((f) => f * viewport)) {
      const { min, max } = holdRange(size, viewport);
      assert.ok(min <= max, `${size} px of graph in ${viewport} px inverted to [${min}, ${max}]`);
    }
  }
});

test('hold moves an out-of-bounds edge back and leaves an in-bounds one alone', () => {
  const size = 2000;
  const { min, max } = holdRange(size, VIEWPORT);
  assert.equal(hold(min - 300, size, VIEWPORT), 300, 'a graph panned off to the left');
  assert.equal(hold(max + 300, size, VIEWPORT), -300, 'a graph panned off to the right');
  assert.equal(hold((min + max) / 2, size, VIEWPORT), 0, 'a graph already in view was moved');
});

test('the ratio ceiling is where the graph has shrunk to exactly the bound', () => {
  // Drawn at 1200 px with ratio 1, so it may grow to 1200 / 600 = 2 before it covers only 60%.
  const ceiling = ratioCeiling(1200, 1, VIEWPORT);
  assert.equal(ceiling, 2);
  // Size runs inversely with ratio, so at the ceiling the graph covers the bound exactly.
  assert.equal(1200 / ceiling, MIN_COVER * VIEWPORT);
});

test('the stretch floor is the same bound solved the other way round', () => {
  // Drawn at 1200 px at stretch 1: compressing to half of that lands exactly on the bound.
  const floor = stretchFloor(1200, 1, VIEWPORT);
  assert.equal(floor, 0.5);
  assert.equal(1200 * floor, MIN_COVER * VIEWPORT);
});

// Fitting is the same arithmetic as the bound, asked for the whole viewport rather than the
// bound's share of it — which is what fit-height did wrong while it sent a fixed ratio.
test('the fitting ratio brings a drawn span to exactly the viewport, either way round', () => {
  // A picture drawn at 400 px must grow to 1000: the camera zooms in, so its ratio falls.
  const short = fillRatio(400, 1, VIEWPORT) as number;
  assert.equal(short, 0.4);
  assert.equal(400 / short, VIEWPORT);
  // And one drawn at 2500 px must shrink to it, from whatever ratio it was measured at.
  const tall = fillRatio(2500, 2, VIEWPORT) as number;
  assert.equal(tall, 5);
  assert.equal((2500 * 2) / tall, VIEWPORT);
});

// The strata are fitted by stretching them, since the camera would magnify the claims with them.
test('the fitting stretch brings a drawn span to exactly the viewport, either way round', () => {
  // Drawn 20 px tall at stretch 1 — a band of strata against a wide axis, which is the case the
  // fit exists for. Filling 1000 px asks for fifty times the room.
  const band = fillStretch(20, 1, VIEWPORT) as number;
  assert.equal(band, 50);
  assert.equal(20 * band, VIEWPORT);
  // Size runs with the stretch, so a picture already taller than the viewport comes back down.
  const tall = fillStretch(4000, 4, VIEWPORT) as number;
  assert.equal(tall, 1);
  assert.equal((4000 / 4) * tall, VIEWPORT);
});

test('a measurement that says nothing yields no bound rather than a wrong one', () => {
  for (const bad of [0, -1]) {
    assert.equal(ratioCeiling(bad, 1, VIEWPORT), null);
    assert.equal(ratioCeiling(1200, bad, VIEWPORT), null);
    assert.equal(ratioCeiling(1200, 1, bad), null);
    assert.equal(stretchFloor(bad, 1, VIEWPORT), null);
    assert.equal(fillRatio(bad, 1, VIEWPORT), null);
    assert.equal(fillRatio(1200, bad, VIEWPORT), null);
    assert.equal(fillRatio(1200, 1, bad), null);
    assert.equal(fillStretch(bad, 1, VIEWPORT), null);
    assert.equal(fillStretch(1200, bad, VIEWPORT), null);
    assert.equal(fillStretch(1200, 1, bad), null);
  }
});

// The two instruments moved the picture under different limits, which is what let a load land
// where the wheel could not follow. Reaching the bound either way must give the same picture.
test('reaching the bound by camera and by stretch leaves the same drawn width', () => {
  const drawn = 1200;
  const byCamera = drawn / (ratioCeiling(drawn, 1, VIEWPORT) as number);
  const byStretch = drawn * (stretchFloor(drawn, 1, VIEWPORT) as number);
  assert.equal(byCamera, byStretch);
  assert.equal(byCamera, MIN_COVER * VIEWPORT);
});
