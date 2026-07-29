/**
 * package: ui / shell
 * type:    view
 * job:     the shell — header, footer, and the tabbed main and side panes
 * limits:  layout only; each pane owns its content (-> ui/panes)
 *
 * Both panes tab independently. The canvas host is mounted once and never unmounted by a
 * re-render: Sigma and the graph live outside React, and this holds only a ref to the
 * host and ids from the store.
 */

import { useEffect, useRef } from 'react';
import { activeView, useExplorer } from '../../core/store.ts';
import type { SidePane } from '../../core/store.ts';
import { setOnLoaded } from '../../core/session.ts';
import { applyViewSettings, mount, refreshSelection } from '../../render/renderer.ts';
import { Tabs } from '../components/Tabs.tsx';
import type { TabItem } from '../components/Tabs.tsx';
import { ConnectionsPane } from '../panes/ConnectionsPane.tsx';
import { QueryPane } from '../panes/QueryPane.tsx';
import { ViewPane } from '../panes/ViewPane.tsx';
import { GraphPane } from '../panes/GraphPane.tsx';
import { LogPane } from '../panes/LogPane.tsx';
import { SelectionPane } from '../panes/SelectionPane.tsx';
import { Footer } from './Footer.tsx';
import { Header } from './Header.tsx';

const SIDE_TABS: { id: SidePane; label: string }[] = [
  { id: 'query', label: 'Query' },
  { id: 'server', label: 'Server' },
  { id: 'view', label: 'View' },
  { id: 'selection', label: 'Selection' },
  { id: 'graph', label: 'Graph' },
  { id: 'log', label: 'Log' },
];

/**
 * The hover preview: label and type, nothing heavier, and driven by the debounced
 * `preview` rather than by `hovered`. The detail pane reads `selected` only, so a
 * pointer sweep cannot make it re-render.
 */
function HoverPreviewChip() {
  const preview = useExplorer((s) => s.preview);
  if (!preview) return null;
  return (
    <div className="hover-preview" role="status">
      <span className="hover-preview-type">{preview.claimType}</span>
      <span className="hover-preview-label">{preview.label}</span>
    </div>
  );
}

/**
 * The loading overlay. A load stages its work and yields between stages, so this
 * actually updates while it runs — the point being that a slow load looks like a slow
 * load rather than a broken app.
 */
function LoadingOverlay() {
  const busy = useExplorer((s) => s.status.busy);
  const progress = useExplorer((s) => s.status.progress);
  if (!busy) return null;
  return (
    <div className="loading" role="status" aria-live="polite">
      <div className="loading-card">
        <span className="loading-stage">{busy}…</span>
        <div className="loading-track">
          <div
            className={`loading-bar${progress === null ? ' is-indeterminate' : ''}`}
            style={progress === null ? undefined : { width: `${Math.round(progress * 100)}%` }}
          />
        </div>
        {progress !== null ? (
          <span className="loading-pct">{Math.round(progress * 100)}%</span>
        ) : null}
      </div>
    </div>
  );
}

function SidePaneBody({ pane }: { pane: SidePane }) {
  switch (pane) {
    case 'query':
      return <QueryPane />;
    case 'server':
      return <ConnectionsPane />;
    case 'view':
      return <ViewPane />;
    case 'selection':
      return <SelectionPane />;
    case 'graph':
      return <GraphPane />;
    case 'log':
      return <LogPane />;
  }
}

export function App() {
  const host = useRef<HTMLDivElement | null>(null);
  const views = useExplorer((s) => s.views);
  const activeViewId = useExplorer((s) => s.activeViewId);
  const view = useExplorer(activeView);
  const sidePane = useExplorer((s) => s.sidePane);
  const setSidePane = useExplorer((s) => s.setSidePane);
  const activateView = useExplorer((s) => s.activateView);
  const busy = useExplorer((s) => s.status.busy);
  const closeView = useExplorer((s) => s.closeView);

  // Mount the renderer once. Not a dependency of any state: a re-render must not
  // recreate it, and StrictMode's double-invoke is harmless because `mount` is
  // idempotent for a given container.
  useEffect(() => {
    if (host.current) mount(host.current);
    // Core stages a load and hands the finished graph back through this hook, so it
    // never has to import the renderer.
    setOnLoaded(refreshSelection);
  }, []);

  // A view switch is a reducer swap plus a refresh — never a rebuild.
  useEffect(() => {
    applyViewSettings(view);
    refreshSelection();
  }, [activeViewId, view?.layout, view?.edges]);

  const viewTabs: TabItem[] = views.map((v) => ({
    id: v.id,
    label: v.label,
    hint: v.layout,
    closable: views.length > 1,
  }));

  return (
    <div className="app">
      <Header />

      <main className="main">
        {views.length > 0 ? (
          <Tabs
            ariaLabel="Graph views"
            items={viewTabs}
            activeId={activeViewId}
            onSelect={activateView}
            onClose={closeView}
          />
        ) : null}
        <div className="canvas-host" ref={host} />
        <HoverPreviewChip />
        <LoadingOverlay />
        {views.length === 0 && !busy ? (
          <div className="canvas-overlay">
            <p>Set a limit in the Query tab and press run.</p>
          </div>
        ) : null}
      </main>

      <aside className="side">
        <Tabs
          ariaLabel="Tooling and details"
          items={SIDE_TABS.map((t) => ({ id: t.id, label: t.label }))}
          activeId={sidePane}
          onSelect={(id) => setSidePane(id as SidePane)}
        />
        <div className="side-body">
          <SidePaneBody pane={sidePane} />
        </div>
      </aside>

      <Footer />
    </div>
  );
}
