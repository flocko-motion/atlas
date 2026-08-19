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
import { createEdgeArrowProgram } from 'sigma/rendering';
import type { Settings } from 'sigma/settings';
import type { DirectedGraph } from 'graphology';
import { contentOf } from '../core/content.ts';
import { brighten } from '../core/graph/build.ts';
import { graph } from '../core/graph/universe.ts';
import { inScope } from '../core/session.ts';
import { useExplorer, activeView } from '../core/store.ts';
import type { ViewState } from '../core/store.ts';
import { applyBound, holdCamera, panIntoView, zoomToBox } from './camera.ts';
import type { Box } from './camera.ts';
import {
  hostOf,
  lensOf,
  lensShowing,
  setHost,
  setLens,
  setUnion,
  showing,
  shownGraph,
  unionOf,
} from './instances.ts';
import { drawNodeLabel } from './labels.ts';
import { pin } from './hold.ts';

/** How long the pointer must rest before the preview updates. */
const HOVER_DEBOUNCE_MS = 120;

/** How much of a claim's content the hover tooltip quotes. */
const PREVIEW_CONTENT_CHARS = 200;

/**
 * contentSnippet is the hover tooltip's second line: a prefix of a claim's content, where it is
 * both text and already read. It never fetches — a pointer sweeping over a thousand nodes must
 * not start a request per node — so a claim whose bytes have not been read this session, or
 * whose encoding is not text, quotes nothing.
 */
function contentSnippet(id: string): string {
  const encoding = graph().getNodeAttribute(id, 'encoding') as string | undefined;
  const textual =
    !!encoding &&
    (encoding.startsWith('text/') ||
      encoding === 'application/json' ||
      encoding === 'application/xml' ||
      encoding.endsWith('+json') ||
      encoding.endsWith('+xml'));
  if (!textual) return '';
  const bytes = contentOf(id);
  if (!bytes) return '';
  const text = new TextDecoder('utf-8', { fatal: false }).decode(bytes).trim();
  if (text.length === 0) return '';
  return text.length > PREVIEW_CONTENT_CHARS ? `${text.slice(0, PREVIEW_CONTENT_CHARS)}…` : text;
}

// Sigma's stock arrowhead scales off edge thickness (`lengthToThicknessRatio` /
// `widenessToThicknessRatio`, defaults 2.5 / 2) — against this app's thin edges the result is a
// sliver, hard to see which way an edge points. Built once at module scope, like the Sigma
// instance itself: a WebGL program is not something to rebuild per render.
const BIG_ARROW_PROGRAM = createEdgeArrowProgram({ lengthToThicknessRatio: 5, widenessToThicknessRatio: 4 });

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
  return unionOf();
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
  const held = unionOf();
  if (held && held.getContainer() === container) return held;
  held?.kill();

  const store = useExplorer.getState();
  const instance = new Sigma(graph(), container, sigmaSettings());
  setUnion(instance);
  applyViewSettings(activeView(store));
  bindEvents(instance);
  store.patchStatus({ renderer: rendererName() });
  return instance;
}

/**
 * sigmaSettings is what every instance is built with. Shared because the lens and the union
 * are drawn by two instances and a reader must not be able to tell which one is showing.
 */
