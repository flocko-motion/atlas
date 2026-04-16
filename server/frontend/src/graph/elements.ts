/**
 * @layer graph
 * @description Transforms core state into Cytoscape element definitions.
 * @depends core/store, core/colors
 * @must-not Import from ui/. Reference React or DOM.
 */

import type { Node, Edge } from '../core/types/nodes';
import { nodeColor, edgeColor } from '../core/colors';
import type { ElementDefinition } from 'cytoscape';

export function nodesToElements(nodes: Node[]): ElementDefinition[] {
  return nodes.map((node) => ({
    data: {
      id: node.id,
      label: node.content
        ? node.content.slice(0, 40)
        : `${node.contentClass}/${node.contentType}`,
      level: node.level,
      contentClass: node.contentClass,
      contentType: node.contentType,
      color: nodeColor(node),
    },
  }));
}

export function edgesToElements(edges: Edge[]): ElementDefinition[] {
  return edges.map((edge) => ({
    data: {
      id: edge.id,
      source: edge.sourceNodeId,
      target: edge.targetNodeId,
      edgeType: edge.type,
      confidence: edge.confidence,
      color: edgeColor(edge),
    },
  }));
}

export function toElements(nodes: Node[], edges: Edge[]): ElementDefinition[] {
  return [...nodesToElements(nodes), ...edgesToElements(edges)];
}
