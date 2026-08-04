/**
 * package: render / renderer
 * type:    adapter
 * job:     own the Sigma instance — the only module that knows a renderer exists
 * limits:  module-scoped, outside framework state (-> ui, core/store)
 *
 * Built once and never recreated by a re-render: React holds a handle, never graph data.
 * A view is a selection applied through reducers over the one union graph, never a graph
 * of its own.
 */

import Sigma from 'sigma';
import { graph } from '../core/graph/universe.ts';
import { inScope } from '../core/session.ts';
import { useExplorer, activeView } from '../core/store.ts';
import type { ViewState } from '../core/store.ts';

/** How long the pointer must rest before the preview updates. */
const HOVER_DEBOUNCE_MS = 120;

let sigma: Sigma | null = null;
let fpsHandle = 0;
let frames = 0;
let lastSample = 0;
let lastFrameAt = 0;
let worstGap = 0;
/** Counted by the reducers during a full refresh, published after it. */
let admittedNodes = 0;
let admittedEdges = 0;
let counting = false;
let hoverTimer = 0;

/** current returns the live Sigma instance, if the canvas has been mounted. */
export function current(): Sigma | null {
  return sigma;
}

/** rendererName reports the GPU behind the canvas — software or real. */
function rendererName(): string {
  const canvas = document.createElement('canvas');
  const gl = canvas.getContext('webgl2') ?? canvas.getContext('webgl');
  if (!gl) return 'none';
  const dbg = gl.getExtension('WEBGL_debug_renderer_info');
  return String(dbg ? gl.getParameter(dbg.UNMASKED_RENDERER_WEBGL) : gl.getParameter(gl.RENDERER));
}

/** admits decides whether a claim is in a view's selection. */
function admits(view: ViewState, node: string, contribution: number, cls: string): boolean {
  // The id set is the source's answer to "what is in this scope"; this only looks in it.
  if (view.scope && !inScope(view.scope, node)) return false;
  if (view.contributionRange) {
    const [lo, hi] = view.contributionRange;
    if (contribution < lo || contribution > hi) return false;
  }
  if (view.classes.length > 0 && !view.classes.includes(cls)) return false;
  return true;
}

/**
 * mount attaches Sigma to a container. Called once by the shell; subsequent calls
 * with the same container are ignored so a re-render cannot disturb the renderer.
 */
export function mount(container: HTMLElement): Sigma {
  if (sigma && sigma.getContainer() === container) return sigma;
  sigma?.kill();

  const store = useExplorer.getState();
  sigma = new Sigma(graph(), container, {
    // Straight, thin edges by default; decoration is a zoom-level decision.
    renderEdgeLabels: false,
    enableEdgeEvents: false,
    allowInvalidContainer: true,
    // The label budget, and the cap itself: at most `labelDensity` labels per
    // `labelGridCellSize` px of viewport, nothing below the size threshold.
    labelRenderedSizeThreshold: 8,
    labelDensity: 1,
    labelGridCellSize: 120,
    nodeReducer: (node, data) => {
      const view = activeView(useExplorer.getState());
      if (!view) return data;
      const visible = admits(view, node, data.contribution as number, data.cls as string);
      if (!visible) return { ...data, hidden: true };
      if (counting) admittedNodes++;
      const { selected, hovered } = useExplorer.getState().selection;
      if (node === selected) return { ...data, highlighted: true, zIndex: 2 };
      if (node === hovered) return { ...data, highlighted: true, zIndex: 1 };
      return data;
    },
    edgeReducer: (edge, data) => {
      const view = activeView(useExplorer.getState());
      if (!view) return data;
      if (!view.edges) return { ...data, hidden: true };
      const g = graph();
      const source = g.source(edge);
      const target = g.target(edge);
      const admitted =
        admits(view, source, g.getNodeAttribute(source, 'contribution') as number, g.getNodeAttribute(source, 'cls') as string) &&
        admits(view, target, g.getNodeAttribute(target, 'contribution') as number, g.getNodeAttribute(target, 'cls') as string);
      if (admitted && counting) admittedEdges++;
      return admitted ? data : { ...data, hidden: true };
    },
  });

  applyViewSettings(activeView(store));
  bindEvents(sigma);
  store.patchStatus({ renderer: rendererName() });
  return sigma;
}

