/**
 * package: ui / shell
 * type:    view
 * job:     the shell — header, footer, and the tabbed main and side panes
 * limits:  layout only; each pane owns its content (-> ui/panes)
 *
 * The canvas host is mounted once and never unmounted by a re-render: Sigma and the graph live
 * outside React, and this holds only a ref and ids.
 */

import { useEffect, useRef, useState } from 'react';
import { activeView, useExplorer } from '../../core/store.ts';
import type { Notice, ScopesState, SidePane, ViewState } from '../../core/store.ts';
import { ARCHIVE_SCOPE } from '../../core/scope.ts';
import {
  discoverScopes,
  lensFor,
  setOnLoaded,
  setOnShowAll,
} from '../../core/session.ts';
import { settleUnion, timelineExtent } from '../../core/timeline.ts';
import { useConnections } from '../../core/connections.ts';
import {
  applyViewSettings,
  hideLens,
  mount,
  mountLens,
  onRender,
  onZoom,
  pinExtent,
  refreshSelection,
  showLens,
  unpinExtent,
} from '../../render/renderer.ts';
import {
  fitHeight,
  graphXAt,
  holdCamera,
  resetCamera,
  zoomX,
  zoomY,
} from '../../render/camera.ts';
import { canvasWidth } from '../../render/instances.ts';
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

/** How wide the side pane opens, and the range a drag may take it to, in pixels. */
const SIDE_DEFAULT = 480;
const SIDE_MIN = 280;
/** The graph keeps at least this much of the window, whatever the pane is dragged to. */
const SIDE_MAX_SHARE = 0.6;

/**
 * SideGrip resizes the side pane by dragging its edge. The width goes onto the grid as a custom
 * property rather than through the store: it is a fact about this window, not about the archive.
 */
function SideGrip({ onWidth }: { onWidth: (px: number) => void }) {
  const [dragging, setDragging] = useState(false);

  const start = (event: React.PointerEvent<HTMLDivElement>) => {
    event.currentTarget.setPointerCapture(event.pointerId);
    setDragging(true);
  };
  const move = (event: React.PointerEvent<HTMLDivElement>) => {
    if (!dragging) return;
    const width = window.innerWidth - event.clientX;
    onWidth(Math.max(SIDE_MIN, Math.min(width, window.innerWidth * SIDE_MAX_SHARE)));
  };
  const end = (event: React.PointerEvent<HTMLDivElement>) => {
    event.currentTarget.releasePointerCapture(event.pointerId);
    setDragging(false);
  };

  return (
    <div
      className={`side-grip${dragging ? ' is-dragging' : ''}`}
      role="separator"
      aria-orientation="vertical"
      title="drag to resize"
      onPointerDown={start}
      onPointerMove={move}
      onPointerUp={end}
      onPointerCancel={end}
    />
  );
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
  const [sideWidth, setSideWidth] = useState(SIDE_DEFAULT);

  // Mount the renderer once. Not a dependency of any state: a re-render must not
  // recreate it, and StrictMode's double-invoke is harmless because `mount` is
  // idempotent for a given container.
  useEffect(() => {
    if (host.current) mount(host.current);
    if (lensHost.current) mountLens(lensHost.current);
    // The wheel zooms time, shift zooms the strata. The lens below is only what keeps a large
    // graph affordable.
    const stopZoom = onZoom(({ factor, viewportX, viewportY, shift }) => {
      const state = useExplorer.getState();
      const view = state.views.find((v) => v.id === state.activeViewId);
      // Only the timeline has an axis apiece to zoom. Everything else is the camera's classic
      // zoom, which takes both at once.
      if (view?.layout !== 'timeline') return false;
      if (shift) zoomY(factor, viewportY);
      else zoomX(factor, viewportX);
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
      // A stretch over a lens reaches the copy alone, so the union is put back on the axis before
      // it can be shown again or cut from — a lens cut from a stale union is stale too.
      if (settleUnion()) refreshSelection();
      const cut = lensFor(left, right);
      if (cut) showLens(cut.graph);
      else hideLens();
    });
    // Core stages a load and hands the finished graph back through this hook, so it
    // never has to import the renderer.
    // Reframing is the camera's, so core asks for it through a hook rather than importing it.
    // All of time across the canvas, and the strata refitted to it once the camera has settled —
    // how tall they should be drawn is a fact about the window, which only this layer can measure.
    setOnShowAll(() => resetCamera(fitHeight));
    setOnLoaded((framing) => {
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
      // The graph a reader has just asked for is framed to the strata, which is the picture the
      // fit-height tool gives — asking for a branch and being shown a band of dots in the middle
      // of an empty canvas is a reader doing the renderer's work.
      if (framing === 'fit') fitHeight();
      // A load is the other way the picture moves without a gesture, and it was the one that
      // used to land outside what the wheel could reach.
      else holdCamera();
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
    <div className="app" style={{ '--side-width': `${sideWidth}px` } as React.CSSProperties}>
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
        <SideGrip onWidth={setSideWidth} />
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
