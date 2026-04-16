/**
 * @layer ui
 * @description Side pane with tabbed Inspector, Filters, Creator, Workers, Dev panels.
 * @depends ui/inspector/NodeInspector, ui/filters/FiltersPanel
 * @must-not Contain business logic. Import from graph/.
 */

import { useState } from 'react';
import { Tabs } from '@mantine/core';
import { NodeInspector } from '../inspector/NodeInspector';
import { FiltersPanel } from '../filters/FiltersPanel';
import { loadMockData } from '../../core/actions';
import './SidePane.css';

function DevPanel() {
  const [mockCount, setMockCount] = useState(500);

  return (
    <div className="sidepane-section">
      <h4>Mock Data</h4>
      <div className="dev-mock-controls">
        <input
          type="number"
          value={mockCount}
          min={10}
          max={100000}
          onChange={(e) => setMockCount(Number(e.target.value))}
        />
        <button onClick={() => loadMockData(mockCount)}>Generate</button>
      </div>
      <p className="dev-hint">Replaces current data with randomly generated graph.</p>
    </div>
  );
}

export function SidePane() {
  return (
    <div className="sidepane">
      <Tabs defaultValue="inspector">
        <Tabs.List>
          <Tabs.Tab value="inspector">Inspector</Tabs.Tab>
          <Tabs.Tab value="filters">Filters</Tabs.Tab>
          <Tabs.Tab value="creator">Creator</Tabs.Tab>
          <Tabs.Tab value="workers">Workers</Tabs.Tab>
          <Tabs.Tab value="dev">Dev</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="inspector">
          <NodeInspector />
        </Tabs.Panel>

        <Tabs.Panel value="filters">
          <FiltersPanel />
        </Tabs.Panel>

        <Tabs.Panel value="creator">
          <div className="sidepane-placeholder">Creator — coming soon</div>
        </Tabs.Panel>

        <Tabs.Panel value="workers">
          <div className="sidepane-placeholder">Workers — coming soon</div>
        </Tabs.Panel>

        <Tabs.Panel value="dev">
          <DevPanel />
        </Tabs.Panel>
      </Tabs>
    </div>
  );
}
