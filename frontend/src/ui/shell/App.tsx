/**
 * package: ui / shell
 * type:    view
 * job:     the shell — header, footer, and the tabbed main and side panes
 * limits:  layout only; each pane owns its content (-> ui/panes)
 *
 * The canvas host is mounted once and never unmounted by a re-render: Sigma and the graph live
 * outside React, and this holds only a ref and ids.
 */

import { useEffect, useRef } from 'react';
import { activeView, useExplorer } from '../../core/store.ts';
import type { Notice, ScopesState, SidePane, ViewState } from '../../core/store.ts';
import { ARCHIVE_SCOPE } from '../../core/scope.ts';
import {
  axisWidth,
  discoverScopes,
  lensFor,
  setOnLoaded,
  setOnShowAll,
  stretchX,
  timelineExtent,
} from '../../core/session.ts';
import { useConnections } from '../../core/connections.ts';
import {
  anchorAt,
  applyViewSettings,
  axisSpanOnScreen,
  canvasWidth,
  graphXAt,
  hideLens,
  mount,
  mountLens,
  onRender,
  onZoom,
  pinExtent,
  refreshSelection,
  repaint,
  resetCamera,
  unpinExtent,
  showLens,
  shownGraph,
} from '../../render/renderer.ts';
import { Tabs } from '../components/Tabs.tsx';
import type { TabItem } from '../components/Tabs.tsx';
import { ConnectionsPane } from '../panes/ConnectionsPane.tsx';
import { QueryPane } from '../panes/QueryPane.tsx';
import { ViewPane } from '../panes/ViewPane.tsx';
import { InfoPane } from '../panes/InfoPane.tsx';
import { LogPane } from '../panes/LogPane.tsx';
import { Footer } from './Footer.tsx';
import { Header } from './Header.tsx';
import { TimeCursor } from './TimeCursor.tsx';
import { TimeRuler } from './TimeRuler.tsx';
import { ZoomBox } from './ZoomBox.tsx';

const SIDE_TABS: { id: SidePane; label: string }[] = [
  { id: 'info', label: 'Info' },
  { id: 'query', label: 'Query' },
  { id: 'view', label: 'View' },
  { id: 'server', label: 'Server' },
  { id: 'log', label: 'Log' },
];

/**
 * The hover preview, driven by the debounced `preview` rather than `hovered`, so a pointer sweep
 * cannot make the detail pane re-render.
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
 * The loading overlay. A load yields between stages, so this updates while it runs — a slow load
 * should look slow rather than broken.
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
    case 'info':
      return <InfoPane />;
    case 'log':
      return <LogPane />;
  }
}

/**
 * EmptyCanvas explains a canvas with nothing on it: an empty archive, a refused read and a view
 * admitting nothing look identical, and all look broken.
 */
function EmptyCanvas() {
  const status = useExplorer((s) => s.status);
  const notice = useExplorer((s) => s.notice);
  const scopes = useExplorer((s) => s.scopes);
  const view = useExplorer(activeView);

  if (status.busy !== null && scopes.state !== 'loading') return null;
  // Something is drawn, so the canvas speaks for itself.
  if (status.nodes > 0 && status.visibleNodes > 0) return null;

  const message = notice ?? emptyReason(scopes, status.nodes, view);
  if (!message) return null;

  return (
    <div className={`canvas-overlay ${message.level === 'error' ? 'is-error' : ''}`}>
      <p className="overlay-text">{message.text}</p>
      {message.hint ? <p className="overlay-hint">{message.hint}</p> : null}
    </div>
  );
}

/** The share of the canvas width the axis compresses down to. A whole one leaves no margin. */
const WIDTH_FIT = 0.6;

/**
 * compressionFloor is the least stretch the wheel allows — the one drawing the axis across
 * WIDTH_FIT of the canvas — from what one unit of stretch is worth on screen, which the camera
 * and the window size both feed into.
 */
function compressionFloor(stretch: number): number {
  const width = axisWidth();
  const canvas = canvasWidth();
  if (width === null || width <= 0 || canvas <= 0 || stretch <= 0) return 0;
  const drawn = axisSpanOnScreen(width * stretch);
  if (drawn === null || drawn <= 0) return 0;
  const perUnit = drawn / stretch;
  return perUnit > 0 ? (WIDTH_FIT * canvas) / perUnit : 0;
}

/**
 * emptyReason names why the canvas is blank when no action reported one. A listed archive is the
 * common case, since nothing auto-selects where there is a choice — so say what is there.
 */
