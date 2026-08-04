/**
 * package: ui / tests
 * type:    test
 * job:     pin that a label's precision follows the span in view
 * limits:  formatting only
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';

import { asText, formatExact, formatInstant, hexDump, isTextual } from './format.ts';

const SECOND = 1000;
const DAY = 86_400 * SECOND;
const YEAR = 365 * DAY;

test('a label is written at the precision the visible span justifies', () => {
  const at = Date.parse('2024-03-04T05:06:07.008Z');
  assert.equal(formatInstant(at, 500), '05:06:07.008');
  assert.equal(formatInstant(at, 30 * SECOND), '05:06:07');
  assert.equal(formatInstant(at, 5 * SECOND * 3600), '05:06');
  assert.equal(formatInstant(at, 5 * DAY), '03-04 05:06');
  assert.equal(formatInstant(at, 200 * DAY), '2024-03-04');
  assert.equal(formatInstant(at, 10 * YEAR), '2024-03');
  assert.equal(formatInstant(Number.NaN, DAY), '—');
});

test('bytes are read as text only where the encoding says they are', () => {
  assert.ok(isTextual('text/plain'));
  assert.ok(isTextual('application/json'));
  assert.ok(isTextual('image/svg+xml'));
  assert.ok(!isTextual('application/octet-stream'));
  assert.ok(!isTextual(undefined));
});

test('a hex dump shows offset, bytes and the printable characters beside them', () => {
  const bytes = new Uint8Array([0xed, 0x01, 0x41, 0x42, 0x0a]);
  const line = hexDump(bytes);
  assert.match(line, /^000000  ed 01 41 42 0a/);
  assert.match(line, /··AB·$/);
  // Two rows once past sixteen bytes, each labelled with its own offset.
  const long = hexDump(new Uint8Array(20));
  assert.equal(long.split('\n').length, 2);
  assert.match(long.split('\n')[1], /^000010/);
});

test('text decodes without throwing on bytes that are not UTF-8', () => {
  assert.equal(asText(new TextEncoder().encode('a note')), 'a note');
  assert.equal(typeof asText(new Uint8Array([0xff, 0xfe])), 'string');
});

// The cursor points at one claim, so it says exactly when — no precision shed for a span.
test('the exact form keeps the date and the milliseconds', () => {
  const at = Date.parse('2024-03-04T05:06:07.008Z');
  assert.equal(formatExact(at), '2024-03-04 05:06:07.008');
  assert.equal(formatExact(Number.NaN), '—');
});
