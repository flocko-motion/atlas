/**
 * @layer ui
 * @description Top bar with title, health indicator, and view toggle.
 * @depends core/hooks, core/actions
 * @must-not Contain business logic. Import from graph/.
 */

import { useAppStore } from '../../core/hooks';
import { setViewMode, setLevelFilter } from '../../core/actions';
import type { ViewMode } from '../../core/types/graph';
import './TopBar.css';

const LEVEL_COLORS = ['var(--l0-color)', 'var(--l1-color)', 'var(--l2-entity-color)'];

export function TopBar() {
  const isConnected = useAppStore((s) => s.isConnected);
  const viewMode = useAppStore((s) => s.viewMode);
  const levels = useAppStore((s) => s.filters.levels);

  const healthClass = isConnected
    ? 'topbar-health topbar-health--connected'
    : 'topbar-health topbar-health--disconnected';

  function toggleLevel(level: number) {
    const next = new Set(levels);
    if (next.has(level)) {
      next.delete(level);
    } else {
      next.add(level);
    }
    setLevelFilter(next);
  }

  return (
    <div className="topbar">
      <span className="topbar-title">RankeDB Explorer</span>
      <div className={healthClass} title={isConnected ? 'Connected' : 'Disconnected'} />

      <div className="topbar-separator" />

      <div className="topbar-levels">
        {[0, 1, 2].map((level) => (
          <button
            key={level}
            className={`topbar-level-btn${levels.has(level) ? ' active' : ''}`}
            style={{ borderColor: levels.has(level) ? LEVEL_COLORS[level] : undefined }}
            onClick={() => toggleLevel(level)}
          >
            L{level}
          </button>
        ))}
      </div>

      <div className="topbar-separator" />

      <div className="topbar-views">
        {(['timeline', 'graph'] as ViewMode[]).map((mode) => (
          <button
            key={mode}
            className={viewMode === mode ? 'active' : ''}
            onClick={() => setViewMode(mode)}
          >
            {mode.charAt(0).toUpperCase() + mode.slice(1)}
          </button>
        ))}
      </div>
    </div>
  );
}