function sigmaSettings(): Partial<Settings> {
  return {
    // Arrows: every edge points from a claim to what it cites, and a graph of provenance
    // without direction drawn is not readable.
    defaultEdgeType: 'arrow',
    // Every edge's own label text is blank except the selected claim's (-> edgeReducer), so
    // turning this on does not draw a caption on every edge — Sigma's own default drawer
    // already no-ops on a blank label (rendering/programs/edge-label.js), which is what makes
    // this safe to leave on rather than a setting to flip per selection.
    renderEdgeLabels: true,
    // Edges are selectable, so the detail pane can answer about one.
    enableEdgeEvents: true,
    allowInvalidContainer: true,
    // Sigma's default label colour is black, which on this theme is black on black.
    labelColor: { color: '#e6e8ee' },
    // Fallback only — every edge carries its own class/subtype colour from the graph
    // (-> core/graph/build.ts colorFor); this is what an edge without one would show.
    defaultEdgeColor: '#4a4f5c',
    // The stock arrowhead, scaled up (-> BIG_ARROW_PROGRAM) so direction reads at a glance.
    edgeProgramClasses: { arrow: BIG_ARROW_PROGRAM },
    defaultDrawNodeLabel: drawNodeLabel,
    // No plate on hover: Sigma's own default draws one, light and hard to read on a dark
    // canvas, and the highlight below (bigger, raised z-index) already marks the node — the
    // hover tooltip is a DOM overlay instead (-> ui/shell/App HoverPreviewChip).
    defaultDrawNodeHover: () => {},
    // The label budget, and the cap itself: at most `labelDensity` labels per
    // `labelGridCellSize` px of viewport, nothing below the size threshold. Sigma picks the
    // `labelDensity` winners in a cell by size alone, with no separation check between them —
    // so a cell density above 2 lets same-cell winners land close enough to collide once rows
    // compress (a low `yStretch` in the timeline layout); 2 keeps most of the gain without that.
    labelRenderedSizeThreshold: 3,
    labelDensity: 2,
    labelGridCellSize: 80,
    nodeReducer: (node, data) => {
      const view = activeView(useExplorer.getState());
      if (!view) return data;
      const visible = admits(view, node, data.contribution as number, data.cls as string);
      if (!visible) return { ...data, hidden: true };
      if (counting) admittedNodes++;
      const { selected, hovered } = useExplorer.getState().selection;
      let out = data;
      // A selected claim's caption is the one the reader asked for; every other caption steps
      // aside for it rather than compete for the same small patch of canvas — cheap because the
      // label-density budget already bounds how many were showing to begin with (-> renderer.ts
      // labelBearingNodes), not by how large the loaded graph is.
      if (selected && node !== selected) out = { ...out, label: '' };
      const own = String(out.color ?? '#999999');
      if (node === selected) {
        // forceLabel bypasses Sigma's own size/density gate, so the selected claim's caption
        // shows even where it would not have earned one on its own.
        return { ...out, color: brighten(own, NODE_SELECTED_BRIGHTEN), highlighted: true, forceLabel: true, zIndex: 2 };
      }
      if (node === hovered) {
        return { ...out, color: brighten(own, NODE_HOVERED_BRIGHTEN), highlighted: true, zIndex: 1 };
      }
      return out;
    },
    edgeReducer: (edge, data) => {
      const view = activeView(useExplorer.getState());
      if (!view) return data;
      if (!view.edges) return { ...data, hidden: true };
      const g = shownGraph() ?? graph();
      const source = g.source(edge);
      const target = g.target(edge);
      const admitted =
        admits(view, source, g.getNodeAttribute(source, 'contribution') as number, g.getNodeAttribute(source, 'cls') as string) &&
        admits(view, target, g.getNodeAttribute(target, 'contribution') as number, g.getNodeAttribute(target, 'cls') as string);
      if (!admitted) return { ...data, hidden: true };
      if (counting) admittedEdges++;

      const { selected, selectedEdge } = useExplorer.getState().selection;
      const own = String(data.color ?? '#4a4f5c');
      // A caption on every edge at once is unreadable, so an edge names itself only once a
      // claim is selected and only where it touches that claim — the few edges a reader is
      // actually asking about, not the whole picture.
      const label = selected && (source === selected || target === selected)
        ? String(data.claimType ?? '').split('/')[1] ?? ''
        : '';
      // A claim is a node and its outgoing edges, so the edges of a selected claim are
      // selected with it — brightened rather than recoloured, so the highlight still says
      // what class and subtype the edge is, not just that it is highlighted.
      if (edge === selectedEdge || source === selected) {
        return { ...data, color: brighten(own, EDGE_OWN_BRIGHTEN), size: 2, zIndex: 2, label };
      }
      // An edge pointing *at* the selection belongs to somebody else — a citation, which the
      // claim could not know when it was written. A lighter brighten than an own edge's, so the
      // two directions are still told apart on the canvas as they are in the pane.
      if (target === selected) {
        return { ...data, color: brighten(own, EDGE_CITATION_BRIGHTEN), size: 2, zIndex: 2, label };
      }
      // Otherwise the edge's own class/subtype colour stands as built (-> build.ts colorFor):
      // contribution/* is already the darkest, greyest family, so the structural spine recedes
      // without a special case here.
      return { ...data, label };
    },
  };
}

/** How far a highlighted edge's own colour is brightened (-> core/graph/build.ts brighten). */
const EDGE_OWN_BRIGHTEN = 0.55;
const EDGE_CITATION_BRIGHTEN = 0.3;

