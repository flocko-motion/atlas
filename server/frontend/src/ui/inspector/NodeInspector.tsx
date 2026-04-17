/**
 * @layer ui
 * @description Inspector panel showing selected node detail.
 * @depends core/hooks, core/actions, core/selectors, core/colors
 * @must-not Contain business logic. Import from graph/.
 */

import { useMemo, useState } from 'react';
import { useAppStore } from '../../core/hooks';
import { selectNode, selectEdge, setViewMode, setGraphMode, previewRun, clearRunPreview, purgeActiveRun } from '../../core/actions';
import { nodeEdges } from '../../core/selectors';
import { nodeColor, edgeColor } from '../../core/colors';
import type { Node, Edge } from '../../core/types/nodes';
import './NodeInspector.css';

function fmt(d: Date | null): string {
  return d ? d.toISOString().slice(0, 19).replace('T', ' ') : '-';
}

function humanSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  const units = ['KB', 'MB', 'GB'];
  let i = -1;
  let b = bytes;
  do { b /= 1024; i++; } while (b >= 1024 && i < units.length - 1);
  return `${b.toFixed(1)} ${units[i]}`;
}

function MetaRow({ label, value }: { label: string; value: string }) {
  if (value === '-' || value === '' || value === 'null') return null;
  return (
    <div className="inspector-meta-row">
      <span className="inspector-meta-key">{label}</span>
      <span className="inspector-meta-value">{value}</span>
    </div>
  );
}

function EdgeRow({ edge, node, currentNodeId }: { edge: Edge; node: Node | undefined; currentNodeId: string }) {
  const dir = edge.sourceNodeId === currentNodeId ? 'outgoing' : 'incoming';
  const otherId = edge.sourceNodeId === currentNodeId ? edge.targetNodeId : edge.sourceNodeId;
  return (
    <div
      className="inspector-edge-row"
      onClick={() => {
        selectEdge(edge.id);
        if (node) selectNode(otherId);
      }}
    >
      <span className="inspector-edge-type" style={{ color: edgeColor(edge) }}>
        {edge.type}
      </span>
      <span className="inspector-edge-dir">{dir}</span>
      <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
        {node?.title ?? node?.contentType ?? otherId.slice(0, 8)}
      </span>
      {edge.runId && (
        <span className="inspector-edge-run" title={edge.runId}>
          run:{edge.runId.slice(0, 8)}
        </span>
      )}
    </div>
  );
}

export function NodeInspector() {
  const selectedNodeIds = useAppStore((s) => s.selectedNodeIds);
  const nodes = useAppStore((s) => s.nodes);

  const primaryId = selectedNodeIds.values().next().value ?? null;
  const node = primaryId ? nodes.get(primaryId) ?? null : null;

  const edges = useMemo(() => {
    if (!primaryId) return [];
    return nodeEdges(primaryId);
  }, [primaryId]);

  if (!node) {
    return <div className="inspector-empty">No node selected</div>;
  }

  return (
    <div className="inspector">
      <span
        className="inspector-level-badge"
        style={{ background: nodeColor(node) }}
      >
        L{node.level} {node.contentClass}/{node.contentType}
      </span>

      <div className="inspector-meta">
        <MetaRow label="Title" value={node.title ?? '-'} />
        <MetaRow label="ID" value={node.id} />
        <MetaRow label="Encoding" value={`${node.encodingClass} / ${node.encodingFormat}`} />
        <MetaRow label="SHA-256" value={node.contentSha256.slice(0, 16) + '...'} />
        <MetaRow label="Size" value={humanSize(node.contentLen)} />
        <MetaRow label="Created" value={fmt(node.createdAt)} />
        <MetaRow label="Artifact date" value={fmt(node.artifactCreatedAt)} />
        <MetaRow label="Artifact blur" value={node.artifactCreatedAtBlur || '-'} />
        <MetaRow label="Origin" value={node.origin ?? '-'} />
        <MetaRow label="Original name" value={node.originalName ?? '-'} />
        <MetaRow label="Valid from" value={fmt(node.validFrom)} />
        <MetaRow label="Valid from blur" value={node.validFromBlur || '-'} />
        <MetaRow label="Valid until" value={fmt(node.validUntil)} />
        <MetaRow label="Valid until blur" value={node.validUntilBlur || '-'} />
        <MetaRow label="Confidence" value={node.confidence !== null ? String(node.confidence) : '-'} />
      </div>

      {node.content && (
        <>
          <span className="inspector-section-title">Content</span>
          <div className="inspector-content">{node.content}</div>
        </>
      )}

      {edges.length > 0 && (
        <>
          <span className="inspector-section-title">Edges ({edges.length})</span>
          <div className="inspector-edges">
            {edges.slice(0, 20).map((edge) => (
              <EdgeRow
                key={edge.id}
                edge={edge}
                node={nodes.get(
                  edge.sourceNodeId === node.id ? edge.targetNodeId : edge.sourceNodeId,
                )}
                currentNodeId={node.id}
              />
            ))}
            {edges.length > 20 && (
              <span style={{ fontSize: 11, color: 'var(--text-muted)', padding: '4px 6px' }}>
                ...and {edges.length - 20} more
              </span>
            )}
          </div>
        </>
      )}

      <RunActions edges={edges} />

      <button
        className="inspector-btn"
        onClick={() => {
          setViewMode('graph');
          setGraphMode('provenance');
        }}
      >
        Show Provenance
      </button>
    </div>
  );
}

function RunActions({ edges }: { edges: Edge[] }) {
  const activeRunId = useAppStore((s) => s.activeRunId);
  const highlightedNodeIds = useAppStore((s) => s.highlightedNodeIds);
  const [purging, setPurging] = useState(false);

  const runIds = useMemo(() => {
    const ids = new Set<string>();
    for (const e of edges) {
      if (e.runId) ids.add(e.runId);
    }
    return [...ids];
  }, [edges]);

  if (runIds.length === 0) return null;

  async function handlePurge() {
    if (!highlightedNodeIds) return;
    if (!confirm(`Purge ${highlightedNodeIds.size} nodes and all derivatives?`)) return;
    setPurging(true);
    try {
      const result = await purgeActiveRun();
      if (result) {
        alert(`Purged: ${result.nodes_deleted} nodes, ${result.edges_deleted} edges`);
      }
    } finally {
      setPurging(false);
    }
  }

  return (
    <>
      <span className="inspector-section-title">Runs</span>
      <div className="inspector-runs">
        {runIds.map((id) => (
          <div key={id} className="inspector-run-row">
            <span
              className={`inspector-run-id inspector-run-link${activeRunId === id ? ' inspector-run-active' : ''}`}
              title={id}
              onClick={() => activeRunId === id ? clearRunPreview() : previewRun(id)}
            >
              {id.slice(0, 12)}...
            </span>
          </div>
        ))}
      </div>

      {activeRunId && highlightedNodeIds && (
        <div className="inspector-run-detail">
          <div className="inspector-run-detail-header">
            <span style={{ fontSize: 11 }}>{highlightedNodeIds.size} nodes affected</span>
            <button
              className="inspector-btn"
              onClick={clearRunPreview}
              style={{ fontSize: 10, padding: '2px 6px' }}
            >
              Clear
            </button>
          </div>
          <button
            className="inspector-btn inspector-btn-danger"
            disabled={purging}
            onClick={handlePurge}
          >
            {purging ? 'Purging...' : `Purge ${highlightedNodeIds.size} nodes`}
          </button>
        </div>
      )}
    </>
  );
}
