/**
 * package: core / timeline
 * type:    logic
 * job:     the time axis a timeline is drawn on, and the stretch of each of its two axes
 * limits:  headless arithmetic and layout; who asks for a load is session's (-> core/session)
 *
 * The axis is held here rather than derived twice, so the ruler and the picture cannot disagree.
 * The stretches are the two zooms a reader has: each scales one axis of the layout and leaves the
 * other where it was, which the camera could not do — it takes both at once and magnifies the
 * claims along with the space between them.
 */

import type { DirectedGraph } from 'graphology';
import { graph } from './graph/universe.ts';
import { TIMELINE_HEIGHT, assignTimeline } from './layout/layouts.ts';
import type { TimelineContext } from './layout/layouts.ts';
import { timeScale } from './layout/timescale.ts';
import type { TimeScale } from './layout/timescale.ts';
import { useExplorer } from './store.ts';
import type { ViewState } from './store.ts';

/** How far each axis is stretched from the layout's own scale. */
export interface Stretch {
  x: number;
  y: number;
}

/** The axis the layout and the ruler share. */
let axis: TimeScale | null = null;

/**
 * The extent the unstretched timeline occupies, which the renderer normalises against — a stretch
 * is only visible if what measures it stands still.
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

/** stretchOf reads both stretches off a view, which is how every layout is asked for. */
export function stretchOf(view: ViewState): Stretch {
  return { x: view.xStretch, y: view.yStretch };
}

/** timelineContext reads the instants and classes off the graph, and builds the axis. */
export function timelineContext(
  visible?: readonly string[],
  stretch: Stretch = { x: 1, y: 1 },
): TimelineContext {
  const g = graph();
  const createdAt = (node: string) => Number(g.getNodeAttribute(node, 'createdAt') ?? 0);
  const instants: number[] = [];
  g.forEachNode((node) => instants.push(createdAt(node)));
  axis = timeScale(instants);
  restExtent = { x0: 0, x1: Math.max(axis.width, 1), y0: 0, y1: TIMELINE_HEIGHT };
  return {
    toX: (at) => (axis as TimeScale).toX(at) * stretch.x,
    createdAt,
    classOf: (node) => String(g.getNodeAttribute(node, 'cls') ?? ''),
    visible,
    yStretch: stretch.y,
  };
}

/** The edges of the archive in time, for jumping to either end. */
export function timeEnds(): { from: number; to: number } | null {
  return axis ? { from: axis.from, to: axis.to } : null;
}

/** stepFrom is the instant one claim along from here, in either direction. */
export function stepFrom(at: number, direction: 1 | -1): number | null {
  return axis ? axis.stepInstant(at, direction) : null;
}

/** axisWidth is the unstretched width of the time axis, which bounds how far it may compress. */
export function axisWidth(): number | null {
  return axis ? axis.width : null;
}

/** active is the view being drawn, which is what every stretch here acts on. */
function active(): ViewState | undefined {
  const store = useExplorer.getState();
  return store.views.find((v) => v.id === store.activeViewId);
}

/** instantAtX reads the instant under a drawn x, undoing the stretch the layout applied. */
export function instantAtX(graphX: number): number | null {
  const view = active();
  if (!axis || !view) return null;
  return axis.atX(graphX / view.xStretch);
}

/** axisXOf is where an instant sits on the drawn axis, stretch included. */
export function axisXOf(at: number): number | null {
  const view = active();
  if (!axis || !view) return null;
  return axis.toX(at) * view.xStretch;
}

/**
 * The range an axis may be stretched over. It compresses as well as stretches, and these rails
 * are the only bound on the vertical zoom.
 *
 * How far time may compress is not a constant: the floor is wherever the drawn axis has shrunk to
 * its allowed share of the viewport (-> render/bounds), which the camera reaches by its own route
 * and is held to as well. The caller measures that and passes a limit, so this is the backstop
 * for a caller that cannot measure.
 */
export const STRETCH_MIN = 1 / 4096;
export const STRETCH_MAX = 4096;

/**
 * stretchX scales the time axis and lays the drawn graph out again; stretchY does the same for the
 * strata. They act on whichever graph is showing, which is what makes a step over a lens cost a
 * thirtieth of one over the union.
 */
export function stretchX(
  factor: number,
  target?: DirectedGraph,
  /** Least stretch the caller will allow, from what it can see of the canvas. */
  floor = STRETCH_MIN,
): { stretch: number; applied: number } {
  return stretchAxis('x', factor, target, floor);
}

export function stretchY(
  factor: number,
  target?: DirectedGraph,
  floor = STRETCH_MIN,
): { stretch: number; applied: number } {
  return stretchAxis('y', factor, target, floor);
}

function stretchAxis(
  axisName: 'x' | 'y',
  factor: number,
  target?: DirectedGraph,
  floor = STRETCH_MIN,
): { stretch: number; applied: number } {
  const view = active();
  if (!view || view.layout !== 'timeline') return { stretch: 1, applied: 1 };

  const held = axisName === 'x' ? view.xStretch : view.yStretch;
  const stretched = Math.min(STRETCH_MAX, Math.max(Math.max(STRETCH_MIN, floor), held * factor));
  const applied = stretched / held;
  if (applied === 1) return { stretch: stretched, applied };
  useExplorer
    .getState()
    .patchView(view.id, axisName === 'x' ? { xStretch: stretched } : { yStretch: stretched });

  const g = target ?? graph();
  if (!axis) return { stretch: stretched, applied };
  // The other axis is laid out from what it already holds, so one stretch never disturbs it.
  layOut(g, view.classes, { ...stretchOf(view), [axisName]: stretched });
  // A lens is a copy, so stretching it leaves the union at the old scale while the view reports
  // the new one — and every measurement of the picture is taken against what the view reports.
  unsettled = g !== graph();
  return { stretch: stretched, applied };
}

/** layOut puts the drawn graph on the axis at these stretches. */
function layOut(g: DirectedGraph, classes: readonly string[], stretch: Stretch): void {
  const scale = axis;
  if (!scale) return;
  assignTimeline(g, {
    toX: (at) => scale.toX(at) * stretch.x,
    createdAt: (node) => Number(g.getNodeAttribute(node, 'createdAt') ?? 0),
    classOf: (node) => String(g.getNodeAttribute(node, 'cls') ?? ''),
    visible: classes,
    yStretch: stretch.y,
  });
}

/** Whether the union is behind the stretch the view reports, from a stretch taken over a lens. */
let unsettled = false;

/**
 * settleUnion lays the union out again where a stretch reached only a lens, and reports whether it
 * did — the caller owes the renderer an upload if so. Called before the union can be shown or cut
 * from, since a copy of a stale union is stale too.
 */
export function settleUnion(): boolean {
  if (!unsettled) return false;
  unsettled = false;
  const view = active();
  if (!view || view.layout !== 'timeline') return false;
  layOut(graph(), view.classes, stretchOf(view));
  return true;
}
