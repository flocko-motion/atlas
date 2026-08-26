/**
 * package: ui / shell
 * type:    view
 * job:     show one claim's raw CBOR, parsed into a tree, with its bytes downloadable
 * limits:  presentation; the read and the parse are core's (-> core/session, core/cbor)
 *
 * A main-pane tab, alongside the graph views: App.tsx shows this instead of the canvas when
 * the active tab is a CBOR tab rather than a view.
 */

import { useEffect, useMemo, useState } from 'react';
import { NodeKeyEdges } from '@flocko-motion/ranke';
import { claimBytesOf } from '../../core/claimBytes.ts';
import { inspectClaimBytes } from '../../core/cbor.ts';
import type { CborNode, RenderedRecord, RenderedSlot } from '../../core/cbor.ts';
import { fetchClaimBytes } from '../../core/session.ts';
import type { CborTabState } from '../../core/store.ts';
import { PaneTitle } from '../components/Field.tsx';
import { hexDump } from '../format.ts';

/** How many leading bytes a byte string shows before the rest waits behind a toggle. */
const BYTE_PREVIEW = 16;

/** keyLabel names a *generic* map key inline — a claim's own record slots use slotLabel below. */
function keyLabel(node: CborNode): string {
  switch (node.kind) {
    case 'int':
      return String(node.value);
    case 'text':
      return JSON.stringify(node.value);
    case 'bool':
      return String(node.value);
    case 'null':
      return 'null';
    case 'bytes':
      return `bytes(${node.value.length})`;
    case 'array':
      return `array(${node.items.length})`;
    case 'map':
      return `map(${node.entries.length})`;
  }
}

/** slotLabel names a record's own slot: the number the bytes carry, with its name where known. */
function slotLabel(slot: RenderedSlot): string {
  return slot.name ? `${slot.key} (${slot.name})` : String(slot.key);
}

/** CborBytes shows a byte string's length and a truncated hex preview, the rest a click away. */
function CborBytes({ bytes }: { bytes: Uint8Array }) {
  const head = bytes.subarray(0, BYTE_PREVIEW);
  const preview = [...head].map((b) => b.toString(16).padStart(2, '0')).join(' ');
  const truncated = bytes.length > BYTE_PREVIEW;
  return (
    <span className="cbor-bytes">
      bytes({bytes.length}) <code>{preview}{truncated ? '…' : ''}</code>
      {truncated ? (
        <details className="cbor-bytes-full">
          <summary>show all {bytes.length} bytes</summary>
          <pre className="content-body is-hex">{hexDump(bytes)}</pre>
        </details>
      ) : null}
    </span>
  );
}

/** CborScalar renders a leaf value — everything a map or array can hold apart from itself. */
function CborScalar({ node }: { node: CborNode }) {
  switch (node.kind) {
    case 'int':
      return <span className="cbor-int">{String(node.value)}</span>;
    case 'bool':
      return <span className="cbor-bool">{String(node.value)}</span>;
    case 'null':
      return <span className="cbor-null">null</span>;
    case 'text':
      return <span className="cbor-text">{JSON.stringify(node.value)}</span>;
    case 'bytes':
      return <CborBytes bytes={node.value} />;
    default:
      return null;
  }
}

/** CborValue renders one value: a leaf inline, a container indented and walked again. */
function CborValue({ node }: { node: CborNode }) {
  if (node.kind === 'array' || node.kind === 'map') {
    return (
      <div className="cbor-nested">
        <CborTree node={node} />
      </div>
    );
  }
  return <CborScalar node={node} />;
}

/** CborTree walks a map or an array; a leaf at the top is the whole claim being one value. */
function CborTree({ node }: { node: CborNode }) {
  if (node.kind === 'map') {
    return (
      <ul className="cbor-map">
        {node.entries.map((entry, i) => (
          <li key={i}>
            <span className="cbor-key">{keyLabel(entry.key)}</span>
            <CborValue node={entry.value} />
          </li>
        ))}
      </ul>
    );
  }
  if (node.kind === 'array') {
    return (
      <ul className="cbor-array">
        {node.items.map((item, i) => (
          <li key={i}>
            <CborValue node={item} />
          </li>
        ))}
      </ul>
    );
  }
  return <CborScalar node={node} />;
}

