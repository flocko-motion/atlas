/**
 * package: core / session
 * type:    logic
 * job:     the actions a user can initiate, expressed without a UI
 * limits:  headless; the UI only dispatches them (-> ui)
 *
 * The UI calls these and renders the resulting state, never touching the graph, the
 * layout or the store directly: a component may read state and dispatch, nothing else.
 */

import { activeConnection, useConnections } from './connections.ts';
import { useQuery } from './query.ts';
import { sourceFor } from './data/source.ts';
import { mergeClaimsProgressively, graph, totalContributions } from './graph/universe.ts';
import { membersOf, setMembers } from './graph/members.ts';
import { contentOf, rememberContent } from './content.ts';
import { covers, lensOf, windowAround } from './graph/lens.ts';
import type { Window } from './graph/lens.ts';
import type { DirectedGraph } from 'graphology';
import { yieldToPaint } from './scheduler.ts';
import { contributionOf, depths, historyStats } from './graph/shape.ts';
import { degreeStats, sizeByDegree } from './graph/build.ts';
import { TIMELINE_HEIGHT, apply, assignTimeline } from './layout/layouts.ts';
import type { LayoutName, TimelineContext } from './layout/layouts.ts';
import { timeScale } from './layout/timescale.ts';
import type { TimeScale } from './layout/timescale.ts';
import { defaultView, useExplorer } from './store.ts';
import type { ViewState } from './store.ts';
import { ARCHIVE_SCOPE, scopeLabel } from './scope.ts';
import type { Scope } from './scope.ts';

export interface LoadRequest {
  /** Cap on claims read — `limit.results` in the query contract. */
  limit?: number;
  layout?: LayoutName;
  /** Open the result as a new view rather than folding into the active one. */
  asNewView?: boolean;
  /** Scope to read, when the caller knows it — selection passes the one it just chose. */
  scope?: Scope | null;
}

let viewCounter = 0;

/** kib renders a byte count for a status line, where the exact figure says nothing. */
function kib(bytes: number): string {
  return `${Math.round(bytes / 1024).toLocaleString('en-US')} KiB`;
}

/** log appends a line to the store's log, which the log pane renders. */
function log(line: string): void {
  useExplorer.getState().appendLog(line);
}

/**
 * The axis the layout and the ruler share, so the ruler cannot disagree with the picture.
 */
let axis: TimeScale | null = null;

/**
 * The extent the unstretched timeline occupies. Held so the renderer can normalise against it
 * rather than against the graph, whose x grows as time is stretched.
 */
let restExtent: { x0: number; x1: number; y0: number; y1: number } | null = null;

/** timelineExtent is the unstretched extent, or null before a timeline is laid out. */
export function timelineExtent() {
  return restExtent;
}

/** timeAxis is the current time axis, or null before anything is laid out on one. */
export function timeAxis(): TimeScale | null {
  return axis;
}

/** timelineContext reads the instants and classes off the graph, and builds the axis. */
function timelineContext(visible?: readonly string[], stretch = 1): TimelineContext {
  const g = graph();
  const createdAt = (node: string) => Number(g.getNodeAttribute(node, 'createdAt') ?? 0);
  const instants: number[] = [];
  g.forEachNode((node) => instants.push(createdAt(node)));
  axis = timeScale(instants);
  restExtent = { x0: 0, x1: Math.max(axis.width, 1), y0: 0, y1: TIMELINE_HEIGHT };
  return {
    toX: (at) => (axis as TimeScale).toX(at) * stretch,
    createdAt,
    classOf: (node) => String(g.getNodeAttribute(node, 'cls') ?? ''),
    visible,
  };
}

/** shapeOf recomputes the union's shape statistics for the status bar and panes. */
export function shapeOf() {
  const g = graph();
  const { depth, stats } = depths(g);
  return {
    depth,
    depthStats: stats,
    history: historyStats(g, contributionOf(g)),
    degree: degreeStats(g),
    order: g.order,
    size: g.size,
    contributions: totalContributions(),
  };
}

/**
 * load reads from the active source and merges into the union, indifferent to who answered.
 * A claim reached twice is one node, so the union grows by union rather than sum.
 */
