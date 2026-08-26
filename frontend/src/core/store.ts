/**
 * package: core / store
 * type:    data
 * job:     hold the headless core's state
 * limits:  no React, no DOM, no Sigma; graph data never enters React state (-> ui)
 *
 * One store holds the union; a view is a predicate over it, not a graph, so switching
 * views swaps reducers. The graph lives at module scope in core/graph/universe.ts; this
 * holds ids, labels and counts.
 */

import { create } from 'zustand';
import type { LayoutName } from './layout/layouts.ts';
import type { Scope } from './scope.ts';

/** A main-pane tab: a named selection over the union graph. */
export interface ViewState {
  kind: 'graph';
  id: string;
  label: string;
  /** Contribution range this view admits, or null for everything loaded. */
  contributionRange: [number, number] | null;
  /**
   * Scope this view is confined to, or null for everything loaded. Only the name and head
   * live here; the ids the scope contains are the server's answer and stay out of React
   * state (-> core/graph/members).
   */
  scope: Scope | null;
  /** Claim classes this view admits, empty meaning all. */
  classes: string[];
  layout: LayoutName;
  /**
   * How far each axis is stretched, 1 being the layout's own scale. An axis apiece is what a
   * reader zooms: the camera would take both at once and magnify the claims along with them,
   * where a stretch spreads them out at the size they are.
   */
  xStretch: number;
  yStretch: number;
  /** Draw edges at all. */
  edges: boolean;
  /** Draw edges while the camera is moving. */
  edgesOnMove: boolean;
  labels: boolean;
  /** Draw labels while the camera is moving. */
  labelsOnMove: boolean;
  sizeByDegree: boolean;
}

/**
 * A main-pane tab showing one claim's raw CBOR. It carries only what a CBOR read needs — a
 * claim has no layout, no strata, none of a graph view's settings — so it is a different
 * shape entirely rather than a `ViewState` with most of its fields unused. The bytes
 * themselves stay out of the tab, and out of the store: they are fetched into a module-scope
 * cache (-> core/claimBytes), which is where content already lives for the same reason — a
 * claim is content-addressed, so nothing here can go stale.
 */
export interface CborTabState {
  kind: 'cbor';
  id: string;
  label: string;
  claimId: string;
  /** Scope the claim was reached through — content-addressed, so any scope holding it will do. */
  scope: Scope | null;
}

/** A main-pane tab strip holds one kind-tagged list rather than two kept in step. */
export type MainTab = ViewState | CborTabState;

/**
 * What the explorer knows about the archive's scopes. `unknown` before anything asked,
 * since a branch name is a fact about the archive in front of us and never a default.
 */
export interface ScopesState {
  state: 'unknown' | 'loading' | 'ready' | 'error';
  scopes: Scope[];
  /**
   * The scope currently chosen, which a new view inherits — the same way a new view
   * inherits the query's classes. A selection can be made before any view exists, so it
   * cannot live only on one.
   */
  selected: Scope | null;
  /** Why the listing failed, when it did. */
  error: string | null;
}

/** Side-pane tabs are fixed: tooling plus detail views. */
export type SidePane = 'query' | 'server' | 'view' | 'info' | 'log';

/** What the footer reports. Written by the renderer, read by the UI. */
export interface StatusState {
  nodes: number;
  edges: number;
  contributions: number;
  /** Drawn after the last full refresh — what the reducers admitted. */
  visibleNodes: number;
  visibleEdges: number;
  fps: number | null;
  frameMs: number | null;
  /** Cost of the last full refresh: re-index plus buffer upload, O(N). */
  lastRefreshMs: number | null;
  /** Longest gap between animation frames in the last window — a stalled thread. */
  stallMs: number | null;
  renderer: string;
  /** Stage label while a load is in flight, null when idle. */
  busy: string | null;
  /** Fraction of the current stage (0..1), when it is measurable. */
  progress: number | null;
}

/**
 * What the last action has to say for itself, shown on the canvas when there is nothing to
 * draw. A blank canvas is indistinguishable from a broken one, so an empty result, a
 * refusal and a scope nothing is cached for each say which they are.
 */
export interface Notice {
  level: 'info' | 'error';
  text: string;
  /** What to do about it, when there is something to do. */
  hint?: string;
}

/**
 * One claim's content, fetched when it is selected. Held here rather than in the graph: a
 * real archive's bytes are not something to carry a hundred thousand of, and a claim's
 * content is wanted only while it is the one being looked at.
 */
export interface ContentState {
  id: string;
  state: 'loading' | 'ready' | 'too-large' | 'none' | 'error';
  bytes?: Uint8Array;
  /** Size as the claim declares it, which is what decides whether to ask for the bytes. */
  size: number;
  encoding?: string;
  error?: string;
}

export interface SelectionState {
  /** Claim id the user clicked; drives the detail pane. */
  selected: string | null;
  /** Edge key the user clicked. A node and an edge are alternatives, never both. */
  selectedEdge: string | null;
  /** Claim id under the pointer; drives only the cheap preview. */
  hovered: string | null;
}

/**
 * A Visit is one thing the reader has looked at — a claim, an edge, or the graph itself, which is
 * what an empty visit is. The pane answers about one at a time, so a trail through them is a
 * trail through what the reader asked.
 */
export interface Visit {
  node: string | null;
  edge: string | null;
}

/** Where the reader has been, and where along it they currently stand. */
export interface HistoryState {
  visits: Visit[];
  at: number;
}

/** The debounced hover preview: type and a content prefix, and nothing heavier. */
export interface HoverPreview {
  id: string;
  claimType: string;
  /** A prefix of the claim's content, where it is text and already read — else empty. */
  content: string;
  /** Pointer position within the canvas host, in pixels — where the tooltip floats. */
  x: number;
  y: number;
}

