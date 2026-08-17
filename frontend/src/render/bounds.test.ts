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
  hold,
  holdRange,
  needed,
  ratioCeiling,
  stretchFloor,
} from './bounds.ts';

/** A 1000 px viewport, so a share reads directly as pixels. */
const VIEWPORT = 1000;

test('the bound asks for the same share of the viewport whatever is being measured', () => {
  assert.equal(needed(2000, VIEWPORT), MIN_COVER * VIEWPORT);
  assert.equal(needed(VIEWPORT, VIEWPORT), MIN_COVER * VIEWPORT);
});

test('a graph too small to meet the share is only asked to stay wholly in view', () => {
  const size = 200; // well under MIN_COVER of the viewport
  assert.equal(needed(size, VIEWPORT), size);
  const { min, max } = holdRange(size, VIEWPORT);
  assert.equal(min, 0, 'its near edge may not go left of the viewport');
  assert.equal(max, VIEWPORT - size, 'nor its far edge right of it');
});

test('panning stops with the required share still on graph, both directions', () => {
  const size = 2000; // wider than the viewport, so the bound is the share
  const { min, max } = holdRange(size, VIEWPORT);
  for (const start of [min, max, (min + max) / 2]) {
    assert.ok(
      covered(start, size, VIEWPORT) >= MIN_COVER * VIEWPORT - 1e-6,
      `start ${start} left only ${covered(start, size, VIEWPORT)} px covered`,
    );
  }
  // A step beyond either end drops under the bound, which is what makes these the ends.
  assert.ok(covered(min - 1, size, VIEWPORT) < MIN_COVER * VIEWPORT);
  assert.ok(covered(max + 1, size, VIEWPORT) < MIN_COVER * VIEWPORT);
});

// Taking `needed` at whichever of its two branches applies, the range has room at both: the
// containment branch leaves the slack the graph does not fill, and the coverage branch leaves
// what may hang off each edge. So hold never has an inverted range to defend against.
test('the range holds an edge rather than inverting, at any size against any viewport', () => {
  for (const viewport of [1, 250, 1000]) {
    for (const size of [0.01, 0.5, MIN_COVER, 0.99, 1, 2, 50].map((f) => f * viewport)) {
      const { min, max } = holdRange(size, viewport);
      assert.ok(min < max, `${size} px of graph in ${viewport} px inverted to [${min}, ${max}]`);
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

test('a measurement that says nothing yields no bound rather than a wrong one', () => {
  for (const bad of [0, -1]) {
    assert.equal(ratioCeiling(bad, 1, VIEWPORT), null);
    assert.equal(ratioCeiling(1200, bad, VIEWPORT), null);
    assert.equal(ratioCeiling(1200, 1, bad), null);
    assert.equal(stretchFloor(bad, 1, VIEWPORT), null);
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
