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
import { store } from '../core/store';

cytoscape.use(dagre);
import { selectNode, selectEdge, clearSelection, setSelection, setToolMode } from '../core/actions';
import { filteredNodes, emphasisSets } from '../core/selectors';
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
  const selectedEdgeIds = useAppStore((s) => s.selectedEdgeIds);
  const hiddenNodeIds = useAppStore((s) => s.hiddenNodeIds);
  const hiddenEdgeIds = useAppStore((s) => s.hiddenEdgeIds);
  const emphasisMode = useAppStore((s) => s.emphasisMode);
  const deemphasisTreatment = useAppStore((s) => s.deemphasisTreatment);
  const runNodeIds = useAppStore((s) => s.runNodeIds);
  const toolMode = useAppStore((s) => s.toolMode);
  const priorToolModeRef = useRef<'pan' | 'select' | 'drag' | null>(null);

  // Topology elements — rebuilt only when structural inputs change.
  // Emphasis and hide treatment are applied in a separate effect as data attributes
  // so selection changes don't trigger a re-layout.
  const elements = useMemo(() => {
    const visibleNodes = filteredNodes();
    const visibleNodeIds = new Set(visibleNodes.map((n) => n.id));
    const visibleEdges = Array.from(edges.values()).filter(
      (e) => visibleNodeIds.has(e.sourceNodeId) && visibleNodeIds.has(e.targetNodeId)
    );
    return toElements(
      visibleNodes,
      visibleEdges,
      { nodes: hiddenNodeIds, edges: hiddenEdgeIds },
    );
  }, [nodes, edges, filters, hiddenNodeIds, hiddenEdgeIds]);

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
      boxSelectionEnabled: false,
      userZoomingEnabled: false,
    });

    // Manual wheel zoom — independent of mode, works even when
    // cy has userPanningEnabled(false) + boxSelectionEnabled(true)
    // (that combination silently kills cytoscape's internal wheel zoom).
    const container = containerRef.current;
    const onWheel = (e: WheelEvent) => {
      e.preventDefault();
      const factor = e.deltaY < 0 ? 1.1 : 1 / 1.1;
      const rect = container.getBoundingClientRect();
      cy.zoom({
        level: cy.zoom() * factor,
        renderedPosition: {
          x: e.clientX - rect.left,
          y: e.clientY - rect.top,
        },
      });
    };
    container.addEventListener('wheel', onWheel, { passive: false });

    // Node click → select in core (suppressed in pan mode)
    cy.on('tap', 'node', (evt) => {
      if (store.getState().toolMode === 'pan') return;
      const id = evt.target.id();
      selectNode(id, evt.originalEvent.shiftKey);
    });

    // Edge click → select in core (suppressed in pan mode)
    cy.on('tap', 'edge', (evt) => {
      if (store.getState().toolMode === 'pan') return;
      const id = evt.target.id();
      selectEdge(id, evt.originalEvent.shiftKey);
    });

    // Background click → clear selection (suppressed in pan mode)
    cy.on('tap', (evt) => {
      if (evt.target === cy && store.getState().toolMode !== 'pan') {
        clearSelection();
      }
    });

    // Marquee selection → sync to core at end of box gesture
    cy.on('boxend', () => {
      const nodeIds = cy.nodes(':selected').map((n) => n.id());
      const edgeIds = cy.edges(':selected').map((e) => e.id());
      setSelection(nodeIds, edgeIds);
    });

    cyRef.current = cy;

    return () => {
      container.removeEventListener('wheel', onWheel);
      cy.destroy();
      cyRef.current = null;
    };
  }, []);

  // Topology effect — rebuild elements and re-layout. Runs only when topology inputs change.
  useEffect(() => {
    const cy = cyRef.current;
    if (!cy) return;

    cy.batch(() => {
      cy.elements().remove();
      if (elements.length > 0) cy.add(elements);
    });

    if (elements.length === 0) return;

    cy.layout({
      name: 'dagre',
      rankDir: 'LR',
      nodeSep: 50,
      rankSep: 120,
      animate: false,
    } as any).run();

    cy.batch(() => {
      const LANE_GAP = 200;
      const MAX_PER_COL = 30;
      const COL_WIDTH = 80;
      const ROW_HEIGHT = 50;

      const byLevel = new Map<number, cytoscape.NodeSingular[]>();
      cy.nodes().forEach((n) => {
        const level = n.data('level') as number;
        if (!byLevel.has(level)) byLevel.set(level, []);
        byLevel.get(level)!.push(n);
      });

      const laneSizes = new Map<number, number>();
      const laneOrder = [2, 1, 0];

      for (const level of laneOrder) {
        const nodes = byLevel.get(level) ?? [];
        if (nodes.length === 0) {
          laneSizes.set(level, 0);
          continue;
        }

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
              y: row * ROW_HEIGHT,
            });
          }
        }
        laneSizes.set(level, maxRows * ROW_HEIGHT);
      }

      let yOffset = 0;
      for (const level of laneOrder) {
        const nodes = byLevel.get(level) ?? [];
        for (const n of nodes) {
          n.position('y', n.position('y') + yOffset);
        }
        yOffset += (laneSizes.get(level) ?? 0) + LANE_GAP;
      }
    });

    cy.fit(undefined, 30);
  }, [elements]);

  // Tool-mode effect — apply cytoscape interaction settings per mode.
  useEffect(() => {
    const cy = cyRef.current;
    if (!cy) return;

    switch (toolMode) {
      case 'pan':
        cy.userPanningEnabled(true);
        cy.autoungrabify(true);
        cy.autounselectify(true);
        cy.boxSelectionEnabled(false);
        break;
      case 'select':
        cy.userPanningEnabled(false);
        cy.autoungrabify(true);
        cy.autounselectify(false);
        cy.boxSelectionEnabled(true);
        break;
      case 'drag':
        cy.userPanningEnabled(true);
        cy.autoungrabify(false);
        cy.autounselectify(false);
        cy.boxSelectionEnabled(false);
        break;
    }
  }, [toolMode]);

  // Space-hold temporarily activates pan; release restores prior mode.
  useEffect(() => {
    const isTypingTarget = (t: EventTarget | null): boolean => {
      if (!(t instanceof HTMLElement)) return false;
      const tag = t.tagName;
      return tag === 'INPUT' || tag === 'TEXTAREA' || t.isContentEditable;
    };

    const onKeyDown = (e: KeyboardEvent) => {
      if (e.code !== 'Space' || e.repeat) return;
      if (isTypingTarget(e.target)) return;
      const current = store.getState().toolMode;
      if (current === 'pan') return;
      priorToolModeRef.current = current;
      setToolMode('pan');
      e.preventDefault();
    };
    const onKeyUp = (e: KeyboardEvent) => {
      if (e.code !== 'Space') return;
      if (priorToolModeRef.current) {
        setToolMode(priorToolModeRef.current);
        priorToolModeRef.current = null;
        e.preventDefault();
      }
    };

    window.addEventListener('keydown', onKeyDown);
    window.addEventListener('keyup', onKeyUp);
    return () => {
      window.removeEventListener('keydown', onKeyDown);
      window.removeEventListener('keyup', onKeyUp);
    };
  }, []);

  // Emphasis effect — recompute emphasis + invisibility via data attributes only.
  // Runs on every selection / mode / treatment / run-preview change. No layout.
  useEffect(() => {
    const cy = cyRef.current;
    if (!cy) return;

    const em = emphasisSets();

    cy.batch(() => {
      cy.nodes().forEach((n) => {
        const id = n.id();
        const emph =
          em.mode === 'none' ? 'normal'
          : em.highlightedNodeIds.has(id) ? 'highlighted'
          : em.normalNodeIds.has(id) ? 'normal'
          : 'dimmed';
        n.data('emphasis', emph);
        n.data('invisible', deemphasisTreatment === 'hide' && emph === 'dimmed' ? 'yes' : 'no');
      });
      cy.edges().forEach((e) => {
        const id = e.id();
        const emph =
          em.mode === 'none' ? 'normal'
          : em.highlightedEdgeIds.has(id) ? 'highlighted'
          : em.normalEdgeIds.has(id) ? 'normal'
          : 'dimmed';
        e.data('emphasis', emph);
        e.data('invisible', deemphasisTreatment === 'hide' && emph === 'dimmed' ? 'yes' : 'no');
      });
    });
  }, [elements, selectedNodeIds, selectedEdgeIds, emphasisMode, deemphasisTreatment, runNodeIds]);

  // Sync selection highlight (red border/line) — separate from emphasis.
  useEffect(() => {
    const cy = cyRef.current;
    if (!cy) return;

    cy.batch(() => {
      cy.elements().unselect();
      for (const id of selectedNodeIds) {
        const el = cy.getElementById(id);
        if (el.length > 0) el.select();
      }
      for (const id of selectedEdgeIds) {
        const el = cy.getElementById(id);
        if (el.length > 0) el.select();
      }
    });
  }, [selectedNodeIds, selectedEdgeIds]);

  return (
    <div className="graph-view">
      <div className={`graph-container graph-container--${toolMode}`} ref={containerRef} />
      {nodes.size === 0 && (
        <div className="graph-empty">
          No data loaded.
        </div>
      )}
    </div>
  );
}
