/**
 * @layer graph
 * @description Cytoscape visual styles for nodes and edges.
 * @depends (none)
 * @must-not Import from ui/ or core/. All styling is self-contained.
 */

// Cytoscape stylesheet — typed as any[] because cytoscape's TS types
// don't cleanly export the stylesheet interface for the style format we use.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const graphStylesheet: any[] = [
  // Nodes — base
  {
    selector: 'node',
    style: {
      'background-color': 'data(color)',
      'label': 'data(label)',
      'color': '#e0e0e0',
      'text-valign': 'bottom',
      'text-halign': 'center',
      'font-size': '10px',
      'text-margin-y': 5,
      'width': 30,
      'height': 30,
      'text-max-width': '100px',
      'text-wrap': 'ellipsis',
    },
  },
  // L0 — rectangle
  {
    selector: 'node[level=0]',
    style: {
      'shape': 'rectangle',
    },
  },
  // L1 — diamond
  {
    selector: 'node[level=1]',
    style: {
      'shape': 'diamond',
    },
  },
  // L2 — ellipse (default)
  {
    selector: 'node[level=2]',
    style: {
      'shape': 'ellipse',
    },
  },
  // Infrastructure nodes (worker config, processed markers)
  {
    selector: 'node[infra="yes"]',
    style: {
      'shape': 'hexagon',
      'width': 18,
      'height': 18,
      'font-size': '8px',
      'opacity': 0.6,
    },
  },
  // Selected
  {
    selector: ':selected',
    style: {
      'border-width': 3,
      'border-color': '#ef4444',
    },
  },
  // Edges — base
  {
    selector: 'edge',
    style: {
      'line-color': 'data(color)',
      'target-arrow-color': 'data(color)',
      'width': 1.5,
      'curve-style': 'bezier',
      'target-arrow-shape': 'triangle',
      'arrow-scale': 0.8,
    },
  },
  // Provenance/input — solid
  {
    selector: 'edge[edgeType="provenance/input"]',
    style: {
      'line-style': 'solid',
    },
  },
  // Provenance/worker — dashed
  {
    selector: 'edge[edgeType="provenance/worker"]',
    style: {
      'line-style': 'dashed',
    },
  },
  // Relation/head — arrow
  {
    selector: 'edge[edgeType="relation/head"]',
    style: {
      'target-arrow-shape': 'triangle',
    },
  },
  // Relation/tail — no arrow
  {
    selector: 'edge[edgeType="relation/tail"]',
    style: {
      'target-arrow-shape': 'none',
    },
  },
  // Dimmed (run preview)
  {
    selector: 'node[dimmed="yes"]',
    style: {
      'opacity': 0.15,
    },
  },
  {
    selector: 'edge[dimmed="yes"]',
    style: {
      'opacity': 0.08,
    },
  },
];
