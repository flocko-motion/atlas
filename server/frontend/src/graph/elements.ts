/**
 * @layer graph
 * @description Transforms core state into Cytoscape element definitions.
 * @depends core/store, core/colors
 * @must-not Import from ui/. Reference React or DOM.
 */

import type { Node, Edge } from '../core/types/nodes';
import { nodeColor, edgeColor, isInfraNode } from '../core/colors';
import type { ElementDefinition } from 'cytoscape';

export function nodesToElements(nodes: Node[], highlight: Set<string> | null): ElementDefinition[] {
  return nodes.map((node) => ({
    data: {
      id: node.id,
      label: node.title
        ?? `${node.contentType}/${node.encodingFormat}`,
      level: node.level,
      contentClass: node.contentClass,
      contentType: node.contentType,
      color: nodeColor(node),
      infra: isInfraNode(node) ? 'yes' : 'no',
      dimmed: highlight && !highlight.has(node.id) ? 'yes' : 'no',
    },
  }));
}

export function edgesToElements(edges: Edge[], highlight: Set<string> | null): ElementDefinition[] {
  return edges.map((edge) => ({
    data: {
      id: edge.id,
      source: edge.sourceNodeId,
      target: edge.targetNodeId,
      edgeType: edge.type,
      confidence: edge.confidence,
      color: edgeColor(edge),
      dimmed: highlight && !(highlight.has(edge.sourceNodeId) && highlight.has(edge.targetNodeId)) ? 'yes' : 'no',
    },
  }));
}

export function toElements(nodes: Node[], edges: Edge[], highlight: Set<string> | null = null): ElementDefinition[] {
  return [...nodesToElements(nodes, highlight), ...edgesToElements(edges, highlight)];
}
