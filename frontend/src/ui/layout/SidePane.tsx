/**
 * @layer ui
 * @description Side pane with tabbed Inspector, Filters, Creator, Workers panels.
 * @depends ui/inspector/NodeInspector, ui/filters/FiltersPanel
 * @must-not Contain business logic. Import from graph/.
 */

import { Tabs } from '@mantine/core';
import { NodeInspector } from '../inspector/NodeInspector';
import { FiltersPanel } from '../filters/FiltersPanel';
import './SidePane.css';

export function SidePane() {
  return (
    <div className="sidepane">
      <Tabs defaultValue="inspector">
        <Tabs.List>
          <Tabs.Tab value="inspector">Inspector</Tabs.Tab>
          <Tabs.Tab value="filters">Filters</Tabs.Tab>
          <Tabs.Tab value="creator">Creator</Tabs.Tab>
          <Tabs.Tab value="workers">Workers</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="inspector">
          <NodeInspector />
        </Tabs.Panel>

        <Tabs.Panel value="filters">
          <FiltersPanel />
        </Tabs.Panel>

        <Tabs.Panel value="creator">
          <div className="sidepane-placeholder">Creator — coming in Phase 2</div>
        </Tabs.Panel>

        <Tabs.Panel value="workers">
          <div className="sidepane-placeholder">Workers — coming in Phase 2</div>
        </Tabs.Panel>
      </Tabs>
    </div>
  );
}
