/**
 * @layer ui
 * @description Status bar showing node/edge counts and level breakdown.
 * @depends core/hooks
 * @must-not Contain business logic. Import from graph/.
 */

import { useAppStore } from '../../core/hooks';
import './StatusBar.css';

export function StatusBar() {
  const nodeCount = useAppStore((s) => s.nodeCount);
  const edgeCount = useAppStore((s) => s.edgeCount);
  const viewMode = useAppStore((s) => s.viewMode);
  const nodes = useAppStore((s) => s.nodes);

  let l0 = 0, l1 = 0, l2 = 0;
  for (const n of nodes.values()) {
    if (n.level === 0) l0++;
    else if (n.level === 1) l1++;
    else l2++;
  }

  return (
    <div className="statusbar">
      <span>{nodeCount} nodes</span>
      <span className="statusbar-sep">|</span>
      <span>{edgeCount} edges</span>
      <span className="statusbar-sep">|</span>
      <span className="statusbar-badge">
        <span className="dot" style={{ background: 'var(--l0-color)' }} />
        L0: {l0}
      </span>
      <span className="statusbar-badge">
        <span className="dot" style={{ background: 'var(--l1-color)' }} />
        L1: {l1}
      </span>
      <span className="statusbar-badge">
        <span className="dot" style={{ background: 'var(--l2-entity-color)' }} />
        L2: {l2}
      </span>
      <span className="statusbar-sep">|</span>
      <span>{viewMode}</span>
    </div>
  );
}
