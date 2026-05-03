/**
 * @layer graph
 * @description Transforms core state into Cytoscape element definitions.
 * @depends core/types/nodes, core/colors
 * @must-not Import from ui/. Reference React or DOM. Depend on selection/emphasis state
 *           — emphasis is applied as data attributes by GraphView after layout.
 */

import type { Node, Edge } from '../core/types/nodes';
import { nodeColor, edgeColor, isInfraNode } from '../core/colors';
import type { ElementDefinition } from 'cytoscape';

export type HiddenInput = { nodes: Set<string>; edges: Set<string> };

export function toElements(
  nodes: Node[],
  edges: Edge[],
  hidden: HiddenInput,
): ElementDefinition[] {
  const elements: ElementDefinition[] = [];
  const visibleNodeIds = new Set<string>();

  for (const node of nodes) {
    if (hidden.nodes.has(node.id)) continue;
    visibleNodeIds.add(node.id);
    elements.push({
      data: {
        id: node.id,
        label: node.title ?? `${node.contentType}/${node.encodingFormat}`,
        level: node.level,
        contentClass: node.contentClass,
        contentType: node.contentType,
        color: nodeColor(node),
        infra: isInfraNode(node) ? 'yes' : 'no',
        emphasis: 'normal',
        invisible: 'no',
      },
    });
  }

  for (const edge of edges) {
    if (hidden.edges.has(edge.id)) continue;
    if (!visibleNodeIds.has(edge.sourceNodeId) || !visibleNodeIds.has(edge.targetNodeId)) continue;
    elements.push({
      data: {
        id: edge.id,
        source: edge.sourceNodeId,
        target: edge.targetNodeId,
        edgeType: edge.type,
        confidence: edge.confidence,
        color: edgeColor(edge),
        emphasis: 'normal',
        invisible: 'no',
      },
    });
  }

  return elements;
}
