/**
 * Session actions — everything the user can initiate, expressed without a UI.
 *
 * The UI calls these and renders the resulting state; it never manipulates the
 * graph, the layout or the store directly. That is the whole of the separation:
 * a component may read state and dispatch an action, and nothing else.
 */

import { activeConnection, useConnections } from './connections.ts';
import { useQuery } from './query.ts';
import { sourceFor } from './data/source.ts';
import { mergeClaimsProgressively, graph, totalContributions } from './graph/universe.ts';
import { yieldToPaint } from './scheduler.ts';
import { contributionOf, depths, historyStats } from './graph/shape.ts';
import { degreeStats, sizeByDegree } from './graph/build.ts';
import { apply } from './layout/layouts.ts';
import type { LayoutName } from './layout/layouts.ts';
import { defaultView, useExplorer } from './store.ts';
import type { ViewState } from './store.ts';

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
 * load reads from the active source, merges the result into the union and lays the
 * union out. It does not know whether the claims came from a generator or a server —
 * that is the point of the data-source port.
 *
 * Merging rather than replacing is the accumulation the design calls for: a claim
 * reached twice resolves to one node, so the union grows by union and not by sum.
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
