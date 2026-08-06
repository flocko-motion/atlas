/**
 * package: core / tests
 * type:    test
 * job:     pin that a claim read arrives incrementally and reports as it goes
 * limits:  the decode is the library's; this pins how the read loop drives it
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';

import { claimsFromBody } from './source.ts';

/** RFC 7464 opens each record with RS and ends it with LF, which is what the server writes. */
const RS = '\u001e';

/** record is one claim as a query encodes it under `output.encoding: json`. */
function record(n: number): string {
  const claim = {
    id: `id-${n}`,
    type: 'source/note',
    created_at: '2024-01-01T00:00:00.000000000Z',
    height: 1,
    encoding: 'text/plain',
    content_size: 3,
    content_hash: 'bciqdlnrhbcnkalcqxrpxpmroin6iu5w6dgfjqoemvxlvvhtwepbe6ma',
    edges: [],
  };
  return `${RS}${JSON.stringify(claim)}\n`;
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

// 7.2 — the count climbs while the body is still arriving, which is the whole point of
// decoding a sequence rather than a document.
test('claims are read as they arrive, and progress climbs', async () => {
  const body = Array.from({ length: 60 }, (_, i) => record(i)).join('');
  const seen: number[] = [];
  const bytes: number[] = [];
  const claims = await claimsFromBody(chunked(body, 7), (read, bytesRead) => {
    seen.push(read);
    bytes.push(bytesRead);
  });

  assert.equal(claims.length, 60);
  assert.equal(claims[0].claim.id, 'id-0');
  assert.equal(claims[59].claim.id, 'id-59');

  assert.ok(seen.length > 1, `progress reported ${seen.length} time(s), want it to climb`);
  assert.ok(
    seen.some((read) => read > 0 && read < 60),
    `a count was only reported at the ends: ${seen}`,
  );
  for (let i = 1; i < seen.length; i++) {
    assert.ok(seen[i] >= seen[i - 1], `progress went backwards: ${seen}`);
    assert.ok(bytes[i] >= bytes[i - 1], `bytes went backwards: ${bytes}`);
  }
  assert.equal(seen[seen.length - 1], 60, 'the final report is not the total');
  // The bytes are the reader's own tally, so they end at the body's length — no header said so.
  assert.equal(bytes[bytes.length - 1], new TextEncoder().encode(body).length);
});

// A chunk boundary lands inside a record, which is the case that would silently lose claims.
test('a record split across chunks survives', async () => {
  const body = record(1) + record(2) + record(3);
  for (const pieces of [2, 3, 5, 11, body.length]) {
    const claims = await claimsFromBody(chunked(body, pieces));
    assert.equal(claims.length, 3, `${pieces} chunk(s) lost a claim`);
    assert.deepEqual(
      claims.map((c) => c.claim.id),
      ['id-1', 'id-2', 'id-3'],
    );
  }
});

test('a body with no stream still parses', async () => {
  const body = record(9);
  const claims = await claimsFromBody({ body: null, text: async () => body } as unknown as Response);
  assert.equal(claims.length, 1);
  assert.equal(claims[0].claim.id, 'id-9');
});

// A query may append an execution report as its last record. The reader passes it over, so
// nothing here inspects a record to decide what it is — which is the point of the boundary.
test('a trailing execution report is not read as a claim', async () => {
  const body = `${record(1)}${RS}{"started_at":"t","elapsed_ns":1200,"results":1}\n`;
  const claims = await claimsFromBody(chunked(body, 3));
  assert.deepEqual(
    claims.map((c) => c.claim.id),
    ['id-1'],
  );
});

// What a read cannot know: a claim arrives with no contribution index, since no output field
// reports one — so it is 0, and the history layout says so rather than drawing a false order.
test('a claim read from a response carries no contribution index', async () => {
  const claims = await claimsFromBody(chunked(record(4), 1));
  assert.equal(claims[0].contribution, 0);
  assert.equal(claims[0].branch, '');
  // Content the read left external is a hash, so the caption falls back to type and id.
  assert.equal(claims[0].label, 'source/note id-4');
});