export async function load(req: LoadRequest = {}): Promise<void> {
  const store = useExplorer.getState();
  const connection = activeConnection();
  if (!connection) {
    log('load        no source configured — add one under Server');
    return;
  }

  const source = sourceFor(connection, useConnections.getState().secretOf(connection.id));

  // The view (and therefore its tab) exists before any work starts, so the app shows
  // something the instant the button is pressed rather than after the load.
  const view = ensureView(req.layout, req.asNewView);
  const { classes } = useQuery.getState().query;
  store.patchView(view.id, { classes });

  store.patchStatus({
    busy: source.kind === 'mock' ? 'generating claims' : 'reading claims',
    progress: null,
  });
  await yieldToPaint();

  // The scope the picker selected is what a server read is generated from; a generator
  // ignores it. The view's own scope keeps the read and the view in agreement, and a caller
  // that has just chosen one passes it rather than waiting for the patch to land.
  const scope =
    req.scope ?? useExplorer.getState().views.find((v) => v.id === view.id)?.scope ?? null;

  const expected = scope ? (membersOf(scope.name)?.size ?? 0) : 0;
  let page;
  try {
    page = await source.fetch({
      limit: req.limit ?? useQuery.getState().query.limit,
      scope,
      onProgress: (read, bytesRead) => {
        // How many claims are still to come is only known where membership was asked for
        // first. Without that there is no total to be a fraction of — a streamed body declares
        // no length — so the bar runs indeterminate and the bytes read say it is moving.
        useExplorer.getState().patchStatus({
          busy: expected > 0
            ? `reading claims · ${read.toLocaleString('en-US')} of ${expected.toLocaleString('en-US')}`
            : `reading claims · ${read.toLocaleString('en-US')} · ${kib(bytesRead)}`,
          progress: expected > 0 ? Math.min(1, read / expected) : null,
        });
      },
    });
  } catch (err) {
    log(`load        failed — ${String(err)}`);
    store.setNotice({
      level: 'error',
      text: 'The read failed.',
      hint: String(err instanceof Error ? err.message : err),
    });
    store.patchStatus({ busy: null, progress: null });
    return;
  }

  if (page.claims.length === 0) {
    store.setNotice({
      level: 'info',
      text: 'The archive answered with no claims.',
      hint: 'Nothing has been contributed to it yet, or the query reached nothing.',
    });
  } else {
    store.setNotice(null);
  }
  log(
    `${source.kind === 'mock' ? 'generate' : 'read    '}    ${page.elapsedMs.toFixed(0)} ms · ` +
      `${page.claims.length.toLocaleString('en-US')} claims, ${page.contributions} contributions ` +
      `(${page.origin})`,
  );

  store.patchStatus({ busy: 'merging claims', progress: 0 });
  await yieldToPaint();
  dropLens();
  const merged = await mergeClaimsProgressively(page.claims, page.contributions, (report) => {
    useExplorer.getState().patchStatus({
      busy: report.stage,
      progress: report.total > 0 ? report.done / report.total : null,
    });
  });
  log(
    `merge       ${merged.mergeMs.toFixed(0)} ms · +${merged.addedNodes} nodes, ` +
      `+${merged.addedEdges} edges, ${merged.duplicateClaims} already present`,
  );

  // The claims that came back *are* the scope's membership, so a read establishes it where it
  // was not already known. Asking separately would be a second walk of the same closure.
  if (scope && !membersOf(scope.name)) {
    setMembers(scope.name, page.claims.map((drawn) => drawn.claim.id));
  }

  const g = graph();
  useExplorer.getState().patchStatus({ busy: 'measuring shape', progress: null });
  await yieldToPaint();
  sizeByDegree(g);
  const shape = shapeOf();
  log(
    `shape       by depth: height ${shape.depthStats.height}, widest ${shape.depthStats.widestLayer} · ` +
      `by history: ${shape.history.rows} rows, widest ${shape.history.widestRow}`,
  );

  // The history layout needs a contribution per claim, which a server read does not carry.
  // Falling back to depth draws the same graph by the axis that is known, rather than
  // piling every claim into one row and looking like a bug.
  let layout = view.layout;
  if (layout === 'history' && shape.history.rows <= 1 && g.order > 1) {
    layout = 'layered';
    store.patchView(view.id, { layout });
    log('layout      history needs a contribution index, which this source does not carry — using layered');
  }

  useExplorer.getState().patchStatus({ busy: 'laying out', progress: null });
  await yieldToPaint();
  const layoutMs = await apply(g, layout, {
    depth: shape.depth,
    contribution: contributionOf(g),
    timeline: layout === 'timeline' ? timelineContext(view.classes, view.xStretch) : undefined,
  });
  log(`layout      ${layoutMs.toFixed(0)} ms · ${layout}`);

  useExplorer.getState().patchStatus({
    busy: 'drawing',
    progress: null,
    nodes: g.order,
    edges: g.size,
    contributions: shape.contributions,
  });
  await yieldToPaint();
  onLoaded?.();
  useExplorer.getState().patchStatus({ busy: null, progress: null });
}