/** How far a highlighted node's own colour is brightened. */
const NODE_SELECTED_BRIGHTEN = 0.5;
const NODE_HOVERED_BRIGHTEN = 0.3;

/**
 * applyViewSettings pushes a view's render flags into Sigma — settings, not reducer
 * output, so they cost nothing to change (unlike `hidden`, which re-indexes).
 */
export function applyViewSettings(view: ViewState | null, instance: Sigma | null = unionOf()): void {
  if (!instance || !view) return;
  instance.setSetting('renderLabels', view.labels);
  instance.setSetting('hideLabelsOnMove', !view.labelsOnMove);
  instance.setSetting('hideEdgesOnMove', !view.edgesOnMove);
}

/**
 * refreshSelection re-runs the reducers when a selection changes: O(N), a re-index plus
 * a full upload, timed and published so a slow frame can be attributed.
 */
export function refreshSelection(): void {
  const instance = unionOf();
  if (!instance) return;
  admittedNodes = 0;
  admittedEdges = 0;
  counting = true;
  const t0 = performance.now();
  instance.refresh();
  const lastRefreshMs = performance.now() - t0;
  counting = false;
  useExplorer.getState().patchStatus({
    lastRefreshMs,
    visibleNodes: admittedNodes,
    visibleEdges: admittedEdges,
  });
}

/**
 * highlight repaints the affected nodes and the edges they own — hover must never cost a full
 * O(N) refresh.
 *
 * A claim is a node *and its outgoing edges*, so highlighting one means highlighting both:
 * asking about a claim and being shown only its dot would leave out what it cites, which is
 * most of what a claim says.
 */
export function highlight(nodes: string[]): void {
  const instance = showing();
  if (!instance) return;
  const drawn = instance.getGraph();
  const present = nodes.filter((n) => drawn.hasNode(n));
  if (present.length === 0) return;

  // Both directions: a claim's own edges are drawn with it, and the edges citing it are drawn
  // against it, so a change of selection has to repaint the edges on either side.
  const edges: string[] = [];
  for (const node of present) {
    drawn.forEachEdge(node, (edge) => edges.push(edge));
  }
  // No skipIndexation: the extent is pinned for the timeline, and skipping indexation makes
  // a partial refresh re-derive it from the nodes it was given — which moved the whole
  // picture on every hover.
  instance.refresh({ partialGraph: { nodes: present, edges } });
}

/**
 * labelBearingNodes are the nodes Sigma is currently drawing a caption for. A change of
 * selection blanks every caption but the selected claim's own (-> nodeReducer), which only
 * takes effect where the reducer actually re-runs — so a selection change's partial refresh
 * must include this set, on top of the claim itself. Cheap regardless of how large the loaded
 * graph is: the label-density budget already bounds how many captions are ever on screen at
 * once, so this set never grows past that budget.
 */
function labelBearingNodes(): string[] {
  const instance = showing();
  return instance ? [...instance.getNodeDisplayedLabels()] : [];
}

/**
 * edgeOwner is the node whose repaint clears a stale edge's highlight — an edge belongs to the
 * claim it points from, so repainting that claim's edges is what re-runs the edge's own
 * reducer. Every selection-changing call site below must pass its *previous* `selectedEdge`
 * through this before the state change: node and edge selection are mutually exclusive at the
 * store (selecting one clears the other), so a caller that only ever tracks the previous
 * *node* leaves a just-deselected edge's brightened colour cached with nothing left to repaint
 * it away — which is exactly what selecting a second edge without first deselecting the first
 * used to do. Null wherever there was no previous edge, or it is no longer drawn at all (a
 * scope change dropped it), neither of which needs clearing.
 */
function edgeOwner(edgeKey: string | null): string | null {
  const drawn = showing()?.getGraph();
  return edgeKey && drawn?.hasEdge(edgeKey) ? drawn.source(edgeKey) : null;
}

/**
 * revealClaim selects a claim named somewhere other than the canvas — a row in a pane — and brings
 * it into view. Selecting alone would repaint nothing and leave the reader looking at a picture
 * that has silently changed what it is about.
 */
