/**
 * @layer core
 * @description All user-initiated operations against the application state.
 * @depends core/store, core/mock, core/types/graph
 * @must-not Import from ui/ or graph/. Reference React or DOM.
 */

import { store } from './store';
import { generateMockGraph } from './mock';
import { fetchNodes, fetchEdges, fetchHealth } from './api';
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

  // Enumerate filter variants for show/hide controls
  const allClasses = new Set<string>();
  const allContentTypes = new Set<string>();
  const allEncodingClasses = new Set<string>();
  const allEncodingFormats = new Set<string>();
  for (const node of nodes) {
    allClasses.add(node.contentClass);
    allContentTypes.add(`${node.contentClass}/${node.contentType}`);
    allEncodingClasses.add(node.encodingClass);
    allEncodingFormats.add(`${node.encodingClass}/${node.encodingFormat}`);
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
      contentTypes: allContentTypes,
      encodingClasses: allEncodingClasses,
      encodingFormats: allEncodingFormats,
      dateRange: { from: null, to: null },
      minConfidence: -1,
    },
  });
}

export async function loadFromApi(): Promise<void> {
  try {
    const health = await fetchHealth();
    store.setState({ isConnected: health.status === 'ok' });

    const [nodes, edges] = await Promise.all([
      fetchNodes({ limit: 10000 }),
      fetchEdges({ limit: 50000 }),
    ]);
    const nodeMap = new Map(nodes.map((n) => [n.id, n]));
    const edgeMap = new Map(edges.map((e) => [e.id, e]));

    let minDate: Date | null = null;
    let maxDate: Date | null = null;
    const allClasses = new Set<string>();
    const allContentTypes = new Set<string>();
    const allEncodingClasses = new Set<string>();
    const allEncodingFormats = new Set<string>();
    for (const node of nodes) {
      allClasses.add(node.contentClass);
      allContentTypes.add(`${node.contentClass}/${node.contentType}`);
      allEncodingClasses.add(node.encodingClass);
      allEncodingFormats.add(`${node.encodingClass}/${node.encodingFormat}`);
      const d = node.artifactCreatedAt ?? node.createdAt;
      if (!minDate || d < minDate) minDate = d;
      if (!maxDate || d > maxDate) maxDate = d;
    }

    store.setState({
      nodes: nodeMap,
      edges: edgeMap,
      nodeCount: nodes.length,
      edgeCount: edgeMap.size,
      useMockData: false,
      timelineDomain: minDate && maxDate ? [minDate, maxDate] : null,
      timelineViewport: minDate && maxDate ? [minDate, maxDate] : null,
      filters: {
        levels: new Set([0, 1, 2]),
        contentClasses: allClasses,
        contentTypes: allContentTypes,
        encodingClasses: allEncodingClasses,
        encodingFormats: allEncodingFormats,
        dateRange: { from: null, to: null },
        minConfidence: -1,
      },
    });
  } catch (err) {
    console.error('Failed to load from API:', err);
    store.setState({ isConnected: false });
  }
}

// --- Selection (multi-select) ---

export function selectNode(id: string, addToSelection = false): void {
  store.setState((s) => {
    const next = addToSelection ? new Set(s.selectedNodeIds) : new Set<string>();
    next.add(id);
    const hidden = s.hiddenNodeIds.has(id) ? new Set(s.hiddenNodeIds) : s.hiddenNodeIds;
    if (hidden !== s.hiddenNodeIds) hidden.delete(id);
    return { selectedNodeIds: next, selectedEdgeIds: new Set(), hiddenNodeIds: hidden };
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
    const hidden = s.hiddenEdgeIds.has(id) ? new Set(s.hiddenEdgeIds) : s.hiddenEdgeIds;
    if (hidden !== s.hiddenEdgeIds) hidden.delete(id);
    return { selectedEdgeIds: next, selectedNodeIds: new Set(), hiddenEdgeIds: hidden };
  });
}