/**
 * discoverScopes asks the source what the archive holds; until it answers, no branch name is
 * known. A sole branch is selected outright — picking among several would be the guess this
 * replaced.
 */
export async function discoverScopes(): Promise<void> {
  const store = useExplorer.getState();
  const connection = activeConnection();
  if (!connection) {
    store.setScopes({ state: 'error', scopes: [], selected: null, error: 'no source configured' });
    return;
  }

  store.setScopes({ state: 'loading', scopes: [], selected: store.scopes.selected, error: null });
  const source = sourceFor(connection, useConnections.getState().secretOf(connection.id));
  let scopes: Scope[];
  try {
    scopes = await source.branches();
  } catch (err) {
    log(`branches    failed — ${String(err)}`);
    const text = err instanceof Error ? err.message : String(err);
    useExplorer.getState().setScopes({ state: 'error', scopes: [], selected: null, error: text });
    useExplorer.getState().setNotice({ level: 'error', text: 'Could not list the branches.', hint: text });
    return;
  }

  // A selection the new listing does not hold belongs to another archive, so it is dropped
  // rather than carried across a change of source.
  const held = useExplorer.getState().scopes.selected;
  const selected = held && scopes.some((s) => s.name === held.name && s.head === held.head) ? held : null;
  useExplorer.getState().setScopes({ state: 'ready', scopes, selected, error: null });
  const named = scopes.filter((s) => s.name !== ARCHIVE_SCOPE);
  log(
    `branches    ${named.length} branch(es)${scopes.length > named.length ? ' + the archive' : ''}` +
      `${named.length > 0 ? ` · ${named.map((s) => s.name).join(', ')}` : ''}`,
  );
  if (named.length === 1) await selectScope(named[0]);
}

/**
 * selectScope confines the active view to a scope, or lifts it with null. Membership is the
 * source's answer (`output.detail: id`) and the view draws its intersection with the cache;
 * ids the cache lacks are counted rather than hidden.
 */
export async function selectScope(scope: Scope | null): Promise<void> {
  const store = useExplorer.getState();
  store.setScopes({ ...store.scopes, selected: scope });
  const active = store.views.find((v) => v.id === store.activeViewId);
  if (active) store.patchView(active.id, { scope });
  useQuery.getState().patchQuery({ branch: scope ? scope.name : null });

  if (!scope) {
    log('scope       everything loaded');
    onLoaded?.();
    return;
  }

  const connection = activeConnection();
  if (!connection) {
    log('scope       no source to ask which claims are in it');
    store.setNotice({
      level: 'error',
      text: 'No source to ask which claims this scope contains.',
      hint: 'Add or activate one under Server.',
    });
    return;
  }

  // Membership and content are different questions, and asking them separately is right —
  // except on the first read of a scope, where nothing is cached and the claims answer both.
  // Asking first would then walk the closure twice for one answer.
  if (!membersOf(scope.name) && graph().order === 0) {
    await load({ scope });
    onLoaded?.();
    return;
  }

  store.patchStatus({ busy: `asking what ${scopeLabel(scope)} holds`, progress: null });
  await yieldToPaint();

  const source = sourceFor(connection, useConnections.getState().secretOf(connection.id));
  const t0 = performance.now();
  let ids: string[];
  try {
    ids = await source.scopeIds(scope);
  } catch (err) {
    const text = err instanceof Error ? err.message : String(err);
    log(`scope       ${scopeLabel(scope)} failed — ${text}`);
    useExplorer.getState().setNotice({
      level: 'error',
      text: `Could not read what ${scopeLabel(scope)} contains.`,
      hint: text,
    });
    useExplorer.getState().patchStatus({ busy: null, progress: null });
    return;
  }

  const members = setMembers(scope.name, ids);
  log(
    `scope       ${scopeLabel(scope)} · ${members.size.toLocaleString('en-US')} claims ` +
      `(${(performance.now() - t0).toFixed(0)} ms)`,
  );

  // Selecting a scope shows it. A read is what makes that possible, so it is the default
  // action rather than a second step the user is told to take — and the scope's own size is
  // the limit, since "this branch" means all of it.
  if (missingFrom(members) > 0) {
    await load({ limit: members.size, scope });
  }

  const missing = missingFrom(members);
  if (members.size === 0) {
    useExplorer.getState().setNotice({
      level: 'info',
      text: `${scopeLabel(scope)} contains no claims.`,
      hint: 'The scope exists and is empty — nothing has been contributed to it.',
    });
  } else if (missing > 0) {
    useExplorer.getState().setNotice({
      level: 'info',
      text: `${missing.toLocaleString('en-US')} of ${scopeLabel(scope)}'s claims could not be read.`,
      hint: `It contains ${members.size.toLocaleString('en-US')}; the rest are drawn.`,
    });
  }

  onLoaded?.();
  useExplorer.getState().patchStatus({ busy: null, progress: null });
}

