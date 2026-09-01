/**
 * package: core / tests
 * type:    test
 * job:     pin that a claim's records render by name, and a malformed one still renders what
 *          could be framed alongside the deviations, rather than going blank
 * limits:  headless; no DOM, no Sigma
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';
import { generateKeyPairSync, sign } from 'node:crypto';
import { CborWriter, contributorFrom, encodeEnvelope, newClaim } from '@rankegraph/ranke';
import type { Contributor, Signer } from '@rankegraph/ranke';

import { inspectClaimBytes } from './cbor.ts';
import type { CborNode, RenderedRecord } from './cbor.ts';

/** slot finds a record's slot by its numeric key. */
function slot(record: RenderedRecord | undefined, key: number) {
  return record?.slots.find((s) => s.key === key);
}

/**
 * testSigner is a throwaway Ed25519 identity: newClaim needs one to build a signed envelope,
 * and decode never checks a signature against a key — that is a separate verification depth
 * this file has no reason to exercise.
 */
function testSigner(): Signer {
  const { publicKey, privateKey } = generateKeyPairSync('ed25519');
  const rawPub = (publicKey.export({ type: 'spki', format: 'der' }) as Buffer).subarray(-32);
  const pubkey = new Uint8Array([0xed, 0x01, ...rawPub]); // ed25519-pub multicodec prefix
  return { pubkey, sign: (message) => new Uint8Array(sign(null, message, privateKey)) };
}

/** testContributor builds and signs the root contributor claim `type` attributes to. */
function testContributor(signer: Signer): Contributor {
  const root = newClaim({
    type: 'contribution/contributor',
    signer,
    content: { kind: 'inline', bytes: signer.pubkey, size: signer.pubkey.length, encoding: 'application/octet-stream' },
  });
  return contributorFrom(root.claim);
}

// A real claim, built and signed through the library's own newClaim rather than hand-encoded,
// exercises the shapes it actually produces: nested maps, a text field, a text-keyed map for
// user fields — so the walk is checked against what the library produces, not a fixture built
// to fit it.
test('a valid claim renders one node record, its slots named from ranke-ts', () => {
  const signer = testSigner();
  const contributor = testContributor(signer);
  const { bytes } = newClaim({
    type: 'entity/zznote',
    contributor,
    signer,
    fields: { note: 'hello' },
  });

  const view = inspectClaimBytes(bytes);
  assert.equal(view.valid, true);
  assert.deepEqual([...view.deviations], []);
  // node + the contributor edge newClaim always attaches.
  assert.equal(view.records.length, 2);
  const node = view.records.find((r) => r.kind === 'node');
  assert.ok(node);

  // Key 1 (type_class) is a string — aliased on the wire for a reserved class like this one,
  // which is the encoder's concern, not this walk's; its slot name comes from ranke-ts.
  const typeClass = slot(node, 1);
  assert.equal(typeClass?.name, 'type_class');
  assert.equal(typeClass?.value?.kind, 'text');

  // Key 8 (fields) is the user-fields map: text keys, text values — the one place a real
  // claim's own map keys are not small ints, so the walk is checked against that too.
  const fields = slot(node, 8);
  assert.equal(fields?.name, 'fields');
  const fieldsValue = fields?.value as { kind: 'map'; entries: { key: CborNode; value: CborNode }[] } | undefined;
  assert.equal(fieldsValue?.kind, 'map');
  assert.deepEqual(fieldsValue?.entries, [
    { key: { kind: 'text', value: 'note' }, value: { kind: 'text', value: 'hello' } },
  ]);
});

// Every claim but a root contributor carries a contributor edge, exercising the second record
// kind: it is embedded raw under the node's own edges slot, and ranke-ts frames it as its own
// named record rather than this file walking into a bare array of anonymous maps.
test('an edge is its own named record, nested under the node that carries it', () => {
  const signer = testSigner();
  const contributor = testContributor(signer);
  const { bytes } = newClaim({ type: 'entity/zznote', contributor, signer });

  const view = inspectClaimBytes(bytes);
  assert.equal(view.valid, true);
  assert.equal(view.records.length, 2);
  assert.deepEqual(view.records.map((r) => r.path), ['node', 'node.edges[0]']);

  const edge = view.records[1];
  assert.equal(edge.kind, 'edge');
  // The reference is stored as the raw multihash bytes an id decodes to, not its multibase
  // text — the wire is compact binary, and decoding that back to a string is content's job,
  // not this generic value walk's.
  const reference = slot(edge, 12);
  assert.equal(reference?.name, 'reference');
  assert.equal(reference?.value?.kind, 'bytes');
});

// §4.2 orders a map's entries strictly ascending by encoded key bytes. `CborWriter` writes
// whatever entries it is handed in whatever order (only `writeSortedMap` sorts, and this
// deliberately bypasses it), so a node record written key 2 then key 1, sealed in a real
// envelope, is the malformed claim a debugger most wants a straight answer about —
// inspectClaim reports it rather than this file re-deriving it, and the view renders
// whatever the walk could still frame instead of going blank.
test('a malformed claim still renders what could be framed, with the deviations named', () => {
  const w = new CborWriter();
  w.writeMapHeader(2);
  w.writeUint(2);
  w.writeText('b');
  w.writeUint(1);
  w.writeText('a');
  const envelope = encodeEnvelope(w.bytes(), new Uint8Array(64));

  const view = inspectClaimBytes(envelope);
  assert.equal(view.valid, false, 'malformed bytes read as a valid claim');
  assert.equal(view.records.length, 1, 'the record the walk could still frame went missing');
  const [node] = view.records;
  assert.equal(node.slots.length, 2, 'a slot the walk could still frame went missing');

  assert.equal(view.deviations.length, 1);
  assert.match(view.deviations[0].message, /canonical order/);
  assert.equal(view.deviations[0].path, 'node');
});

// A node record can frame the same key twice — inspectClaim reports the repeat as a
// deviation rather than collapsing it, so a slot list is not guaranteed unique keys. The
// viewer keys its rendered list by index rather than by slot.key for exactly this reason
// (-> ui/shell/CborView.tsx CborRecord); this pins that the shape it guards against is real.
test('a record framed with a repeated key keeps both slots, not just one', () => {
  const w = new CborWriter();
  w.writeMapHeader(2);
  w.writeUint(1);
  w.writeText('a');
  w.writeUint(1);
  w.writeText('b');
  const envelope = encodeEnvelope(w.bytes(), new Uint8Array(64));

  const view = inspectClaimBytes(envelope);
  const [node] = view.records;
  assert.equal(node.slots.length, 2, 'a repeated key collapsed to one slot');
  assert.deepEqual(node.slots.map((s) => s.key), [1, 1]);
  assert.match(view.deviations[0]?.message ?? '', /canonical order/);
});
