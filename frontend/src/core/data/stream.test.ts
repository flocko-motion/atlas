/**
 * package: core / tests
 * type:    test
 * job:     pin that a claim read arrives incrementally and reports as it goes
 * limits:  the parse; the query that produces it is the server's
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';

import { claimsFromStream } from './source.ts';

/** record is one claim as the query encodes it, JSON-sequence style. */
function record(n: number): string {
  const claim = {
    id: `id-${n}`,
    type: 'source/note',
    created_at: '2024-01-01T00:00:00.000000000Z',
    encoding: 'text/plain',
    content_size: 3,
    edges: [],
  };
  return JSON.stringify(claim) + '\n';
}

/** chunked serves a body in pieces, splitting mid-record to be awkward on purpose. */
function chunked(body: string, pieces: number): Response {
  const bytes = new TextEncoder().encode(body);
  const size = Math.ceil(bytes.length / pieces);
  let at = 0;
  const stream = new ReadableStream<Uint8Array>({
    pull(controller) {
      if (at >= bytes.length) {
        controller.close();
        return;
      }
      controller.enqueue(bytes.subarray(at, at + size));
      at += size;
    },
  });
  return { body: stream, text: async () => body } as unknown as Response;
}

test('claims are read as they arrive, and progress climbs', async () => {
  const body = Array.from({ length: 60 }, (_, i) => record(i)).join('');
  const seen: number[] = [];
  const claims = await claimsFromStream(chunked(body, 7), (read) => seen.push(read));

  assert.equal(claims.length, 60);
  assert.equal(claims[0].id, 'id-0');
  assert.equal(claims[59].id, 'id-59');

  // The point of streaming: the count is reported more than once, and it only climbs.
  assert.ok(seen.length > 1, `progress reported ${seen.length} time(s), want it to climb`);
  for (let i = 1; i < seen.length; i++) {
    assert.ok(seen[i] >= seen[i - 1], `progress went backwards: ${seen}`);
  }
  assert.equal(seen[seen.length - 1], 60, 'the final report is not the total');
});

// A chunk boundary lands inside a record, which is the case that would silently lose claims.
test('a record split across chunks survives', async () => {
  const body = record(1) + record(2) + record(3);
  for (const pieces of [2, 3, 5, 11, body.length]) {
    const claims = await claimsFromStream(chunked(body, pieces));
    assert.equal(claims.length, 3, `${pieces} chunk(s) lost a claim`);
    assert.deepEqual(
      claims.map((c) => c.id),
      ['id-1', 'id-2', 'id-3'],
    );
  }
});

test('a body with no stream still parses', async () => {
  const body = record(9);
  const claims = await claimsFromStream({ body: null, text: async () => body } as unknown as Response);
  assert.equal(claims.length, 1);
  assert.equal(claims[0].id, 'id-9');
});
