/**
 * package: core / tests
 * type:    test
 * job:     pin selfOrigin's detection of a page ranke-db itself served
 * limits:  the detection only; loadPersisted's use of it is one line and untested here
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';

import { selfOrigin } from './connections.ts';

/** withWindow runs fn with globalThis.window set to a fake location, restoring after. */
function withWindow<T>(pathname: string, origin: string, fn: () => T): T {
  const original = (globalThis as { window?: unknown }).window;
  (globalThis as { window?: unknown }).window = { location: { pathname, origin } };
  try {
    return fn();
  } finally {
    (globalThis as { window?: unknown }).window = original;
  }
}

test('no window (Node, this test runner) is null', () => {
  assert.equal(selfOrigin(), null);
});

test('served at any path but /explorer is null', () => {
  assert.equal(
    withWindow('/', 'http://localhost:8080', selfOrigin),
    null,
  );
});

test('served at exactly /explorer is that origin', () => {
  assert.equal(
    withWindow('/explorer', 'https://ranke.example.com', selfOrigin),
    'https://ranke.example.com',
  );
});