/**
 * inScope is the renderer's per-node test: a lookup in the source's id set. An unasked scope
 * admits everything, so no view is blank for want of an answer.
 */
export function inScope(scope: Scope, node: string): boolean {
  const members = membersOf(scope.name);
  return members === null || members.has(node);
}

/** missingFrom counts which of these ids the union does not hold. */
function missingFrom(ids: Set<string>): number {
  const g = graph();
  let missing = 0;
  for (const id of ids) if (!g.hasNode(id)) missing++;
  return missing;
}

/**
 * scopeCounts reports what a scope holds against what this session has read. Null where the
 * scope has not been asked about, which is a different thing from holding nothing.
 */
export function scopeCounts(scope: Scope | null): { contains: number; loaded: number } | null {
  if (!scope) return null;
  const members = membersOf(scope.name);
  if (!members) return null;
  const missing = missingFrom(members);
  return { contains: members.size, loaded: members.size - missing };
}

/** onLoaded is the renderer's hook, so core hands over a finished graph without importing it. */
let onLoaded: (() => void) | null = null;

/** setOnLoaded lets the render layer register its refresh without core importing it. */
export function setOnLoaded(fn: () => void): void {
  onLoaded = fn;
}

/**
 * showAll returns to the whole archive: time back to its own scale, and the camera framing the
 * extent that scale occupies. The way back from anywhere, which is what makes zooming freely
 * comfortable.
 */
export function showAll(): void {
  const store = useExplorer.getState();
  const active = store.views.find((v) => v.id === store.activeViewId);
  if (active && active.xStretch !== 1) stretchX(1 / active.xStretch);
  onLoaded?.();
  onShowAll?.();
}

/** The edges of the archive in time, for jumping to either end. */
export function timeEnds(): { from: number; to: number } | null {
  return axis ? { from: axis.from, to: axis.to } : null;
}

/** instantAtX reads the instant under a drawn x, undoing the stretch the layout applied. */
export function instantAtX(graphX: number): number | null {
  const store = useExplorer.getState();
  const active = store.views.find((v) => v.id === store.activeViewId);
  if (!axis || !active) return null;
  return axis.atX(graphX / active.xStretch);
}

/** stepFrom is the instant one claim along from here, in either direction. */
export function stepFrom(at: number, direction: 1 | -1): number | null {
  return axis ? axis.stepInstant(at, direction) : null;
}

/** axisWidth is the unstretched width of the time axis, which bounds how far it may compress. */
export function axisWidth(): number | null {
  return axis ? axis.width : null;
}

/** axisXOf is where an instant sits on the drawn axis, stretch included. */
export function axisXOf(at: number): number | null {
  const store = useExplorer.getState();
  const active = store.views.find((v) => v.id === store.activeViewId);
  if (!axis || !active) return null;
  return axis.toX(at) * active.xStretch;
}

/**
 * onShowAll is the renderer's camera reset. Core decides *that* the view should be reframed and
 * leaves *how* to the layer that owns the camera.
 */
let onShowAll: (() => void) | null = null;

/** setOnShowAll lets the render layer register its reset without core importing it. */
export function setOnShowAll(fn: () => void): void {
  onShowAll = fn;
}