/**
 * applyViewSettings pushes a view's render flags into Sigma — settings, not reducer
 * output, so they cost nothing to change (unlike `hidden`, which re-indexes).
 */
export function applyViewSettings(view: ViewState | null): void {
  if (!sigma || !view) return;
  sigma.setSetting('renderLabels', view.labels);
  sigma.setSetting('hideLabelsOnMove', !view.labelsOnMove);
  sigma.setSetting('hideEdgesOnMove', !view.edgesOnMove);
}

/**
 * refreshSelection re-runs the reducers when a selection changes: O(N), a re-index plus
 * a full upload, timed and published so a slow frame can be attributed.
 */
export function refreshSelection(): void {
  if (!sigma) return;
  admittedNodes = 0;
  admittedEdges = 0;
  counting = true;
  const t0 = performance.now();
  sigma.refresh();
  const lastRefreshMs = performance.now() - t0;
  counting = false;
  useExplorer.getState().patchStatus({
    lastRefreshMs,
    visibleNodes: admittedNodes,
    visibleEdges: admittedEdges,
  });
}

/** highlight repaints only the affected nodes — hover must never cost a full O(N) refresh. */
export function highlight(nodes: string[]): void {
  if (!sigma || nodes.length === 0) return;
  const present = nodes.filter((n) => graph().hasNode(n));
  if (present.length === 0) return;
  sigma.refresh({ partialGraph: { nodes: present }, skipIndexation: true });
}

/** bindEvents wires pointer interaction into core state, hover apart from selection. */
function bindEvents(instance: Sigma): void {
  const store = useExplorer.getState();

  instance.on('clickNode', ({ node }) => {
    const previous = useExplorer.getState().selection.selected;
    useExplorer.getState().select(node);
    highlight([node, ...(previous ? [previous] : [])]);
  });

  instance.on('clickStage', () => {
    const previous = useExplorer.getState().selection.selected;
    useExplorer.getState().select(null);
    if (previous) highlight([previous]);
  });

  // Hover drives only a partial repaint: the highlight is immediate (two nodes), the
  // preview debounced so a sweeping pointer cannot outrun the reader.
  instance.on('enterNode', ({ node }) => {
    const previous = useExplorer.getState().selection.hovered;
    useExplorer.getState().hover(node);
    highlight([node, ...(previous ? [previous] : [])]);
    window.clearTimeout(hoverTimer);
    hoverTimer = window.setTimeout(() => {
      if (useExplorer.getState().selection.hovered !== node) return;
      const g = graph();
      if (!g.hasNode(node)) return;
      useExplorer.getState().setPreview({
        id: node,
        label: String(g.getNodeAttribute(node, 'label') ?? ''),
        claimType: String(g.getNodeAttribute(node, 'claimType') ?? ''),
      });
    }, HOVER_DEBOUNCE_MS);
  });

  instance.on('leaveNode', ({ node }) => {
    window.clearTimeout(hoverTimer);
    useExplorer.getState().hover(null);
    useExplorer.getState().setPreview(null);
    highlight([node]);
  });

  instance.on('afterRender', () => {
    frames++;
    const now = performance.now();
    if (lastFrameAt > 0) worstGap = Math.max(worstGap, now - lastFrameAt);
    lastFrameAt = now;
  });

  cancelAnimationFrame(fpsHandle);
  lastSample = performance.now();
  const tick = () => {
    const now = performance.now();
    if (now - lastSample >= 500) {
      const fps = (frames / (now - lastSample)) * 1000;
      store.patchStatus({
        fps,
        frameMs: frames > 0 ? (now - lastSample) / frames : null,
        stallMs: worstGap > 0 ? worstGap : null,
      });
      frames = 0;
      worstGap = 0;
      lastSample = now;
    }
    fpsHandle = requestAnimationFrame(tick);
  };
  fpsHandle = requestAnimationFrame(tick);
}

/** unmount tears the renderer down — only on page teardown, never on re-render. */
export function unmount(): void {
  cancelAnimationFrame(fpsHandle);
  sigma?.kill();
  sigma = null;
}