function emptyReason(scopes: ScopesState, nodes: number, view: ViewState | null): Notice | null {
  if (nodes === 0) {
    if (scopes.state === 'loading') {
      return { level: 'info', text: 'Listing branches…', hint: 'Asking the archive what it holds.' };
    }
    if (scopes.state === 'ready') {
      const branches = scopes.scopes.filter((s) => s.name !== ARCHIVE_SCOPE).length;
      return {
        level: 'info',
        text: `Archive holds ${branches.toLocaleString('en-US')} branch${branches === 1 ? '' : 'es'}.`,
        hint: 'Select one in the header to load it, or run a query.',
      };
    }
    return {
      level: 'info',
      text: 'Nothing loaded yet.',
      hint: 'List the archive’s branches from the header, or run a query.',
    };
  }
  // Claims are held but none survive this view's predicates, which is a filter, not a fault.
  const filters = [
    view?.scope ? `scope ${view.scope.name}` : null,
    view?.classes.length ? `classes ${view.classes.join('+')}` : null,
    view?.contributionRange ? 'a contribution range' : null,
  ].filter(Boolean);
  return {
    level: 'info',
    text: `${nodes.toLocaleString('en-US')} claims loaded, none shown by this view.`,
    hint: filters.length > 0 ? `Narrowed by ${filters.join(' and ')}.` : undefined,
  };
}

export function App() {
  const host = useRef<HTMLDivElement | null>(null);
  const lensHost = useRef<HTMLDivElement | null>(null);
  const views = useExplorer((s) => s.views);
  const activeViewId = useExplorer((s) => s.activeViewId);
  const view = useExplorer(activeView);
  const sidePane = useExplorer((s) => s.sidePane);
  const setSidePane = useExplorer((s) => s.setSidePane);
  const activateView = useExplorer((s) => s.activateView);
  const closeView = useExplorer((s) => s.closeView);
  const activeConnectionId = useConnections((s) => s.activeId);

  // Mount the renderer once. Not a dependency of any state: a re-render must not
  // recreate it, and StrictMode's double-invoke is harmless because `mount` is
  // idempotent for a given container.
  useEffect(() => {
    if (host.current) mount(host.current);
    if (lensHost.current) mountLens(lensHost.current);
    // Shift stretches time; the plain wheel is the camera's. The lens below is only what keeps a
    // large graph affordable.
    const stopZoom = onZoom(({ factor, viewportX, shift }) => {
      const state = useExplorer.getState();
      const view = state.views.find((v) => v.id === state.activeViewId);
      // Only the timeline has an axis worth stretching, and only shift asks for it. Everything
      // else is the camera's classic zoom — which, alternated with a compression of time, is a
      // vertical stretch in all but name.
      if (view?.layout !== 'timeline' || !shift) return false;

      // Compression stops with the axis across WIDTH_FIT of the viewport: past that the picture
      // is stranded in empty space and the camera's zoom is the right instrument.
      const floor = compressionFloor(view.xStretch);

      // A stretch multiplies graph x, so the content's new position is arithmetic.
      const under = graphXAt(viewportX);
      const { applied } = stretchX(factor, shownGraph() ?? undefined, floor);
      repaint();
      if (under !== null && applied !== 1) anchorAt(viewportX, under * applied);
      return true;
    });

    // Which graph to show follows the camera, so it is decided after a frame rather than by
    // the UI watching for gestures.
    const stop = onRender(() => {
      const width = canvasWidth();
      if (width <= 0) return;
      const left = graphXAt(0);
      const right = graphXAt(width);
      if (left === null || right === null) return;
      const cut = lensFor(left, right);
      if (cut) showLens(cut.graph);
      else hideLens();
    });
    // Core stages a load and hands the finished graph back through this hook, so it
    // never has to import the renderer.
    // Reframing is the camera's, so core asks for it through a hook rather than importing it.
    setOnShowAll(() => resetCamera());
    setOnLoaded(() => {
      const state = useExplorer.getState();
      const view = state.views.find((v) => v.id === state.activeViewId);
      const extent = timelineExtent();
      // A stretch is only visible if the renderer stops fitting the graph it is stretching.
      if (view?.layout === 'timeline' && extent) {
        pinExtent(extent.x0, extent.x1, extent.y0, extent.y1);
      } else {
        unpinExtent();
      }
      refreshSelection();
    });
    // What an archive holds is the first thing worth knowing, and asking for it is not a
    // decision the reader should have to make. A sole branch then loads itself.
    void discoverScopes();
    return () => {
      stopZoom();
      stop();
    };
  }, []);

  // A change of source is a different archive, so its branches are asked for afresh.
  useEffect(() => {
    if (activeConnectionId) void discoverScopes();
  }, [activeConnectionId]);

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
        {/* The overlays are positioned against the canvas, not the pane, so none of them
            reaches up over the tab handles. */}
        <div className="canvas-area">
          <div className="canvas-host" ref={host} />
          {/* The lens's own canvas, hidden until a zoom asks for it. Both are resident, so
              swapping them uploads nothing. */}
          <div className="canvas-host canvas-lens" ref={lensHost} />
          <ZoomBox />
          <TimeCursor />
          <TimeRuler />
          <HoverPreviewChip />
          <LoadingOverlay />
          <EmptyCanvas />
        </div>
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
