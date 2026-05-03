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

  // Per-element visibility (user intent, survives selection changes)
  hiddenNodeIds: Set<string>;
  hiddenEdgeIds: Set<string>;

  // Emphasis modifiers
  emphasisMode: 'neighbors' | 'provenance';
  deemphasisTreatment: 'dim' | 'hide';

  // Tool mode — pan / select (with marquee) / drag (nodes movable)
  toolMode: 'pan' | 'select' | 'drag';

  // View
  viewMode: ViewMode;
  graphMode: GraphMode;

  // Filters
  filters: FilterState;

  // Timeline
  timelineDomain: [Date, Date] | null;
  timelineViewport: [Date, Date] | null;

  // Active run preview — raw membership, consumed by emphasis selector
  runNodeIds: Set<string> | null;
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

  hiddenNodeIds: new Set<string>(),
  hiddenEdgeIds: new Set<string>(),

  emphasisMode: 'neighbors',
  deemphasisTreatment: 'dim',

  toolMode: 'pan',

  viewMode: 'graph' as ViewMode,
  graphMode: 'provenance' as GraphMode,

  filters: {
    levels: new Set([0, 1, 2]),
    contentClasses: new Set<string>(),
    contentTypes: new Set<string>(),
    encodingClasses: new Set<string>(),
    encodingFormats: new Set<string>(),
    dateRange: { from: null, to: null },
    minConfidence: -1,
  },

  timelineDomain: null,
  timelineViewport: null,

  runNodeIds: null,
  activeRunId: null,

  nodeCount: 0,
  edgeCount: 0,
}));