export function revealClaim(id: string): void {
  const previousNode = useExplorer.getState().selection.selected;
  const previousEdgeOwner = edgeOwner(useExplorer.getState().selection.selectedEdge);
  useExplorer.getState().select(id);
  highlight([
    id,
    ...(previousNode ? [previousNode] : []),
    ...(previousEdgeOwner ? [previousEdgeOwner] : []),
    ...labelBearingNodes(),
  ]);
  panIntoView(id);
}

/**
 * walkHistory steps along the reader's own trail — back or forward — and shows where they land.
 * Walking it adds nothing to it, which is what makes forward mean anything.
 */
export function walkHistory(delta: -1 | 1): void {
  const beforeNode = useExplorer.getState().selection.selected;
  const beforeEdgeOwner = edgeOwner(useExplorer.getState().selection.selectedEdge);
  useExplorer.getState().stepHistory(delta);
  const now = useExplorer.getState().selection.selected;
  // A history step can land on an edge visit as easily as a node one, so the edge it lands on
  // needs its own repaint too — not just the edge it steps away from.
  const nowEdgeOwner = edgeOwner(useExplorer.getState().selection.selectedEdge);
  highlight([
    ...[now, beforeNode, beforeEdgeOwner, nowEdgeOwner].filter((id): id is string => id !== null),
    ...labelBearingNodes(),
  ]);
  if (now) panIntoView(now);
}

