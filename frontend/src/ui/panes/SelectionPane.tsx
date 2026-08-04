/**
 * package: ui / panes
 * type:    view
 * job:     show the claim the user clicked
 * limits:  presentation; selection state is the store's (-> core/store)
 *
 * The selection pane: the claim the user clicked.
 *
 * It reads `selection.selected` and never `selection.hovered` — hover drives only a
 * cheap preview, so sweeping the pointer across a thousand nodes must not re-render
 * anything here.
 */

import { useEffect } from 'react';
import { CONTENT_LIMIT, claimDetail, fetchContent } from '../../core/session.ts';
import { useExplorer } from '../../core/store.ts';
import { Empty, KeyValue } from '../components/Field.tsx';
import { asText, hexDump, isTextual } from '../format.ts';

function bytes(size: number | undefined): string {
  if (size === undefined) return '—';
  if (size < 1024) return `${size} B`;
  if (size < 1048576) return `${(size / 1024).toFixed(1)} KiB`;
  return `${(size / 1048576).toFixed(1)} MiB`;
}

/**
 * ContentBlock shows the claim's bytes, read as text where the encoding says they are text
 * and as a hex dump where it does not. A size on its own says nothing about a note.
 */
function ContentBlock() {
  const content = useExplorer((s) => s.content);
  const selected = useExplorer((s) => s.selection.selected);
  if (!content || content.id !== selected) return null;

  return (
    <>
      <h2>content</h2>
      {contentBody(content)}
    </>
  );
}

/** contentBody is what to show for each state the read can be in. */
function contentBody(content: NonNullable<ReturnType<typeof useExplorer.getState>['content']>) {
  switch (content.state) {
    case 'none':
      return <Empty>This claim carries no content.</Empty>;
    case 'loading':
      return <p className="note">reading {bytes(content.size)}…</p>;
    case 'too-large':
      return (
        <p className="note">
          {bytes(content.size)} — too much to show. The limit is{' '}
          {bytes(CONTENT_LIMIT)}, and the bytes are left where they are rather than fetched.
        </p>
      );
    case 'error':
      return <p className="note content-error">{content.error}</p>;
    case 'ready': {
      const data = content.bytes ?? new Uint8Array();
      const text = isTextual(content.encoding);
      return (
        <>
          <p className="note content-about">
            {bytes(data.length)} · {content.encoding ?? 'no encoding declared'}
            {text ? '' : ' — shown as bytes, the encoding naming no way to read them'}
          </p>
          <pre className={`content-body${text ? '' : ' is-hex'}`}>
            {text ? asText(data) : hexDump(data)}
          </pre>
        </>
      );
    }
  }
}

export function SelectionPane() {
  const selected = useExplorer((s) => s.selection.selected);
  const detail = selected ? claimDetail(selected) : null;

  // Content is read on selection: it is the substance of most claims, and a size alone
  // answers nothing. Immutability makes the cache behind this safe, so re-selecting is free.
  useEffect(() => {
    if (selected) void fetchContent(selected);
  }, [selected]);

  if (!detail) return <Empty>Click a claim to inspect it.</Empty>;

  return (
    <div className="pane">
      <KeyValue
        rows={[
          ['id', <code className="claim-id">{detail.id}</code>],
          ['type', detail.claimType || '—'],
          ['label', detail.label || '—'],
          ['contribution', detail.contribution.toLocaleString('en-US')],
          ['created', new Date(detail.createdAt).toISOString().replace('T', ' ').slice(0, 19)],
          ['degree', detail.degree.toLocaleString('en-US')],
          ['cited by', detail.citedBy.toLocaleString('en-US')],
        ]}
      />

      <ContentBlock />

      <h2>references</h2>
      {detail.references.length === 0 ? (
        <Empty>An initial node — it references nothing.</Empty>
      ) : (
        <ul className="refs">
          {detail.references.slice(0, 40).map((ref) => (
            <li key={`${ref.type}:${ref.id}`}>
              <span className="ref-type">{ref.type}</span>
              <button
                type="button"
                className="ref-id"
                onClick={() => useExplorer.getState().select(ref.id)}
                title={ref.id}
              >
                {ref.id.slice(0, 12)}…
              </button>
            </li>
          ))}
        </ul>
      )}
      {detail.references.length > 40 ? (
        <p className="note">
          {(detail.references.length - 40).toLocaleString('en-US')} further references not listed.
        </p>
      ) : null}
    </div>
  );
}
