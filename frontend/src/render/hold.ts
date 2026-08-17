/**
 * package: render / hold
 * type:    logic
 * job:     keep the camera inside the bound, whichever instrument moved the picture
 * limits:  acts on the camera it is handed; which instance is showing is the renderer's
 *          (-> render/renderer), and the arithmetic is bounds' (-> render/bounds)
 *
 * The picture moves under two independent instruments, and a limit on either one is a limit the
 * other walks past — which is how selecting a branch used to land at an x-zoom the wheel could
 * neither reach nor undo. Everything here works from the drawn graph instead, so zooming out too
 * far and panning off the edge are one violation with one correction.
 */

import type Sigma from 'sigma';
import { fillStretch, hold, ratioCeiling } from './bounds.ts';

export interface Extent {
  x0: number;
  x1: number;
  y0: number;
  y1: number;
}

/** The extent the camera is held against, kept from the last pin. */
let pinned: Extent | null = null;
/** Set while the bound is moving the camera, so its own correction does not re-enter. */
let holding = false;

/** pin records what the camera is held against; null is a layout that wants plain fitting. */
export function pin(extent: Extent | null): void {
  pinned = extent;
}

/** How far each axis is stretched away from the pinned extent. */
export interface Stretch {
  x: number;
  y: number;
}

/**
 * drawnGraph is where the graph sits on the canvas, in viewport pixels. Both axes carry their
 * stretch, because stretching one moves the picture while the camera stands still — which is the
 * whole reason a bound on the camera alone was never the bound.
 */
function drawnGraph(
  showing: Sigma,
  stretch: Stretch,
): { x: number; y: number; width: number; height: number } | null {
  if (!pinned) return null;
  // Measured against the camera as it is now. Sigma's cached projection is the one the last
  // frame was drawn with, so a correction applied since would be measured a second time and
  // applied twice — which is what put a freshly loaded graph half off the bottom of the canvas.
  const now = { cameraState: showing.getCamera().getState() };
  const near = showing.graphToViewport({ x: pinned.x0 * stretch.x, y: pinned.y0 * stretch.y }, now);
  const far = showing.graphToViewport({ x: pinned.x1 * stretch.x, y: pinned.y1 * stretch.y }, now);
  return {
    x: Math.min(near.x, far.x),
    y: Math.min(near.y, far.y),
    width: Math.abs(far.x - near.x),
    height: Math.abs(far.y - near.y),
  };
}

/**
 * ceiling is the largest ratio that still leaves the bound's share of the viewport on graph. It
 * is handed to Sigma's own camera, which checks it on every state it accepts, so the wheel, an
 * animation and a plain setState all stop in the same place with no caller remembering to ask.
 */
export function ceiling(showing: Sigma | null, canvas: number, stretch: Stretch): number | null {
  if (!showing || canvas <= 0) return null;
  const rect = drawnGraph(showing, stretch);
  if (!rect) return null;
  return ratioCeiling(rect.width, showing.getCamera().getState().ratio, canvas);
}

/**
 * fitStretch is the y stretch at which the strata come to the full height of the viewport. Like
 * the ceiling it is read off the drawn picture, so it answers for wherever the camera happens to
 * be, and it is a fit rather than a bound: the whole height, not a share of it.
 */
export function fitStretch(showing: Sigma | null, height: number, stretch: Stretch): number | null {
  if (!showing || height <= 0) return null;
  const rect = drawnGraph(showing, stretch);
  if (!rect) return null;
  return fillStretch(rect.height, stretch.y, height);
}

/**
 * offCentreY is how far the drawn strata sit below the middle of the canvas. A fit answers how
 * tall they are and says nothing about where, and a picture that fills the height off-centre is
 * not the view a reader was asking for.
 */
export function offCentreY(showing: Sigma | null, height: number, stretch: Stretch): number | null {
  if (!showing || height <= 0) return null;
  const rect = drawnGraph(showing, stretch);
  if (!rect) return null;
  return rect.y + rect.height / 2 - height / 2;
}

/**
 * drawnRect is where the pinned extent sits on the canvas, for a readout. The bound is measured
 * from exactly this, so a picture that sits wrong can be reported as numbers rather than as an
 * impression of it.
 */
export function drawnRect(
  showing: Sigma | null,
  stretch: Stretch,
): { x: number; y: number; width: number; height: number } | null {
  return showing ? drawnGraph(showing, stretch) : null;
}

/**
 * holdTo puts the picture back inside the bound. Only the pan is done here: the ratio is the
 * camera's own ceiling, which it checks on every state it accepts.
 *
 * Sigma has a pan boundary of its own (`cameraPanBoundaries`), but it measures against the graph
 * Sigma knows about, and a stretch multiplies node x while the pinned extent stays unstretched —
 * so the bound it would keep is not the one the reader can see.
 */
export function holdTo(showing: Sigma | null, width: number, height: number, stretch: Stretch): void {
  if (!showing || !pinned || holding || width <= 0 || height <= 0) return;

  holding = true;
  try {
    const camera = showing.getCamera();
    const rect = drawnGraph(showing, stretch);
    if (!rect) return;
    const dx = hold(rect.x, rect.width, width);
    const dy = hold(rect.y, rect.height, height);
    if (dx === 0 && dy === 0) return;

    // A viewport distance means nothing to the camera, so it is converted through the same
    // transform the renderer draws with — and moving the graph one way moves the camera the other.
    const origin = showing.viewportToFramedGraph({ x: 0, y: 0 });
    const moved = showing.viewportToFramedGraph({ x: dx, y: dy });
    const now = camera.getState();
    camera.setState({ ...now, x: now.x - (moved.x - origin.x), y: now.y - (moved.y - origin.y) });
  } finally {
    holding = false;
  }
}
