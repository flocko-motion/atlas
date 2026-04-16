/**
 * @layer core
 * @description Derived/computed state from the Zustand store.
 * @depends core/store, core/types/nodes, core/actions
 * @must-not Import from ui/ or graph/. Reference React or DOM.
 */

import { store } from './store';
import type { Node, Edge, Relation } from './types/nodes';
import { temporalPosition } from './actions';

export function selectedNodes(): Node[] {
  const { selectedNodeIds, nodes } = store.getState();
  const result: Node[] = [];
  for (const id of selectedNodeIds) {
    const node = nodes.get(id);
    if (node) result.push(node);
  }
  return result;
}

export function primarySelectedNode(): Node | null {
  const { selectedNodeIds, nodes } = store.getState();
  const first = selectedNodeIds.values().next().value;
  if (!first) return null;
  return nodes.get(first) ?? null;
}

export function filteredNodes(): Node[] {
  const { nodes, filters } = store.getState();
  const result: Node[] = [];

  for (const node of nodes.values()) {
    if (!filters.levels.has(node.level)) continue;
    if (filters.contentClasses.size > 0 && !filters.contentClasses.has(node.contentClass)) continue;
    if (filters.dateRange.from || filters.dateRange.to) {
      const d = temporalPosition(node);
      if (filters.dateRange.from && d < filters.dateRange.from) continue;
      if (filters.dateRange.to && d > filters.dateRange.to) continue;
    }
    if (node.confidence !== null && node.confidence < filters.minConfidence) continue;
    result.push(node);
  }

  return result;
}

export function timelineNodes(): Array<Node & { temporalPosition: Date }> {
  return filteredNodes().map((n) => ({
    ...n,
    temporalPosition: temporalPosition(n),
  }));
}

export function nodeEdges(nodeId: string): Edge[] {
  const { edges } = store.getState();
  const result: Edge[] = [];
  for (const edge of edges.values()) {
    if (edge.sourceNodeId === nodeId || edge.targetNodeId === nodeId) {
      result.push(edge);
    }
  }
  return result;
}

export function provenanceChain(nodeId: string): { nodes: Node[]; edges: Edge[] } {
  const { nodes, edges } = store.getState();
  const visitedNodes = new Set<string>();
  const resultEdges: Edge[] = [];
  const queue = [nodeId];

  while (queue.length > 0) {
    const current = queue.pop()!;
    if (visitedNodes.has(current)) continue;
    visitedNodes.add(current);

    for (const edge of edges.values()) {
      if (
        edge.sourceNodeId === current &&
        (edge.type === 'provenance/input' || edge.type === 'provenance/worker')
      ) {
        resultEdges.push(edge);
        if (!visitedNodes.has(edge.targetNodeId)) {
          queue.push(edge.targetNodeId);
        }
      }
    }
  }

  const resultNodes: Node[] = [];
  for (const id of visitedNodes) {
    const node = nodes.get(id);
    if (node) resultNodes.push(node);
  }

  return { nodes: resultNodes, edges: resultEdges };
}

export function entityRelations(entityId: string): Relation[] {
  const { nodes, edges } = store.getState();
  const relations: Relation[] = [];
  const seen = new Set<string>();

  for (const edge of edges.values()) {
    if (
      (edge.type === 'relation/head' || edge.type === 'relation/tail') &&
      edge.targetNodeId === entityId
    ) {
      const relNodeId = edge.sourceNodeId;
      if (seen.has(relNodeId)) continue;
      seen.add(relNodeId);

      const relNode = nodes.get(relNodeId);
      if (!relNode) continue;

      const headEdges: Edge[] = [];
      const tailEdges: Edge[] = [];
      for (const e of edges.values()) {
        if (e.sourceNodeId === relNodeId && e.type === 'relation/head') headEdges.push(e);
        if (e.sourceNodeId === relNodeId && e.type === 'relation/tail') tailEdges.push(e);
      }
      relations.push({ node: relNode, headEdges, tailEdges });
    }
  }

  return relations;
}

export function stats(): {
  nodeCount: number;
  edgeCount: number;
  l0Count: number;
  l1Count: number;
  l2Count: number;
} {
  const { nodes } = store.getState();
  let l0 = 0, l1 = 0, l2 = 0;
  for (const n of nodes.values()) {
    if (n.level === 0) l0++;
    else if (n.level === 1) l1++;
    else l2++;
  }
  return { nodeCount: nodes.size, edgeCount: store.getState().edges.size, l0Count: l0, l1Count: l1, l2Count: l2 };
}
