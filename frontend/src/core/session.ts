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
import { yieldToPaint } from './scheduler.ts';
import { contributionOf, depths, historyStats } from './graph/shape.ts';
import { degreeStats, sizeByDegree } from './graph/build.ts';
import { apply } from './layout/layouts.ts';
import type { LayoutName } from './layout/layouts.ts';
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
}

let viewCounter = 0;

/** log appends a line to the store's log, which the log pane renders. */
function log(line: string): void {
  useExplorer.getState().appendLog(line);
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
 * load reads from the active source, merges into the union and lays it out, indifferent
 * to whether a generator or a server answered. Merging is the accumulation the design
 * calls for: a claim reached twice is one node, so the union grows by union, not sum.
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

  let page;
  try {
    page = await source.fetch({ limit: req.limit ?? useQuery.getState().query.limit });
  } catch (err) {
    log(`load        failed — ${String(err)}`);
    store.patchStatus({ busy: null, progress: null });
    return;
  }
  log(
    `${source.kind === 'mock' ? 'generate' : 'read    '}    ${page.elapsedMs.toFixed(0)} ms · ` +
      `${page.claims.length.toLocaleString('en-US')} claims, ${page.contributions} contributions ` +
      `(${page.origin})`,
  );

  store.patchStatus({ busy: 'merging claims', progress: 0 });
  await yieldToPaint();
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

  const g = graph();
  useExplorer.getState().patchStatus({ busy: 'measuring shape', progress: null });
  await yieldToPaint();
  sizeByDegree(g);
  const shape = shapeOf();
  log(
    `shape       by depth: height ${shape.depthStats.height}, widest ${shape.depthStats.widestLayer} · ` +
      `by history: ${shape.history.rows} rows, widest ${shape.history.widestRow}`,
  );

  useExplorer.getState().patchStatus({ busy: 'laying out', progress: null });
  await yieldToPaint();
  const layoutMs = await apply(g, view.layout, {
    depth: shape.depth,
    contribution: contributionOf(g),
  });
  log(`layout      ${layoutMs.toFixed(0)} ms · ${view.layout}`);

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
 * discoverScopes asks the active source what scopes the archive holds. Until it answers
 * the explorer knows no branch name, so nothing scope-confined can be read.
 *
 * One branch is selected outright — there is no choice to make. Several are left unset:
 * picking by name, position or convention would be the guess this replaced.
 */
export async function discoverScopes(): Promise<void> {
  const store = useExplorer.getState();
  const connection = activeConnection();
  if (!connection) {
    store.setScopes({ state: 'error', scopes: [], error: 'no source configured' });
    return;
  }

  store.setScopes({ state: 'loading', scopes: [], error: null });
  const source = sourceFor(connection, useConnections.getState().secretOf(connection.id));
  let scopes: Scope[];
  try {
    scopes = await source.branches();
  } catch (err) {
    log(`branches    failed — ${String(err)}`);
    useExplorer.getState().setScopes({ state: 'error', scopes: [], error: String(err) });
    return;
  }

  useExplorer.getState().setScopes({ state: 'ready', scopes, error: null });
  const named = scopes.filter((s) => s.name !== ARCHIVE_SCOPE);
  log(
    `branches    ${named.length} branch(es)${scopes.length > named.length ? ' + the archive' : ''}` +
      `${named.length > 0 ? ` · ${named.map((s) => s.name).join(', ')}` : ''}`,
  );
  if (named.length === 1) await selectScope(named[0]);
}

/**
 * selectScope confines the active view to a scope, or lifts the confinement with null.
 *
 * The membership question goes to the source: a scoped query with `output.detail: id`
 * returns the identities in that scope, and the view then draws the intersection with what
 * is cached. So the engine decides what a branch contains, and this decides what to show.
 *
 * Ids the answer names but the cache lacks are counted rather than hidden: they are claims
 * the server has and this session has not read, which is the honest reading of a graph
 * loaded once against an archive that moves.
 */
export async function selectScope(scope: Scope | null): Promise<void> {
  const store = useExplorer.getState();
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
    return;
  }

  store.patchStatus({ busy: `scoping to ${scopeLabel(scope)}`, progress: null });
  await yieldToPaint();

  const source = sourceFor(connection, useConnections.getState().secretOf(connection.id));
  const t0 = performance.now();
  let ids: string[];
  try {
    ids = await source.scopeIds(scope);
  } catch (err) {
    log(`scope       ${scopeLabel(scope)} failed — ${String(err)}`);
    useExplorer.getState().patchStatus({ busy: null, progress: null });
    return;
  }

  const members = setMembers(scope.name, ids);
  const g = graph();
  let missing = 0;
  for (const id of members) if (!g.hasNode(id)) missing++;
  log(
    `scope       ${scopeLabel(scope)} · ${members.size.toLocaleString('en-US')} claims ` +
      `(${(performance.now() - t0).toFixed(0)} ms)` +
      `${missing > 0 ? ` · ${missing.toLocaleString('en-US')} not cached yet` : ''}`,
  );

  onLoaded?.();
  useExplorer.getState().patchStatus({ busy: null, progress: null });
}

/**
 * inScope is the membership test the renderer applies per node: a lookup in the id set the
 * source returned. A scope never asked admits everything, so a view is never blank for
 * want of an answer.
 */
export function inScope(scope: Scope, node: string): boolean {
  const members = membersOf(scope.name);
  return members === null || members.has(node);
}

/** scopeShortfall counts claims a scope names that this session has not cached. */
export function scopeShortfall(scope: Scope | null): number {
  if (!scope) return 0;
  const members = membersOf(scope.name);
  if (!members) return 0;
  const g = graph();
  let missing = 0;
  for (const id of members) if (!g.hasNode(id)) missing++;
  return missing;
}

/**
 * onLoaded is set by the renderer: core must not import it, but the graph has to be
 * handed to Sigma once the union changes, and the last stage of a load is drawing it.
 */
let onLoaded: (() => void) | null = null;

/** setOnLoaded lets the render layer register its refresh without core importing it. */
export function setOnLoaded(fn: () => void): void {
  onLoaded = fn;
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
  const ms = await apply(g, layout, { depth, contribution: contributionOf(g) });
  log(`layout      ${ms.toFixed(0)} ms · ${layout}`);
  onLoaded?.();
  useExplorer.getState().patchStatus({ busy: null, progress: null });
}

/** claimDetail gathers what the selection pane shows for one claim. */
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
