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

// Level palettes: [hueMin, hueMax, satMin, satMax, lightMin, lightMax]
// Each palette spans 60° of hue and ~25 points each of saturation and lightness,
// so content + encoding spread across a richer per-level color space.
const PALETTES = {
  infra: [200, 260,  5, 30, 35, 60],  // gray-blue, desaturated
  l0:    [200, 260, 55, 80, 40, 65],  // blue world
  l1:    [25,   85, 60, 85, 40, 65],  // amber/yellow world
  l2e:   [110, 170, 50, 75, 35, 60],  // green world (entities)
  l2r:   [280, 340, 50, 75, 40, 65],  // violet/magenta world (relations)
} as const;

function paletteColor(palette: readonly number[], contentKey: string, encodingKey: string): string {
  const [hMin, hMax, sMin, sMax, lMin, lMax] = palette;
  const h = hMin + hash01(contentKey) * (hMax - hMin);
  // Saturation hashes over the combined key so even same-content/same-encoding
  // remains stable, while different pairs scatter across the full sat window.
  const s = sMin + hash01(contentKey + '|' + encodingKey) * (sMax - sMin);
  const l = lMin + hash01(encodingKey) * (lMax - lMin);
  return `hsl(${Math.round(h)}, ${Math.round(s)}%, ${Math.round(l)}%)`;
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
