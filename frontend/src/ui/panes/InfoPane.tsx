/**
 * package: ui / panes
 * type:    view
 * job:     answer about whatever is selected — an edge, a claim, or the graph
 * limits:  presentation; what is selected lives in the store (-> core/store)
 *
 * One pane rather than three tabs to choose between: the question "what is this" has one
 * answer at a time, and which thing it is about is the selection rather than a tab.
 */

import { edgeDetail } from '../../core/session.ts';
import { useExplorer } from '../../core/store.ts';
import { Empty, KeyValue } from '../components/Field.tsx';
import { GraphPane } from './GraphPane.tsx';
import { SelectionPane } from './SelectionPane.tsx';

export function InfoPane() {
  const selected = useExplorer((s) => s.selection.selected);
  const selectedEdge = useExplorer((s) => s.selection.selectedEdge);

  if (selectedEdge) return <EdgeInfo edgeKey={selectedEdge} />;
  if (selected) return <SelectionPane />;
  return <GraphPane />;
}

/**
 * EdgeInfo shows one edge: its type, and the claims at either end. The direction is the
 * substance — an edge points from a claim to what it cites — so both ends are named and the
 * arrow between them is spelled out rather than implied.
 */
function EdgeInfo({ edgeKey }: { edgeKey: string }) {
  const detail = edgeDetail(edgeKey);
  const select = useExplorer((s) => s.select);

  if (!detail) return <Empty>That edge is no longer in the graph.</Empty>;

  return (
    <div className="pane">
      <KeyValue rows={[['type', detail.edgeType || '—']]} />

      <h2>cites from</h2>
      <EdgeEnd
        id={detail.from}
        label={detail.fromLabel}
        claimType={detail.fromType}
        onSelect={() => select(detail.from)}
      />

      <h2>to</h2>
      <EdgeEnd
        id={detail.to}
        label={detail.toLabel}
        claimType={detail.toType}
        onSelect={() => select(detail.to)}
      />

      <p className="note">
        The edge belongs to the claim it points from — that claim created it — so its
        provenance is read there.
      </p>
    </div>
  );
}

/** EdgeEnd names one end of an edge and offers to select it. */
function EdgeEnd({
  id,
  label,
  claimType,
  onSelect,
}: {
  id: string;
  label: string;
  claimType: string;
  onSelect: () => void;
}) {
  return (
    <div className="edge-end">
      <span className="ref-type">{claimType || '—'}</span>
      <button type="button" className="ref-id" onClick={onSelect} title={id}>
        {label || `${id.slice(0, 12)}…`}
      </button>
    </div>
  );
}
