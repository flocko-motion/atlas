/**
 * @layer graph
 * @description Cytoscape graph renderer. Subscribes to core state, dispatches core actions.
 * @depends core/hooks, core/actions, core/selectors, graph/elements, graph/stylesheet
 * @must-not Contain business logic. Manage application state beyond layout/zoom.
 */

import { useEffect, useRef, useMemo } from 'react';
import cytoscape from 'cytoscape';
import dagre from 'cytoscape-dagre';
import { useAppStore } from '../core/hooks';

cytoscape.use(dagre);
import { selectNode, clearSelection } from '../core/actions';
import { filteredNodes } from '../core/selectors';
import { toElements } from './elements';
import { graphStylesheet } from './stylesheet';
import './GraphView.css';

export function GraphView() {
  const containerRef = useRef<HTMLDivElement>(null);
  const cyRef = useRef<cytoscape.Core | null>(null);
  const nodes = useAppStore((s) => s.nodes);
  const edges = useAppStore((s) => s.edges);
  const filters = useAppStore((s) => s.filters);
  const selectedNodeIds = useAppStore((s) => s.selectedNodeIds);
  const highlightedNodeIds = useAppStore((s) => s.highlightedNodeIds);

  const elements = useMemo(() => {
    const visibleNodes = filteredNodes();
    const visibleNodeIds = new Set(visibleNodes.map((n) => n.id));
    const visibleEdges = Array.from(edges.values()).filter(
      (e) => visibleNodeIds.has(e.sourceNodeId) && visibleNodeIds.has(e.targetNodeId)
    );
    return toElements(visibleNodes, visibleEdges, highlightedNodeIds);
  }, [nodes, edges, filters, highlightedNodeIds]);

  // Initialize Cytoscape
  useEffect(() => {
    if (!containerRef.current) return;

    const cy = cytoscape({
      container: containerRef.current,
      elements: [],
      style: graphStylesheet,
      layout: { name: 'grid' },
      minZoom: 0.1,
      maxZoom: 5,
      autoungrabify: true,
    });

    // Node click → select in core
    cy.on('tap', 'node', (evt) => {
      const id = evt.target.id();
      selectNode(id, evt.originalEvent.shiftKey);
    });

    // Background click → clear selection
    cy.on('tap', (evt) => {
      if (evt.target === cy) {
        clearSelection();
      }
    });

    cyRef.current = cy;

    return () => {
      cy.destroy();
      cyRef.current = null;
    };
  }, []);

  // Update elements when data changes
  useEffect(() => {
    const cy = cyRef.current;
    if (!cy) return;

    cy.elements().remove();
    if (elements.length > 0) {
      cy.add(elements);
      cy.layout({
        name: 'dagre',
        rankDir: 'LR',
        nodeSep: 50,
        rankSep: 120,
        animate: false,
      } as any).run();

      // Post-layout: assign level lanes (L2 top, L1 mid, L0 bottom)
      // and grid-spread within each lane
      const LANE_GAP = 200;      // gap between lanes
      const MAX_PER_COL = 30;
      const COL_WIDTH = 80;
      const ROW_HEIGHT = 50;

      // Group nodes by level, then by dagre x-rank within each level
      const byLevel = new Map<number, cytoscape.NodeSingular[]>();
      cy.nodes().forEach((n) => {
        const level = n.data('level') as number;
        if (!byLevel.has(level)) byLevel.set(level, []);
        byLevel.get(level)!.push(n);
      });

      // Lane y-offsets: L2=0, L1=below L2, L0=below L1
      const laneSizes = new Map<number, number>(); // level → height used
      const laneOrder = [2, 1, 0]; // top to bottom

      for (const level of laneOrder) {
        const nodes = byLevel.get(level) ?? [];
        if (nodes.length === 0) {
          laneSizes.set(level, 0);
          continue;
        }

        // Sub-group by dagre x position
        const byX = new Map<number, cytoscape.NodeSingular[]>();
        for (const n of nodes) {
          const x = Math.round(n.position('x'));
          if (!byX.has(x)) byX.set(x, []);
          byX.get(x)!.push(n);
        }

        let maxRows = 0;
        for (const [baseX, group] of byX) {
          group.sort((a, b) => a.position('y') - b.position('y'));
          const rows = Math.min(group.length, MAX_PER_COL);
          if (rows > maxRows) maxRows = rows;
          for (let i = 0; i < group.length; i++) {
            const col = Math.floor(i / MAX_PER_COL);
            const row = i % MAX_PER_COL;
            group[i].position({
              x: baseX + col * COL_WIDTH,
              y: row * ROW_HEIGHT, // relative to lane, offset applied next
            });
          }
        }
        laneSizes.set(level, maxRows * ROW_HEIGHT);
      }

      // Apply lane y-offsets
      let yOffset = 0;
      for (const level of laneOrder) {
        const nodes = byLevel.get(level) ?? [];
        for (const n of nodes) {
          n.position('y', n.position('y') + yOffset);
        }
        yOffset += (laneSizes.get(level) ?? 0) + LANE_GAP;
      }

      cy.fit(undefined, 30);
    }
  }, [elements]);

  // Sync selection from core → Cytoscape
  useEffect(() => {
    const cy = cyRef.current;
    if (!cy) return;

    cy.elements().unselect();
    for (const id of selectedNodeIds) {
      const el = cy.getElementById(id);
      if (el.length > 0) el.select();
    }
  }, [selectedNodeIds]);

  return (
    <div className="graph-view">
      <div className="graph-container" ref={containerRef} />
      {nodes.size === 0 && (
        <div className="graph-empty">
          No data loaded.
        </div>
      )}
    </div>
  );
}
