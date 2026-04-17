/**
 * @layer core
 * @description Level/class to color mapping for nodes and edges.
 * @depends core/types/nodes
 * @must-not Import from ui/ or graph/. Reference React or DOM.
 */

import type { Node, Edge } from './types/nodes';

const L0_BLUE = '#3b82f6';
const L1_AMBER = '#f59e0b';
const L2_ENTITY_EMERALD = '#10b981';
const L2_RELATION_PURPLE = '#a855f7';
const INFRA_GRAY = '#6b7280';

const EDGE_GRAY = '#6b7280';
const EDGE_EMERALD = '#10b981';

export function isInfraNode(node: Node): boolean {
  return node.contentClass === 'worker'
    || (node.contentClass === 'observation' && node.contentType === 'processed');
}

export function nodeColor(node: Node): string {
  if (isInfraNode(node)) return INFRA_GRAY;
  if (node.level === 0) return L0_BLUE;
  if (node.level === 1) return L1_AMBER;
  if (node.contentClass.startsWith('relation')) return L2_RELATION_PURPLE;
  return L2_ENTITY_EMERALD;
}

export function edgeColor(edge: Edge): string {
  if (edge.type === 'relation/head' || edge.type === 'relation/tail') {
    return EDGE_EMERALD;
  }
  return EDGE_GRAY;
}
