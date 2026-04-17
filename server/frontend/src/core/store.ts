/**
 * @layer core
 * @description Zustand store holding all application state.
 * @depends core/types/nodes, core/types/filters, core/types/graph
 * @must-not Import from ui/ or graph/. Reference React or DOM.
 */

import { createStore } from 'zustand/vanilla';
import type { Node, Edge } from './types/nodes';
import type { FilterState } from './types/filters';
import type { ViewMode, GraphMode } from './types/graph';

export interface AppState {
  // Connection
  isConnected: boolean;
  useMockData: boolean;

  // Data
  nodes: Map<string, Node>;
  edges: Map<string, Edge>;

  // Selection (multi-select)
  selectedNodeIds: Set<string>;
  selectedEdgeIds: Set<string>;

  // View
  viewMode: ViewMode;
  graphMode: GraphMode;

  // Filters
  filters: FilterState;

  // Timeline
  timelineDomain: [Date, Date] | null;
  timelineViewport: [Date, Date] | null;

  // Highlight (e.g. run preview — dims everything outside this set)
  highlightedNodeIds: Set<string> | null;
  activeRunId: string | null;

  // Stats
  nodeCount: number;
  edgeCount: number;
}

export const store = createStore<AppState>()(() => ({
  isConnected: false,
  useMockData: true,

  nodes: new Map(),
  edges: new Map(),

  selectedNodeIds: new Set<string>(),
  selectedEdgeIds: new Set<string>(),

  viewMode: 'graph' as ViewMode,
  graphMode: 'provenance' as GraphMode,

  filters: {
    levels: new Set([0, 1, 2]),
    contentClasses: new Set<string>(),
    dateRange: { from: null, to: null },
    minConfidence: -1,
  },

  timelineDomain: null,
  timelineViewport: null,

  highlightedNodeIds: null,
  activeRunId: null,

  nodeCount: 0,
  edgeCount: 0,
}));
