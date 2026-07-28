/**
 * The headless core's state. No React, no DOM, no Sigma.
 *
 * Two rules from the explorer design govern this file:
 *
 * 1. **One store holds the union; views are selections over it.** The graphology
 *    instance is the union of every claim loaded this session. A view does not own
 *    a graph — it owns a predicate. Switching views is a reducer swap, never a
 *    mutation of the graph.
 * 2. **Graph data never enters framework state.** The graph itself lives in
 *    `core/graph/universe.ts` at module scope; this store holds only ids, labels
 *    and counts, which is what the UI is allowed to see.
 */

import { create } from 'zustand';
import type { LayoutName } from './layout/layouts.ts';

/** A main-pane tab: a named selection over the union graph. */
export interface ViewState {
  id: string;
  label: string;
  /** Contribution range this view admits, or null for everything loaded. */
  contributionRange: [number, number] | null;
  /** Claim classes this view admits, empty meaning all. */
  classes: string[];
  layout: LayoutName;
  /** Draw edges at all. */
  edges: boolean;
  /** Draw edges while the camera is moving. */
  edgesOnMove: boolean;
  labels: boolean;
  /** Draw labels while the camera is moving. */
  labelsOnMove: boolean;
  sizeByDegree: boolean;
}

/** Side-pane tabs are fixed: tooling plus detail views. */
export type SidePane = 'query' | 'server' | 'view' | 'selection' | 'graph' | 'log';

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

export interface SelectionState {
  /** Claim id the user clicked; drives the detail pane. */
  selected: string | null;
  /** Claim id under the pointer; drives only the cheap preview. */
  hovered: string | null;
}

/** The debounced hover preview: label and type, and nothing heavier. */
export interface HoverPreview {
  id: string;
  label: string;
  claimType: string;
}

export interface ExplorerState {
  views: ViewState[];
  activeViewId: string | null;
  sidePane: SidePane;
  status: StatusState;
  selection: SelectionState;
  /** Debounced, so sweeping the pointer cannot thrash the UI. */
  preview: HoverPreview | null;
  log: string[];

  addView: (view: ViewState) => void;
  closeView: (id: string) => void;
  activateView: (id: string) => void;
  patchView: (id: string, patch: Partial<ViewState>) => void;
  setSidePane: (pane: SidePane) => void;
  patchStatus: (patch: Partial<StatusState>) => void;
  select: (id: string | null) => void;
  hover: (id: string | null) => void;
  setPreview: (preview: HoverPreview | null) => void;
  appendLog: (line: string) => void;
}

/** DEFAULT_VIEW is the shape a fresh graph tab starts in. */
export function defaultView(id: string, label: string): ViewState {
  return {
    id,
    label,
    contributionRange: null,
    classes: [],
    layout: 'history',
    edges: true,
    edgesOnMove: false,
    labels: true,
    labelsOnMove: true,
    sizeByDegree: true,
  };
}

const MAX_LOG = 200;

export const useExplorer = create<ExplorerState>((set) => ({
  views: [],
  activeViewId: null,
  sidePane: 'query',
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
  selection: { selected: null, hovered: null },
  preview: null,
  log: [],

  addView: (view) => set((s) => ({ views: [...s.views, view], activeViewId: view.id })),

  closeView: (id) =>
    set((s) => {
      const views = s.views.filter((v) => v.id !== id);
      const activeViewId =
        s.activeViewId === id ? (views[views.length - 1]?.id ?? null) : s.activeViewId;
      return { views, activeViewId };
    }),

  activateView: (id) => set({ activeViewId: id }),

  patchView: (id, patch) =>
    set((s) => ({ views: s.views.map((v) => (v.id === id ? { ...v, ...patch } : v)) })),

  setSidePane: (sidePane) => set({ sidePane }),

  patchStatus: (patch) => set((s) => ({ status: { ...s.status, ...patch } })),

  select: (selected) => set((s) => ({ selection: { ...s.selection, selected } })),

  hover: (hovered) => set((s) => ({ selection: { ...s.selection, hovered } })),

  setPreview: (preview) => set({ preview }),

  appendLog: (line) => set((s) => ({ log: [...s.log, line].slice(-MAX_LOG) })),
}));

/** activeView is the selector the UI uses; nothing else derives it. */
export function activeView(s: ExplorerState): ViewState | null {
  return s.views.find((v) => v.id === s.activeViewId) ?? null;
}