/** bindEvents wires pointer interaction into core state, hover apart from selection. */
function bindEvents(instance: Sigma): void {
  const store = useExplorer.getState();

  // Every instrument ends at the camera, so holding it here is what makes the bound one bound
  // rather than a limit per tool. Sigma reports a resize the same way, which is the case where
  // nothing moved but the viewport the bound is measured against.
  instance.getCamera().on('updated', holdCamera);
  instance.on('resize', () => {
    applyBound();
    holdCamera();
  });

  instance.on('clickNode', ({ node }) => {
    const previousNode = useExplorer.getState().selection.selected;
    const previousEdgeOwner = edgeOwner(useExplorer.getState().selection.selectedEdge);
    useExplorer.getState().select(node);
    // Clicking a claim is asking about it, so the pane that answers comes forward.
    useExplorer.getState().setSidePane('info');
    highlight([
      node,
      ...(previousNode ? [previousNode] : []),
      ...(previousEdgeOwner ? [previousEdgeOwner] : []),
      ...labelBearingNodes(),
    ]);
  });

  instance.on('clickEdge', ({ edge }) => {
    const previousNode = useExplorer.getState().selection.selected;
    const previousEdgeOwner = edgeOwner(useExplorer.getState().selection.selectedEdge);
    const drawn = instance.getGraph();
    useExplorer.getState().selectEdge(edge);
    useExplorer.getState().setSidePane('info');
    // The edge's own claim is the one that owns it, so repainting from there covers both — and
    // the previously selected edge's owner must repaint too, or its highlight survives with
    // nothing left to clear it (-> edgeOwner).
    highlight([
      drawn.source(edge),
      ...(previousNode ? [previousNode] : []),
      ...(previousEdgeOwner ? [previousEdgeOwner] : []),
    ]);
  });

  instance.on('clickStage', () => {
    const previousNode = useExplorer.getState().selection.selected;
    const previousEdgeOwner = edgeOwner(useExplorer.getState().selection.selectedEdge);
    useExplorer.getState().select(null);
    // Only a previously-selected claim or edge leaves anything to clear — clicking empty
    // canvas with nothing selected repaints nothing.
    const toClear = [...(previousNode ? [previousNode] : []), ...(previousEdgeOwner ? [previousEdgeOwner] : [])];
    if (toClear.length > 0) highlight([...toClear, ...labelBearingNodes()]);
  });

  // Hover drives only a partial repaint: the highlight is immediate (two nodes), the
  // preview debounced so a sweeping pointer cannot outrun the reader.
  instance.on('enterNode', ({ node, event }) => {
    const previous = useExplorer.getState().selection.hovered;
    useExplorer.getState().hover(node);
    highlight([node, ...(previous ? [previous] : [])]);
    window.clearTimeout(hoverTimer);
    // event.x/y are already relative to the canvas host, which is what the floating
    // tooltip is positioned against — no further conversion needed.
    const { x, y } = event;
    hoverTimer = window.setTimeout(() => {
      if (useExplorer.getState().selection.hovered !== node) return;
      const g = graph();
      if (!g.hasNode(node)) return;
      useExplorer.getState().setPreview({
        id: node,
        claimType: String(g.getNodeAttribute(node, 'claimType') ?? ''),
        content: contentSnippet(node),
        x,
        y,
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


/**
 * onPointer reports the pointer's canvas position, and null when it leaves. Taken from
 * Sigma's own captor rather than a DOM listener, so it agrees with what the renderer thinks
 * the pointer is doing.
 */
export function onPointer(fn: (at: { x: number; y: number } | null) => void): () => void {
  const instance = unionOf();
  if (!instance) return () => {};
  const captor = instance.getMouseCaptor();
  const move = (e: { x: number; y: number }) => fn({ x: e.x, y: e.y });
  const leave = () => fn(null);
  captor.on('mousemove', move);
  captor.on('mouseleave', leave);
  return () => {
    captor.off('mousemove', move);
    captor.off('mouseleave', leave);
  };
}

// --- the lens ---------------------------------------------------------------
//
// Stretching x rewrites every node, so it is affordable only over a window. Zooming in
// therefore shows a second Sigma bound to a small graph, and coming back shows the first one
// again — its buffers were never invalidated, since a lens only reads.
//
// Both instances are alive throughout. Swapping is a change of which canvas is visible, so
// neither direction pays an upload and the switch has nothing to hide.

/**
 * showLens draws this graph in place of the union. The lens is built and uploaded before the
 * swap, so no frame is ever shown empty, and it is handed the union's camera so the picture
 * does not move.
 */
export function showLens(lensGraph: DirectedGraph): void {
  const union = unionOf();
  const host = hostOf();
  if (!union || !host) return;
  const camera = union.getCamera().getState();

  let lens = lensOf();
  if (lens) lens.setGraph(lensGraph);
  else {
    lens = new Sigma(lensGraph, host, sigmaSettings());
    setLens(lens);
    bindEvents(lens);
  }
  // The lens must normalise against the same extent, or swapping would change the scale.
  lens.setCustomBBox(union.getCustomBBox());
  lens.getCamera().setState(camera);
  applyViewSettings(activeView(useExplorer.getState()), lens);
  // Uploaded first, shown second.
  lens.refresh();
  host.style.visibility = 'visible';
  union.getContainer().style.visibility = 'hidden';
  // A new lens is built from the shared settings, which carry no ceiling of their own, so the
  // instance the reader is about to drive is given one as it comes forward.
  applyBound();
}

/**
 * hideLens shows the union again. Nothing is rebuilt: it has been resident and current all
 * along, so this is a change of visibility and a camera handed back.
 */
export function hideLens(): void {
  const union = unionOf();
  const host = hostOf();
  const lens = lensOf();
  if (!union || !host || host.style.visibility === 'hidden') return;
  if (lens) union.getCamera().setState(lens.getCamera().getState());
  host.style.visibility = 'hidden';
  union.getContainer().style.visibility = 'visible';
}

/** mountLens gives the lens its own canvas, hidden until it is wanted. */
export function mountLens(container: HTMLElement): void {
  setHost(container);
  container.style.visibility = 'hidden';
}

/**
 * onZoom sees the wheel before Sigma's captor does, and returning false hands it back.
 *
 * The plain wheel is time: a timeline is read along its axis, and the gesture a reader reaches
 * for first should act on the axis they came for. The strata are the occasional adjustment, so
 * they sit behind the modifier.
 */
export interface ZoomGesture {
  factor: number;
  viewportX: number;
  viewportY: number;
  /** Shift zooms the strata; without it the wheel zooms time. */
  shift: boolean;
}

export function onZoom(fn: (gesture: ZoomGesture) => boolean): () => void {
  const container = unionOf()?.getContainer();
  if (!container) return () => {};
  const wheel = (event: WheelEvent) => {
    // A notch up magnifies, a notch down shrinks; the exponent keeps it smooth on trackpads.
    const factor = Math.pow(1.0015, -event.deltaY);
    const bounds = (event.currentTarget as HTMLElement).getBoundingClientRect();
    const handled = fn({
      factor,
      viewportX: event.clientX - bounds.left,
      viewportY: event.clientY - bounds.top,
      shift: event.shiftKey,
    });
    // Unhandled means the camera keeps it, which is the classic zoom every other tool has.
    if (!handled) return;
    event.preventDefault();
    event.stopPropagation();
  };
  // Capture, so it is seen before Sigma's own captor.
  container.addEventListener('wheel', wheel, { capture: true, passive: false });
  hostOf()?.addEventListener('wheel', wheel, { capture: true, passive: false });
  return () => {
    container.removeEventListener('wheel', wheel, { capture: true });
    hostOf()?.removeEventListener('wheel', wheel, { capture: true });
  };
}

/**
 * pinExtent freezes what the renderer normalises against.
 *
 * Sigma fits the graph's own bounding box to the viewport on every refresh, so widening x in
 * graph space is scaled straight back out — a stretch would be invisible, and squashing y is
 * what a reader would see instead. Pinning the extent to the *unstretched* layout is what lets
 * a stretch mean something: nodes then run past the box and off the screen, which is what
 * being zoomed in is.
 */
export function pinExtent(x0: number, x1: number, y0: number, y1: number): void {
  const bbox = { x: [x0, x1] as [number, number], y: [y0, y1] as [number, number] };
  unionOf()?.setCustomBBox(bbox);
  lensOf()?.setCustomBBox(bbox);
  // Every measurement of the picture is read off the projection, and a new box is a new
  // projection. The union is re-uploaded by the load that pinned it; a lens the reader happens to
  // be looking at is not, so it is drawn again here rather than answering from the old box.
  if (lensShowing()) lensOf()?.refresh();
  pin({ x0, x1, y0, y1 });
  applyBound();
}

/** unpinExtent hands normalisation back to the graph, for the layouts that want fitting. */
export function unpinExtent(): void {
  unionOf()?.setCustomBBox(null);
  lensOf()?.setCustomBBox(null);
  pin(null);
  applyBound();
}

/** onRender calls fn after each frame and returns the unsubscribe. */
export function onRender(fn: () => void): () => void {
  const instance = unionOf();
  if (!instance) return () => {};
  instance.on('afterRender', fn);
  return () => instance.off('afterRender', fn);
}

/**
 * onBox reports a shift-drag as it happens, and null when it ends or is abandoned. Registered
 * in the capture phase and stopped there, so Sigma's captor never sees the drag and does not
 * pan the camera underneath it.
 */
export function onBox(fn: (box: Box | null) => void): () => void {
  const hosts = [unionOf()?.getContainer(), hostOf()].filter((h): h is HTMLElement => !!h);
  if (hosts.length === 0) return () => {};

  let start: { x: number; y: number } | null = null;

  const at = (event: MouseEvent, host: HTMLElement) => {
    const bounds = host.getBoundingClientRect();
    return { x: event.clientX - bounds.left, y: event.clientY - bounds.top };
  };

  const down = (event: MouseEvent) => {
    if (!event.shiftKey || event.button !== 0) return;
    start = at(event, event.currentTarget as HTMLElement);
    event.preventDefault();
    event.stopPropagation();
  };
  const move = (event: MouseEvent) => {
    if (!start) return;
    const now = at(event, event.currentTarget as HTMLElement);
    fn({ x0: start.x, y0: start.y, x1: now.x, y1: now.y });
    event.preventDefault();
    event.stopPropagation();
  };
  const up = (event: MouseEvent) => {
    if (!start) return;
    const now = at(event, event.currentTarget as HTMLElement);
    const box = { x0: start.x, y0: start.y, x1: now.x, y1: now.y };
    start = null;
    fn(null);
    // A box too small to have been meant is a click, not a zoom.
    if (Math.abs(box.x1 - box.x0) > 6 && Math.abs(box.y1 - box.y0) > 6) zoomToBox(box);
    event.preventDefault();
    event.stopPropagation();
  };

  for (const host of hosts) {
    host.addEventListener('mousedown', down, true);
    host.addEventListener('mousemove', move, true);
    host.addEventListener('mouseup', up, true);
    host.addEventListener('mouseleave', up, true);
  }
  return () => {
    for (const host of hosts) {
      host.removeEventListener('mousedown', down, true);
      host.removeEventListener('mousemove', move, true);
      host.removeEventListener('mouseup', up, true);
      host.removeEventListener('mouseleave', up, true);
    }
  };
}

/** unmount tears the renderer down — only on page teardown, never on re-render. */
export function unmount(): void {
  cancelAnimationFrame(fpsHandle);
  lensOf()?.kill();
  setLens(null);
  unionOf()?.kill();
  setUnion(null);
}
