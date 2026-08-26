/**
 * package: core / cbor
 * type:    logic
 * job:     turn a claim's raw CBOR into a tree a pane can render, by record — names, offsets
 *          and the malformed-claim path all come from ranke-ts's inspectClaim, not from here
 * limits:  headless; no React, no DOM — rendering the tree is the UI's (-> ui)
 */

import { CborReader, inspectClaim } from '@flocko-motion/ranke';
import type { Deviation, InspectedSlot } from '@flocko-motion/ranke';

/** One decoded CBOR value — a slot's own value, or something nested inside it. */
export type CborNode =
  | { kind: 'int'; value: bigint }
  | { kind: 'bytes'; value: Uint8Array }
  | { kind: 'text'; value: string }
  | { kind: 'bool'; value: boolean }
  | { kind: 'null' }
  | { kind: 'array'; items: CborNode[] }
  | { kind: 'map'; entries: { key: CborNode; value: CborNode }[] };

/** One rendered slot of a record — inspectClaim's own key/name/at/length, plus its decoded value. */
export interface RenderedSlot extends InspectedSlot {
  value: CborNode | null;
  error: string | null;
}

/** One rendered record — a node, or one edge embedded in it (path "node.edges[N]"). */
export interface RenderedRecord {
  kind: 'node' | 'edge';
  path: string;
  at: number;
  slots: RenderedSlot[];
}

/** ClaimView is what a claim's bytes hold and what is wrong with them, ready to render. */
export interface ClaimView {
  valid: boolean;
  records: RenderedRecord[];
  deviations: readonly Deviation[];
}

const MAJOR_UINT = 0;
const MAJOR_NEGINT = 1;
const MAJOR_BYTES = 2;
const MAJOR_TEXT = 3;
const MAJOR_ARRAY = 4;
const MAJOR_MAP = 5;
const MAJOR_TAG = 6;
const MAJOR_SIMPLE = 7;

// The reader has no peek, so dispatch reads the leading byte's own major-type bits — every
// canonical decoder's, not this file's — to pick which typed accessor reads the value. Key
// order is not re-checked here: a slot's bytes were already framed by inspectClaim, whose own
// walk is the one place that concern belongs (record_keys.ts recordKeyName / inspect.ts).
function readValue(r: CborReader, bytes: Uint8Array): CborNode {
  const head = bytes[r.position];
  if (head === undefined) throw new Error('unexpected end of input');
  switch (head >> 5) {
    case MAJOR_UINT:
    case MAJOR_NEGINT:
      return { kind: 'int', value: r.readInt() };
    case MAJOR_BYTES:
      return { kind: 'bytes', value: r.readBytes() };
    case MAJOR_TEXT:
      return { kind: 'text', value: r.readText() };
    case MAJOR_ARRAY: {
      const n = r.readArrayHeader();
      const items: CborNode[] = [];
      for (let i = 0; i < n; i++) items.push(readValue(r, bytes));
      return { kind: 'array', items };
    }
    case MAJOR_MAP: {
      const n = r.readMapHeader();
      const entries: { key: CborNode; value: CborNode }[] = [];
      for (let i = 0; i < n; i++) {
        const key = readValue(r, bytes);
        const value = readValue(r, bytes);
        entries.push({ key, value });
      }
      return { kind: 'map', entries };
    }
    case MAJOR_SIMPLE: {
      const b = r.readSimple();
      return b === null ? { kind: 'null' } : { kind: 'bool', value: b };
    }
    case MAJOR_TAG:
      throw new Error('tags are not used in a claim');
    default:
      throw new Error(`unsupported major type ${head >> 5}`);
  }
}

/** decodeValue reads one whole value from a byte span, refusing trailing bytes past it. */
function decodeValue(bytes: Uint8Array): CborNode {
  const r = new CborReader(bytes);
  const value = readValue(r, bytes);
  r.expectEnd();
  return value;
}

/**
 * slotValueAt is where a slot's value begins. inspectClaim computes this internally
 * (inspect.ts's own private valueStart) but reports only the key's offset and the value's
 * length, so it is re-derived here by position arithmetic over the exported reader — a stopgap
 * for ranke-ts inspect.ts, worth closing by exporting the value offset on InspectedSlot.
 */
function slotValueAt(bytes: Uint8Array, slot: InspectedSlot): number {
  const r = new CborReader(bytes.subarray(slot.at));
  r.readInt();
  return slot.at + r.position;
}

function renderSlot(bytes: Uint8Array, slot: InspectedSlot): RenderedSlot {
  try {
    const valueAt = slotValueAt(bytes, slot);
    const value = decodeValue(bytes.subarray(valueAt, valueAt + slot.length));
    return { ...slot, value, error: null };
  } catch (err) {
    return { ...slot, value: null, error: err instanceof Error ? err.message : String(err) };
  }
}

/**
 * inspectClaimBytes renders a claim's bytes by record: inspectClaim frames each record's slots
 * (name, offset, length) and every deviation from canonical, including for bytes it could only
 * partly frame — this only decodes each slot's own value bytes into a tree for display.
 */
export function inspectClaimBytes(bytes: Uint8Array): ClaimView {
  const inspection = inspectClaim(bytes);
  return {
    valid: inspection.valid,
    records: inspection.records.map((record) => ({
      kind: record.kind,
      path: record.path,
      at: record.at,
      slots: record.slots.map((slot) => renderSlot(bytes, slot)),
    })),
    deviations: inspection.deviations,
  };
}
