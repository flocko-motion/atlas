/**
 * @layer ui
 * @description Main content area showing a scrollable node list for Phase 1.
 * @depends core/hooks, core/actions, core/colors
 * @must-not Contain business logic beyond display formatting. Import from graph/.
 */

import { useMemo } from 'react';
import { useAppStore } from '../../core/hooks';
import { selectNode, temporalPosition } from '../../core/actions';
import { nodeColor } from '../../core/colors';
import type { Node } from '../../core/types/nodes';
import './MainPane.css';

function formatDate(d: Date): string {
  return d.toISOString().slice(0, 10);
}

function levelLabel(level: number): string {
  return `L${level}`;
}

function confidenceLabel(c: number | null): string {
  if (c === null) return '';
  return c.toFixed(2);
}

function confidenceColor(c: number | null): string {
  if (c === null) return 'var(--text-muted)';
  if (c < 0) return '#ef4444';
  if (c < 0.5) return '#f59e0b';
  return '#10b981';
}

export function MainPane() {
  const nodes = useAppStore((s) => s.nodes);
  const filters = useAppStore((s) => s.filters);
  const selectedNodeIds = useAppStore((s) => s.selectedNodeIds);

  const sortedNodes = useMemo(() => {
    const result: Node[] = [];
    for (const node of nodes.values()) {
      if (!filters.levels.has(node.level)) continue;
      if (filters.contentClasses.size > 0 && !filters.contentClasses.has(node.contentClass)) continue;
      if (filters.dateRange.from || filters.dateRange.to) {
        const d = temporalPosition(node);
        if (filters.dateRange.from && d < filters.dateRange.from) continue;
        if (filters.dateRange.to && d > filters.dateRange.to) continue;
      }
      if (node.confidence !== null && node.confidence < filters.minConfidence) continue;
      result.push(node);
    }
    result.sort((a, b) => b.createdAt.getTime() - a.createdAt.getTime());
    return result;
  }, [nodes, filters]);

  if (nodes.size === 0) {
    return (
      <div className="mainpane">
        <div className="mainpane-empty">
          No data loaded. Click "Generate" in the top bar to create mock data.
        </div>
      </div>
    );
  }

  return (
    <div className="mainpane">
      <div className="mainpane-header">
        {sortedNodes.length} nodes (sorted by created date, newest first)
      </div>
      <div className="mainpane-list">
        {sortedNodes.map((node) => (
          <div
            key={node.id}
            className={`node-row${selectedNodeIds.has(node.id) ? ' selected' : ''}`}
            onClick={() => selectNode(node.id)}
          >
            <span
              className="node-row-level"
              style={{ background: nodeColor(node) }}
            >
              {levelLabel(node.level)}
            </span>
            <span className="node-row-class">
              {node.contentClass}/{node.contentType}
            </span>
            <span className="node-row-content">
              {node.content ?? '(no content)'}
            </span>
            <span
              className="node-row-confidence"
              style={{ color: confidenceColor(node.confidence) }}
            >
              {confidenceLabel(node.confidence)}
            </span>
            <span className="node-row-date">
              {formatDate(temporalPosition(node))}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