/** ensureView returns the view to render into, creating one when needed. */
function ensureView(layout?: LayoutName, asNewView?: boolean): ViewState {
  const store = useExplorer.getState();
  const active = store.views.find((v) => v.id === store.activeViewId);
  if (active && !asNewView) {
    if (layout && layout !== active.layout) store.patchView(active.id, { layout });
    return { ...active, layout: layout ?? active.layout };
  }
  const id = `view-${++viewCounter}`;
  const view = defaultView(id, `view ${viewCounter}`);
  // A view made after a scope was chosen inherits it, or its first read has no scope.
  view.scope = store.scopes.selected;
  if (layout) view.layout = layout;
  store.addView(view);
  return view;
}

/** relayout re-runs the active view's layout over the whole union. */
export async function relayout(layout: LayoutName): Promise<void> {
  const store = useExplorer.getState();
  const active = store.views.find((v) => v.id === store.activeViewId);
  if (!active) return;
  store.patchView(active.id, { layout });
  store.patchStatus({ busy: 'laying out', progress: null });
  await yieldToPaint();
  const g = graph();
  const { depth } = depths(g);
  const ms = await apply(g, layout, {
    depth,
    contribution: contributionOf(g),
    timeline: layout === 'timeline' ? timelineContext(active.classes, active.xStretch) : undefined,
  });
  log(`layout      ${ms.toFixed(0)} ms · ${layout}`);
  onLoaded?.();
  useExplorer.getState().patchStatus({ busy: null, progress: null });
}

/** The most content to fetch: beyond it the size is reported and the bytes stay put. */
export const CONTENT_LIMIT = 4096;

/**
 * fetchContent reads the selected claim's bytes, deciding from the declared size whether to
 * ask at all — so an oversized claim costs no request.
 */
export async function fetchContent(id: string): Promise<void> {
  const store = useExplorer.getState();
  const g = graph();
  if (!g.hasNode(id)) return;

  const size = Number(g.getNodeAttribute(id, 'contentSize') ?? 0);
  const encoding = g.getNodeAttribute(id, 'encoding') as string | undefined;
  if (!size) {
    store.setContent({ id, state: 'none', size: 0, encoding });
    return;
  }
  // A claim's id is the hash of its bytes, so once read they are right forever.
  const held = contentOf(id);
  if (held) {
    store.setContent({ id, state: 'ready', size, encoding, bytes: held });
    return;
  }
  if (size > CONTENT_LIMIT) {
    store.setContent({ id, state: 'too-large', size, encoding });
    return;
  }

  const connection = activeConnection();
  if (!connection) {
    store.setContent({ id, state: 'error', size, encoding, error: 'no source configured' });
    return;
  }
  store.setContent({ id, state: 'loading', size, encoding });

  const source = sourceFor(connection, useConnections.getState().secretOf(connection.id));
  const scope = useExplorer.getState().scopes.selected;
  try {
    const bytes = await source.content(scope, id);
    rememberContent(id, bytes);
    // A later selection may have overtaken this read; the answer to an old question is not
    // worth showing against a new one. The bytes are kept either way — they are correct
    // whenever they are next wanted.
    if (useExplorer.getState().selection.selected !== id) return;
    useExplorer.getState().setContent({ id, state: 'ready', size, encoding, bytes });
  } catch (err) {
    if (useExplorer.getState().selection.selected !== id) return;
    useExplorer.getState().setContent({
      id,
      state: 'error',
      size,
      encoding,
      error: err instanceof Error ? err.message : String(err),
    });
  }
}

/**
 * Below this many claims the union is small enough to stretch directly, and a lens would be
 * ceremony. Above it, zooming in shows a window instead.
 */
export const LENS_ABOVE = 5000;

/** The window a lens currently covers, so a small pan reuses it rather than cutting a new one. */
let lensWindow: Window | null = null;

/**
 * lensFor decides what to show for the x range in view: null shows the union again, costing
 * nothing since lensing never invalidated it.
 */