/** CborSlot renders one record slot: its label, then its value — or why it has none. */
function CborSlot({ record, slot }: { record: RenderedRecord; slot: RenderedSlot }) {
  // The edges slot is already rendered as its own named records below (-> CborRecord), so
  // dumping its raw array here a second time would just repeat the same bytes unlabelled.
  if (record.kind === 'node' && slot.key === NodeKeyEdges && slot.value?.kind === 'array') {
    return (
      <li>
        <span className="cbor-key">{slotLabel(slot)}</span>
        <span className="note">{slot.value.items.length} edge(s) — shown as records below</span>
      </li>
    );
  }
  return (
    <li>
      <span className="cbor-key">{slotLabel(slot)}</span>
      {slot.value ? <CborValue node={slot.value} /> : <span className="content-error">{slot.error}</span>}
    </li>
  );
}

/** CborRecord renders one node or edge record: its path, then each of its slots. */
function CborRecord({ record }: { record: RenderedRecord }) {
  return (
    <div className="cbor-record">
      <h3>{record.path}</h3>
      <ul className="cbor-map">
        {record.slots.map((slot, i) => (
          // Index, not slot.key: a malformed record can frame the same key twice
          // (-> cbor.test.ts), and inspectClaim reports rather than collapses that.
          <CborSlot key={i} record={record} slot={slot} />
        ))}
      </ul>
    </div>
  );
}

export function CborView({ tab }: { tab: CborTabState }) {
  const [status, setStatus] = useState<'loading' | 'ready' | 'error'>('loading');
  const [error, setError] = useState<string | null>(null);
  // Held together, and only ever set together: rendering the filename from tab.claimId while
  // the url still trails one effect run behind (switching straight to another CBOR tab) would
  // pair a fresh name with the previous claim's bytes for one frame.
  const [download, setDownload] = useState<{ url: string; claimId: string } | null>(null);

  // Fetched here rather than on open: a tab that is never brought forward costs no request,
  // and re-selecting one already read costs nothing either — the cache below is why.
  useEffect(() => {
    let cancelled = false;
    setStatus('loading');
    setError(null);
    fetchClaimBytes(tab.claimId, tab.scope).then(
      () => {
        if (!cancelled) setStatus('ready');
      },
      (err: unknown) => {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : String(err));
        setStatus('error');
      },
    );
    return () => {
      cancelled = true;
    };
  }, [tab.claimId, tab.scope]);

  // The bytes themselves never enter React state — this reads the module-scope cache fresh
  // and holds only the derived, much smaller tree.
  const view = useMemo(() => {
    if (status !== 'ready') return null;
    const bytes = claimBytesOf(tab.claimId);
    return bytes ? inspectClaimBytes(bytes) : null;
  }, [status, tab.claimId]);

  // A resource with a lifetime (revoke it, or it leaks) belongs in an effect, not a memo:
  // StrictMode double-invokes a memo, and React may discard and recompute one at any time.
  useEffect(() => {
    if (status !== 'ready') return;
    const bytes = claimBytesOf(tab.claimId);
    if (!bytes) return;
    const url = URL.createObjectURL(new Blob([new Uint8Array(bytes)], { type: 'application/cbor' }));
    setDownload({ url, claimId: tab.claimId });
    return () => {
      URL.revokeObjectURL(url);
      setDownload(null);
    };
  }, [status, tab.claimId]);

  return (
    <div className="pane cbor-view">
      <PaneTitle hint={tab.scope?.name ?? 'no scope'}>claim CBOR</PaneTitle>
      <p className="note">
        <code className="claim-id">{tab.claimId}</code>
      </p>

      {status === 'loading' ? <p className="note">reading…</p> : null}
      {status === 'error' ? <p className="note content-error">{error}</p> : null}

      {download ? (
        <p className="row">
          <a className="btn" href={download.url} download={`${download.claimId}.cbor`}>
            download {download.claimId}.cbor
          </a>
        </p>
      ) : null}

      {view && !view.valid ? (
        <p className="note content-error">not a canonical claim — see the deviations below</p>
      ) : null}

      {view?.deviations.length ? (
        <ul className="cbor-deviations">
          {view.deviations.map((d, i) => (
            <li key={i} className="content-error">
              {d.path}: {d.message} (offset {d.at})
            </li>
          ))}
        </ul>
      ) : null}

      {view?.records.length ? (
        view.records.map((record) => <CborRecord key={record.path} record={record} />)
      ) : view && view.deviations.length === 0 ? (
        <p className="note">no records could be framed from these bytes</p>
      ) : null}
    </div>
  );
}