export function clearSelection(): void {
  store.setState({ selectedNodeIds: new Set(), selectedEdgeIds: new Set() });
}

// --- Visibility ---

export function hideNode(id: string): void {
  store.setState((s) => {
    const next = new Set(s.hiddenNodeIds);
    next.add(id);
    return { hiddenNodeIds: next };
  });
}

export function showNode(id: string): void {
  store.setState((s) => {
    if (!s.hiddenNodeIds.has(id)) return {};
    const next = new Set(s.hiddenNodeIds);
    next.delete(id);
    return { hiddenNodeIds: next };
  });
}

export function hideEdge(id: string): void {
  store.setState((s) => {
    const next = new Set(s.hiddenEdgeIds);
    next.add(id);
    return { hiddenEdgeIds: next };
  });
}

export function showEdge(id: string): void {
  store.setState((s) => {
    if (!s.hiddenEdgeIds.has(id)) return {};
    const next = new Set(s.hiddenEdgeIds);
    next.delete(id);
    return { hiddenEdgeIds: next };
  });
}

export function hideSelected(): void {
  store.setState((s) => ({
    hiddenNodeIds: new Set([...s.hiddenNodeIds, ...s.selectedNodeIds]),
    hiddenEdgeIds: new Set([...s.hiddenEdgeIds, ...s.selectedEdgeIds]),
    selectedNodeIds: new Set(),
    selectedEdgeIds: new Set(),
  }));
}

export function showAll(): void {
  store.setState({ hiddenNodeIds: new Set(), hiddenEdgeIds: new Set() });
}

// --- Emphasis modifiers ---

export function setEmphasisMode(mode: 'neighbors' | 'provenance'): void {
  store.setState({ emphasisMode: mode });
}

export function setDeemphasisTreatment(treatment: 'dim' | 'hide'): void {
  store.setState({ deemphasisTreatment: treatment });
}

// --- Tool mode ---

export function setToolMode(mode: 'pan' | 'select' | 'drag'): void {
  store.setState({ toolMode: mode });
}

export function setSelection(nodeIds: Iterable<string>, edgeIds: Iterable<string>): void {
  store.setState({
    selectedNodeIds: new Set(nodeIds),
    selectedEdgeIds: new Set(edgeIds),
  });
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

export function setContentTypeFilter(types: Set<string>): void {
  store.setState((s) => ({
    filters: { ...s.filters, contentTypes: types },
  }));
}

export function setEncodingClassFilter(classes: Set<string>): void {
  store.setState((s) => ({
    filters: { ...s.filters, encodingClasses: classes },
  }));
}

export function setEncodingFormatFilter(formats: Set<string>): void {
  store.setState((s) => ({
    filters: { ...s.filters, encodingFormats: formats },
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

// --- Run preview ---

export async function previewRun(runId: string): Promise<void> {
  const resp = await fetch(`/api/runs/${runId}`);
  if (!resp.ok) return;
  const data = await resp.json();
  const ids = new Set<string>((data.nodes ?? []).map((n: { id: string }) => n.id));
  store.setState({ runNodeIds: ids, activeRunId: runId });
}

export function clearRunPreview(): void {
  store.setState({ runNodeIds: null, activeRunId: null });
}

export async function purgeActiveRun(): Promise<{ nodes_deleted: number; edges_deleted: number } | null> {
  const runId = store.getState().activeRunId;
  if (!runId) return null;
  const resp = await fetch(`/api/runs/${runId}`, { method: 'DELETE' });
  if (!resp.ok) return null;
  const result = await resp.json();
  store.setState({ runNodeIds: null, activeRunId: null });
  await loadFromApi();
  return result;
}

// --- Helpers ---

/** Get the temporal position for a node (for timeline / sorting). */
export function temporalPosition(node: Node): Date {
  return node.artifactCreatedAt ?? node.validFrom ?? node.createdAt;
}