export function lensFor(x0: number, x1: number): { graph: DirectedGraph; inside: number } | null {
  const g = graph();
  if (g.order <= LENS_ABOVE) return null;
  if (lensWindow && covers(lensWindow, x0, x1)) return null; // the lens in place still holds

  const window = windowAround(x0, x1);
  const cut = lensOf(g, window);
  // A window holding nearly everything is not a lens; showing the union costs less and reads
  // the same.
  if (cut.inside >= g.order * 0.8) {
    lensWindow = null;
    return null;
  }
  lensWindow = window;
  log(
    `lens        ${cut.inside.toLocaleString('en-US')} claims · ` +
      `${cut.leaving.toLocaleString('en-US')} references leave the view (${cut.buildMs.toFixed(0)} ms)`,
  );
  return { graph: cut.graph, inside: cut.inside };
}

/** dropLens forgets the window, so the next view is cut fresh. */
export function dropLens(): void {
  lensWindow = null;
}

/**
 * The range time may be stretched over. It compresses as well as stretches — zooming classically
 * scales both axes and compressing time afterwards leaves the height alone, which is a vertical
 * zoom in all but name.
 *
 * How far it may compress is not a constant: the floor is wherever the drawn axis would become
 * narrower than the viewport, since past that the picture is stranded in empty space and the
 * camera's own zoom is the right instrument. The caller measures that and passes a limit, so this
 * bound is only the backstop for a caller that cannot measure.
 */
export const STRETCH_MIN = 1 / 4096;
export const STRETCH_MAX = 4096;

/**
 * stretchX scales the time axis and lays the drawn graph out again. Only x moves: the strata
 * already fill the height. It acts on whichever graph is showing, which is what makes a step
 * over a lens cost a thirtieth of one over the union.
 */
export function stretchX(
  factor: number,
  target?: DirectedGraph,
  /** Least stretch the caller will allow, from what it can see of the canvas. */
  floor = STRETCH_MIN,
): { stretch: number; applied: number } {
  const store = useExplorer.getState();
  const active = store.views.find((v) => v.id === store.activeViewId);
  if (!active || active.layout !== 'timeline') return { stretch: 1, applied: 1 };

  const stretched = Math.min(STRETCH_MAX, Math.max(Math.max(STRETCH_MIN, floor), active.xStretch * factor));
  const applied = stretched / active.xStretch;
  if (applied === 1) return { stretch: stretched, applied };
  store.patchView(active.id, { xStretch: stretched });

  const g = target ?? graph();
  const scale = axis;
  if (!scale) return { stretch: stretched, applied };
  assignTimeline(g, {
    toX: (at) => scale.toX(at) * stretched,
    createdAt: (node) => Number(g.getNodeAttribute(node, 'createdAt') ?? 0),
    classOf: (node) => String(g.getNodeAttribute(node, 'cls') ?? ''),
    visible: active.classes,
  });
  return { stretch: stretched, applied };
}

/**
 * edgeDetail gathers what the detail pane shows for one edge: its type, and the two claims it
 * joins. An edge belongs to the claim it points *from* — that claim created it — so the source
 * is where its provenance is read.
 */
export function edgeDetail(key: string) {
  const g = graph();
  if (!g.hasEdge(key)) return null;
  const from = g.source(key);
  const to = g.target(key);
  const label = (node: string) => String(g.getNodeAttribute(node, 'label') ?? '');
  const claimType = (node: string) => String(g.getNodeAttribute(node, 'claimType') ?? '');
  return {
    key,
    edgeType: String(g.getEdgeAttribute(key, 'claimType') ?? ''),
    from,
    fromLabel: label(from),
    fromType: claimType(from),
    to,
    toLabel: label(to),
    toType: claimType(to),
  };
}

/** claimDetail gathers what the detail pane shows for one claim. */
export function claimDetail(id: string) {
  const g = graph();
  if (!g.hasNode(id)) return null;
  const attrs = g.getNodeAttributes(id) as Record<string, unknown>;
  const references: { id: string; type: string }[] = [];
  g.forEachOutEdge(id, (_edge, edgeAttrs, _s, target) => {
    references.push({ id: target, type: String((edgeAttrs as { claimType?: string }).claimType ?? '') });
  });
  return {
    id,
    claimType: String(attrs.claimType ?? ''),
    contribution: Number(attrs.contribution ?? 0),
    createdAt: Number(attrs.createdAt ?? 0),
    contentSize: attrs.contentSize as number | undefined,
    encoding: attrs.encoding as string | undefined,
    label: String(attrs.label ?? ''),
    degree: g.degree(id),
    references,
    citedBy: g.inDegree(id),
  };
}