export interface ExplorerState {
  tabs: MainTab[];
  activeTabId: string | null;
  sidePane: SidePane;
  scopes: ScopesState;
  status: StatusState;
  selection: SelectionState;
  history: HistoryState;
  /** Debounced, so sweeping the pointer cannot thrash the UI. */
  preview: HoverPreview | null;
  /** What the last action has to say — read by the canvas when it has nothing to draw. */
  notice: Notice | null;
  /** The selected claim's content, or null when none is selected. */
  content: ContentState | null;
  log: string[];

  addTab: (tab: MainTab) => void;
  closeTab: (id: string) => void;
  activateTab: (id: string) => void;
  patchView: (id: string, patch: Partial<ViewState>) => void;
  setSidePane: (pane: SidePane) => void;
  setScopes: (scopes: ScopesState) => void;
  patchStatus: (patch: Partial<StatusState>) => void;
  select: (id: string | null) => void;
  selectEdge: (key: string | null) => void;
  /** Walks the trail without adding to it: -1 is back, +1 is forward. */
  stepHistory: (delta: number) => void;
  hover: (id: string | null) => void;
  setPreview: (preview: HoverPreview | null) => void;
  setNotice: (notice: Notice | null) => void;
  setContent: (content: ContentState | null) => void;
  appendLog: (line: string) => void;
}

/** DEFAULT_VIEW is the shape a fresh graph tab starts in. */
export function defaultView(id: string, label: string): ViewState {
  return {
    kind: 'graph',
    id,
    label,
    contributionRange: null,
    scope: null,
    classes: [],
    // Time first: an archive is historical, so that is the axis a reader wants.
    layout: 'timeline',
    xStretch: 1,
    yStretch: 1,
    edges: true,
    // Edges while the camera moves: what is being looked at is the shape, and hiding them
    // mid-gesture hides the thing.
    edgesOnMove: true,
    labels: true,
    labelsOnMove: true,
    sizeByDegree: true,
  };
}

const MAX_LOG = 200;

/**
 * visited adds where the reader has just gone. Forward is where they were before they turned
 * back, so stepping anywhere new replaces it — a trail forks at the point it is walked from.
 */
function visited(history: HistoryState, visit: Visit): HistoryState {
  const here = history.visits[history.at];
  if (here && here.node === visit.node && here.edge === visit.edge) return history;
  const visits = [...history.visits.slice(0, history.at + 1), visit];
  return { visits, at: visits.length - 1 };
}

export const useExplorer = create<ExplorerState>((set) => ({
  tabs: [],
  activeTabId: null,
  sidePane: 'info',
  scopes: { state: 'unknown', scopes: [], selected: null, error: null },
  status: {
    nodes: 0,
    edges: 0,
    contributions: 0,
    visibleNodes: 0,
    visibleEdges: 0,
    fps: null,
    frameMs: null,
    lastRefreshMs: null,
    stallMs: null,
    renderer: 'unknown',
    busy: null,
    progress: null,
  },
  selection: { selected: null, selectedEdge: null, hovered: null },
  history: { visits: [], at: -1 },
  preview: null,
  notice: null,
  content: null,
  log: [],

  addTab: (tab) => set((s) => ({ tabs: [...s.tabs, tab], activeTabId: tab.id })),

  closeTab: (id) =>
    set((s) => {
      const tabs = s.tabs.filter((t) => t.id !== id);
      const activeTabId = s.activeTabId === id ? (tabs[tabs.length - 1]?.id ?? null) : s.activeTabId;
      return { tabs, activeTabId };
    }),

  activateTab: (id) => set({ activeTabId: id }),

  patchView: (id, patch) =>
    set((s) => ({
      tabs: s.tabs.map((t) => (t.id === id && t.kind === 'graph' ? { ...t, ...patch } : t)),
    })),

  setSidePane: (sidePane) => set({ sidePane }),

  setScopes: (scopes) => set({ scopes }),

  patchStatus: (patch) => set((s) => ({ status: { ...s.status, ...patch } })),

  // Selecting one clears the other: the detail pane answers about one thing at a time.
  select: (selected) =>
    set((s) => ({
      selection: { ...s.selection, selected, selectedEdge: null },
      history: visited(s.history, { node: selected, edge: null }),
    })),

  selectEdge: (selectedEdge) =>
    set((s) => ({
      selection: { ...s.selection, selectedEdge, selected: null },
      history: visited(s.history, { node: null, edge: selectedEdge }),
    })),

  stepHistory: (delta) =>
    set((s) => {
      const at = s.history.at + delta;
      const visit = s.history.visits[at];
      if (!visit) return {};
      return {
        history: { ...s.history, at },
        selection: { ...s.selection, selected: visit.node, selectedEdge: visit.edge },
      };
    }),

  hover: (hovered) => set((s) => ({ selection: { ...s.selection, hovered } })),

  setPreview: (preview) => set({ preview }),

  setNotice: (notice) => set({ notice }),

  setContent: (content) => set({ content }),

  appendLog: (line) => set((s) => ({ log: [...s.log, line].slice(-MAX_LOG) })),
}));

/**
 * activeView is the selector the UI uses; nothing else derives it. Null both when nothing is
 * active and when the active tab is a CBOR tab — a graph view's settings, camera bound and
 * reducers all read through this, and none of them apply to a claim's bytes.
 */
export function activeView(s: ExplorerState): ViewState | null {
  const tab = s.tabs.find((t) => t.id === s.activeTabId);
  return tab?.kind === 'graph' ? tab : null;
}

