/**
 * @layer core
 * @description All user-initiated operations against the application state.
 * @depends core/store, core/mock, core/types/graph
 * @must-not Import from ui/ or graph/. Reference React or DOM.
 */

import { store } from './store';
import { generateMockGraph } from './mock';
import type { ViewMode, GraphMode } from './types/graph';
import type { Node } from './types/nodes';

// --- Data loading ---

export function loadMockData(count: number): void {
  const { nodes, edges } = generateMockGraph(count);

  const nodeMap = new Map(nodes.map((n) => [n.id, n]));
  const edgeMap = new Map(edges.map((e) => [e.id, e]));

  // Compute timeline domain
  let minDate: Date | null = null;
  let maxDate: Date | null = null;
  for (const node of nodes) {
    const d = node.artifactCreatedAt ?? node.createdAt;
    if (!minDate || d < minDate) minDate = d;
    if (!maxDate || d > maxDate) maxDate = d;
  }

  // Collect all content classes for filter initialization
  const allClasses = new Set<string>();
  for (const node of nodes) {
    allClasses.add(node.contentClass);
  }

  store.setState({
    nodes: nodeMap,
    edges: edgeMap,
    nodeCount: nodes.length,
    edgeCount: edges.length,
    useMockData: true,
    timelineDomain: minDate && maxDate ? [minDate, maxDate] : null,
    timelineViewport: minDate && maxDate ? [minDate, maxDate] : null,
    filters: {
      levels: new Set([0, 1, 2]),
      contentClasses: allClasses,
      dateRange: { from: null, to: null },
      minConfidence: -1,
    },
  });
}

export async function loadFromApi(): Promise<void> {
  // Placeholder for Phase 2 — will use core/api.ts wrapper
  console.warn('loadFromApi not yet implemented');
}

// --- Selection (multi-select) ---

export function selectNode(id: string, addToSelection = false): void {
  store.setState((s) => {
    const next = addToSelection ? new Set(s.selectedNodeIds) : new Set<string>();
    next.add(id);
    return { selectedNodeIds: next, selectedEdgeIds: new Set() };
  });
}

export function deselectNode(id: string): void {
  store.setState((s) => {
    const next = new Set(s.selectedNodeIds);
    next.delete(id);
    return { selectedNodeIds: next };
  });
}

export function selectEdge(id: string, addToSelection = false): void {
  store.setState((s) => {
    const next = addToSelection ? new Set(s.selectedEdgeIds) : new Set<string>();
    next.add(id);
    return { selectedEdgeIds: next, selectedNodeIds: new Set() };
  });
}

export function clearSelection(): void {
  store.setState({ selectedNodeIds: new Set(), selectedEdgeIds: new Set() });
}

// --- View ---

export function setViewMode(mode: ViewMode): void {
  store.setState({ viewMode: mode });
}

export function setGraphMode(mode: GraphMode): void {
  store.setState({ graphMode: mode });
}

// --- Filters ---

export function setLevelFilter(levels: Set<number>): void {
  store.setState((s) => ({
    filters: { ...s.filters, levels },
  }));
}

export function setContentClassFilter(classes: Set<string>): void {
  store.setState((s) => ({
    filters: { ...s.filters, contentClasses: classes },
  }));
}

export function setDateRange(from: Date | null, to: Date | null): void {
  store.setState((s) => ({
    filters: { ...s.filters, dateRange: { from, to } },
  }));
}

export function setMinConfidence(value: number): void {
  store.setState((s) => ({
    filters: { ...s.filters, minConfidence: value },
  }));
}

// --- Timeline ---

export function setTimelineViewport(from: Date, to: Date): void {
  store.setState({ timelineViewport: [from, to] });
}

// --- Helpers ---

/** Get the temporal position for a node (for timeline / sorting). */
export function temporalPosition(node: Node): Date {
  return node.artifactCreatedAt ?? node.validFrom ?? node.createdAt;
}
