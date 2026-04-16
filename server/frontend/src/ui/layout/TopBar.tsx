/**
 * @layer ui
 * @description Top bar with title, health indicator, and view toggle.
 * @depends core/hooks, core/actions
 * @must-not Contain business logic. Import from graph/.
 */

import { useAppStore } from '../../core/hooks';
import { setViewMode } from '../../core/actions';
import type { ViewMode } from '../../core/types/graph';
import './TopBar.css';

export function TopBar() {
  const isConnected = useAppStore((s) => s.isConnected);
  const viewMode = useAppStore((s) => s.viewMode);

  const healthClass = isConnected
    ? 'topbar-health topbar-health--connected'
    : 'topbar-health topbar-health--disconnected';

  return (
    <div className="topbar">
      <span className="topbar-title">RankeDB Explorer</span>
      <div className={healthClass} title={isConnected ? 'Connected' : 'Disconnected'} />

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
