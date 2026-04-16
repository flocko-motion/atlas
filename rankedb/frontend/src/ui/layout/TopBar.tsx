/**
 * @layer ui
 * @description Top bar with title, health indicator, mock controls, and view toggle.
 * @depends core/hooks, core/actions
 * @must-not Contain business logic. Import from graph/.
 */

import { useState } from 'react';
import { useAppStore } from '../../core/hooks';
import { loadMockData, setViewMode } from '../../core/actions';
import type { ViewMode } from '../../core/types/graph';
import './TopBar.css';

export function TopBar() {
  const isConnected = useAppStore((s) => s.isConnected);
  const viewMode = useAppStore((s) => s.viewMode);
  const useMockData = useAppStore((s) => s.useMockData);
  const [mockCount, setMockCount] = useState(500);

  const healthClass = isConnected
    ? 'topbar-health topbar-health--connected'
    : 'topbar-health topbar-health--disconnected';

  return (
    <div className="topbar">
      <span className="topbar-title">RankeDB Explorer</span>
      <div className={healthClass} title={isConnected ? 'Connected' : 'Disconnected'} />

      <div className="topbar-separator" />

      <div className="topbar-mock">
        <label>
          <input
            type="checkbox"
            checked={useMockData}
            readOnly
          />{' '}
          Mock data
        </label>
        <input
          type="number"
          value={mockCount}
          min={10}
          max={100000}
          onChange={(e) => setMockCount(Number(e.target.value))}
        />
        <button onClick={() => loadMockData(mockCount)}>Generate</button>
      </div>

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
