/**
 * @layer core
 * @description Level/class to color mapping for nodes and edges.
 * @depends core/types/nodes
 * @must-not Import from ui/ or graph/. Reference React or DOM.
 */

import type { Node, Edge } from './types/nodes';

// Simple string hash → 0..1
function hash01(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) {
    h = ((h << 5) - h + s.charCodeAt(i)) | 0;
  }
  return ((h >>> 0) % 1000) / 1000;
}

// Level palettes: [hueMin, hueMax, saturation, lightnessMin, lightnessMax]
const PALETTES = {
  infra: [220, 260, 15, 35, 50],    // gray-blue, desaturated
  l0:    [200, 240, 65, 45, 60],    // blue world
  l1:    [30, 60, 70, 45, 60],      // yellow/amber world
  l2e:   [140, 180, 55, 40, 55],    // green world (entities)
  l2r:   [260, 300, 55, 45, 60],    // purple world (relations)
} as const;

function paletteColor(palette: readonly number[], contentKey: string, encodingKey: string): string {
  const [hMin, hMax, sat, lMin, lMax] = palette;
  const h = hMin + hash01(contentKey) * (hMax - hMin);
  const l = lMin + hash01(encodingKey) * (lMax - lMin);
  return `hsl(${Math.round(h)}, ${sat}%, ${Math.round(l)}%)`;
}

export function isInfraNode(node: Node): boolean {
  return node.contentClass === 'worker'
    || (node.contentClass === 'observation' && node.contentType === 'processed');
}

export function nodeColor(node: Node): string {
  const contentKey = `${node.contentClass}/${node.contentType}`;
  const encodingKey = `${node.encodingClass}/${node.encodingFormat}`;

  if (isInfraNode(node)) return paletteColor(PALETTES.infra, contentKey, encodingKey);
  if (node.level === 0) return paletteColor(PALETTES.l0, contentKey, encodingKey);
  if (node.level === 1) return paletteColor(PALETTES.l1, contentKey, encodingKey);
  if (node.contentClass === 'relation') return paletteColor(PALETTES.l2r, contentKey, encodingKey);
  return paletteColor(PALETTES.l2e, contentKey, encodingKey);
}

const EDGE_GRAY = '#6b7280';
const EDGE_EMERALD = '#10b981';

export function edgeColor(edge: Edge): string {
  if (edge.type === 'relation/head' || edge.type === 'relation/tail') {
    return EDGE_EMERALD;
  }
  return EDGE_GRAY;
}
