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

      // Spread stacked nodes into a grid when too many share the same x
      const MAX_PER_COL = 30;
      const COL_WIDTH = 80;
      const ROW_HEIGHT = 50;
      const byX = new Map<number, cytoscape.NodeSingular[]>();
      cy.nodes().forEach((n) => {
        const x = Math.round(n.position('x'));
        if (!byX.has(x)) byX.set(x, []);
        byX.get(x)!.push(n);
      });
      for (const [baseX, group] of byX) {
        if (group.length <= MAX_PER_COL) continue;
        // Sort by current y to preserve relative order
        group.sort((a, b) => a.position('y') - b.position('y'));
        for (let i = 0; i < group.length; i++) {
          const col = Math.floor(i / MAX_PER_COL);
          const row = i % MAX_PER_COL;
          group[i].position({
            x: baseX + col * COL_WIDTH,
            y: row * ROW_HEIGHT,
          });
        }
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
