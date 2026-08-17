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
import { hold, ratioCeiling } from './bounds.ts';

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

/**
 * drawnGraph is where the graph sits on the canvas, in viewport pixels. x carries the stretch,
 * because stretching time moves the picture while the camera stands still — which is the whole
 * reason a bound on the camera alone was never the bound.
 */
function drawnGraph(
  showing: Sigma,
  stretch: number,
): { x: number; y: number; width: number; height: number } | null {
  if (!pinned) return null;
  const near = showing.graphToViewport({ x: pinned.x0 * stretch, y: pinned.y0 });
  const far = showing.graphToViewport({ x: pinned.x1 * stretch, y: pinned.y1 });
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
export function ceiling(showing: Sigma | null, canvas: number, stretch: number): number | null {
  if (!showing || canvas <= 0) return null;
  const rect = drawnGraph(showing, stretch);
  if (!rect) return null;
  return ratioCeiling(rect.width, showing.getCamera().getState().ratio, canvas);
}

/**
 * holdTo puts the picture back inside the bound. The ratio goes through the camera, which bounds
 * it on the way in; the pan is ours, since Sigma limits how far out a camera may zoom but not
 * where a graph may be pushed to.
 */
export function holdTo(showing: Sigma | null, width: number, height: number, stretch: number): void {
  if (!showing || !pinned || holding || width <= 0 || height <= 0) return;

  holding = true;
  try {
    const camera = showing.getCamera();
    // Re-stating the ratio is what applies a ceiling that has just moved under it.
    const state = camera.getState();
    const bounded = camera.getBoundedRatio(state.ratio);
    if (bounded !== state.ratio) camera.setState({ ...state, ratio: bounded });

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
