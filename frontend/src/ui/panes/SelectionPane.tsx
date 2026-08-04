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

import { claimDetail } from '../../core/session.ts';
import { useExplorer } from '../../core/store.ts';
import { Empty, KeyValue } from '../components/Field.tsx';

function bytes(size: number | undefined): string {
  if (size === undefined) return '—';
  if (size < 1024) return `${size} B`;
  if (size < 1048576) return `${(size / 1024).toFixed(1)} KiB`;
  return `${(size / 1048576).toFixed(1)} MiB`;
}

export function SelectionPane() {
  const selected = useExplorer((s) => s.selection.selected);
  const detail = selected ? claimDetail(selected) : null;

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
          ['content', `${bytes(detail.contentSize)}${detail.encoding ? ` · ${detail.encoding}` : ''}`],
          ['degree', detail.degree.toLocaleString('en-US')],
          ['cited by', detail.citedBy.toLocaleString('en-US')],
        ]}
      />

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
