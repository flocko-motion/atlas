/**
 * package: render / camera
 * type:    logic
 * job:     where the picture sits on the canvas, and the two zooms that scale it
 * limits:  acts on whichever instance is showing (-> render/instances); the arithmetic of the
 *          bound is bounds' and hold's (-> render/bounds, render/hold)
 *
 * A reader zooms one axis at a time, which the camera cannot do: it takes both at once and
 * magnifies the claims along with the space between them. So both zooms are stretches of the
 * layout, and the camera is left to say where the picture is.
 */

import type Sigma from 'sigma';
import { graph } from '../core/graph/universe.ts';
import { axisWidth, stretchX, stretchY } from '../core/timeline.ts';
import type { Stretch } from '../core/timeline.ts';
import { activeView, useExplorer } from '../core/store.ts';
import { stretchFloor } from './bounds.ts';
import { ceiling, drawnRect, fitStretch, holdTo, offCentreY } from './hold.ts';
import {
  both,
  canvasHeight,
  canvasWidth,
  repaint,
  showing,
  shownGraph,
} from './instances.ts';

/**
 * graphXAt converts a position on the canvas to the graph x under it. The ruler samples
 * screen space and asks what time is there, so panning and zooming need no arithmetic of
 * their own — the camera has already done it.
 */
export function graphXAt(viewportX: number): number | null {
  return showing()?.viewportToGraph({ x: viewportX, y: 0 }).x ?? null;
}

/** graphYAt is the same question about the other axis — the stratum under a point. */
export function graphYAt(viewportY: number): number | null {
  return showing()?.viewportToGraph({ x: 0, y: viewportY }).y ?? null;
}

/** viewportXAt converts a graph x to the position on the canvas it draws at. */
export function viewportXAt(graphX: number): number | null {
  const instance = showing();
  return instance?.graphToViewport({ x: graphX, y: 0 }, cameraNow(instance)).x ?? null;
}
/** stretchNow is how far each axis is stretched, which the drawn graph carries but the camera does not. */
function stretchNow(): Stretch {
  const view = activeView(useExplorer.getState());
  return { x: view?.xStretch ?? 1, y: view?.yStretch ?? 1 };
}

/**
 * applyBound gives both cameras the ceiling, so neither can be zoomed out past the picture.
 *
 * It goes in as a setting rather than onto the camera, because every settings update reinstalls
 * the camera's limits from the settings — so a ceiling written to the field is wiped by the next
 * unrelated toggle, while one held here is restored by it and applied to the live state.
 */
export function applyBound(): void {
  const limit = ceiling(showing(), canvasWidth(), stretchNow());
  for (const each of both()) {
    // Setting schedules a refresh, and a stretch applies this on every wheel tick.
    if (each && each.getSetting('maxCameraRatio') !== limit) {
      each.setSetting('maxCameraRatio', limit);
    }
  }
}

/** holdCamera brings the picture back inside the bound after something moved it. */
export function holdCamera(): void {
  holdTo(showing(), canvasWidth(), canvasHeight(), stretchNow());
}

/**
 * correctDrift moves the camera by a distance the picture has drifted on the canvas. A viewport
 * distance means nothing to the camera, so it goes through the same transform the renderer draws
 * with.
 */
function correctDrift(instance: Sigma, dx: number, dy: number): void {
  const origin = instance.viewportToFramedGraph({ x: 0, y: 0 });
  const moved = instance.viewportToFramedGraph({ x: dx, y: dy });
  const camera = instance.getCamera();
  const state = camera.getState();
  camera.setState({ ...state, x: state.x + (moved.x - origin.x), y: state.y + (moved.y - origin.y) });
}

/**
 * anchorAt pans so that a graph position lands under a given point on the canvas. Zooming moves
 * everything away from the cursor, and what a reader means by zooming is that the thing they are
 * pointing at stays put — so the camera makes up the difference.
 */
export function anchorAt(viewportX: number, graphX: number): void {
  const instance = showing();
  if (!instance) return;
  const drifted = instance.graphToViewport({ x: graphX, y: 0 }, cameraNow(instance)).x - viewportX;
  if (Math.abs(drifted) < 0.01) return;
  correctDrift(instance, drifted, 0);
}

/** anchorYAt keeps the stratum under the pointer where it is, which is the same for y. */
export function anchorYAt(viewportY: number, graphY: number): void {
  const instance = showing();
  if (!instance) return;
  const drifted = instance.graphToViewport({ x: 0, y: graphY }, cameraNow(instance)).y - viewportY;
  if (Math.abs(drifted) < 0.01) return;
  correctDrift(instance, 0, drifted);
}

/**
 * cameraNow makes a projection read the camera as it is rather than as the last frame drew it.
 * Two corrections in one frame is the ordinary case — a stretch and a hold, or a fit and the
 * load's second pass — and without this the first is measured again and applied twice.
 */
function cameraNow(instance: Sigma) {
  return { cameraState: instance.getCamera().getState() };
}

/**
 * compressionFloor is the least stretch on offer: the one that still leaves the bound's share of
 * the canvas on graph. The camera reaches the same bound by its own route, so both read the drawn
 * axis rather than each keeping a limit of its own.
 *
 * It bounds an x-zoom alone. A y-zoom compresses the axis by exactly what the camera magnifies,
 * so the drawn axis comes out the width it went in at and this floor has nothing to say about it.
 */
function compressionFloor(stretch: number): number {
  const width = axisWidth();
  const canvas = canvasWidth();
  if (width === null || width <= 0 || canvas <= 0 || stretch <= 0) return 0;
  const drawn = axisSpanOnScreen(width * stretch);
  if (drawn === null || drawn <= 0) return 0;
  return stretchFloor(drawn, stretch, canvas) ?? 0;
}

/**
 * The two zooms a reader has on a timeline: one axis apiece, each leaving the other exactly as it
 * was, and each keeping what is under the pointer under it.
 *
 * Both are stretches of the layout rather than moves of the camera. The camera magnifies the
 * claims along with the space between them and it takes both axes at once, so a camera zoom deep
 * enough to fill the height with a shallow band of strata draws that band as one solid mass. A
 * stretch spreads the claims out at the size they are, which is what a reader asked for.
 */
export function zoomX(factor: number, viewportX: number): void {
  // A stretch multiplies graph x, so where the content lands is arithmetic.
  const under = graphXAt(viewportX);
  const { applied } = stretchX(factor, shownGraph() ?? undefined, compressionFloor(stretchNow().x));
  if (applied === 1) return;
  repaint();
  if (under !== null) anchorAt(viewportX, under * applied);
  // The stretch moved the picture without moving the camera, so the camera's own share of the
  // bound has just shifted under it.
  applyBound();
  holdCamera();
}

export function zoomY(factor: number, viewportY: number): void {
  const under = graphYAt(viewportY);
  // No floor of the axis's kind: a band of strata shorter than the canvas is an ordinary picture,
  // where an archive drawn narrower than the canvas is a picture stranded in empty space.
  const { applied } = stretchY(factor, shownGraph() ?? undefined);
  if (applied === 1) return;
  repaint();
  if (under !== null) anchorYAt(viewportY, under * applied);
  holdCamera();
}

/**
 * fitHeight brings the strata to the full height of the viewport and centres them there. It is
 * the y-zoom above taken to a measured target: time is untouched, and the camera with it.
 *
 * A layout with no pinned extent has no strata to fit, so there the whole graph is framed, which
 * is the nearest thing to the question.
 */
export function fitHeight(): void {
  const instance = showing();
  if (!instance) return;
  const height = canvasHeight();
  const target = fitStretch(instance, height, stretchNow());
  if (target === null) {
    const camera = instance.getCamera();
    camera.animate({ ...camera.getState(), y: 0.5, ratio: 1 }, { duration: 180 });
    return;
  }

  const { applied } = stretchY(target / stretchNow().y, shownGraph() ?? undefined);
  if (applied !== 1) repaint();
  // Measured again after the stretch: where the strata ended up is what centring is about.
  const drift = offCentreY(instance, height, stretchNow());
  if (drift !== null && Math.abs(drift) >= 0.5) correctDrift(instance, 0, drift);
  holdCamera();
}

/**
 * axisSpanOnScreen is how wide the drawn axis currently is, in canvas pixels: the stretch and the
 * camera's ratio composed. Compression is measured against it, by the caller that owns the bound.
 */
export function axisSpanOnScreen(axisWidth: number): number | null {
  const instance = showing();
  if (!instance) return null;
  const now = cameraNow(instance);
  const left = instance.graphToViewport({ x: 0, y: 0 }, now).x;
  const right = instance.graphToViewport({ x: axisWidth, y: 0 }, now).x;
  return Math.abs(right - left);
}

/**
 * resetCamera frames the pinned extent again: the whole picture, both axes. `then` runs once the
 * animation is over, since anything that measures the picture must wait for it to stop moving.
 */
export function resetCamera(then?: () => void): void {
  const instance = showing();
  if (!instance) return;
  instance.getCamera().animate({ x: 0.5, y: 0.5, ratio: 1, angle: 0 }, { duration: 200 }, () => then?.());
}

/**
 * panIntoView brings a claim onto the canvas, and leaves the picture where it is when the claim is
 * already on it. A claim reached from a list is what the reader is now looking at, so it is
 * centred — but the zoom is theirs, and only the position moves.
 */
export function panIntoView(node: string): void {
  const instance = showing();
  const g = graph();
  if (!instance || !g.hasNode(node)) return;
  const at = instance.graphToViewport(
    { x: Number(g.getNodeAttribute(node, 'x')), y: Number(g.getNodeAttribute(node, 'y')) },
    cameraNow(instance),
  );
  const { width, height } = instance.getDimensions();
  const margin = 0.08 * Math.min(width, height);
  const inside =
    at.x >= margin && at.x <= width - margin && at.y >= margin && at.y <= height - margin;
  if (inside) return;

  const origin = instance.viewportToFramedGraph({ x: 0, y: 0 });
  const moved = instance.viewportToFramedGraph({ x: at.x - width / 2, y: at.y - height / 2 });
  const camera = instance.getCamera();
  const state = camera.getState();
  camera.animate(
    { ...state, x: state.x + (moved.x - origin.x), y: state.y + (moved.y - origin.y) },
    { duration: 220 },
  );
}

/**
 * geometry is what the bound is measured from, in the units it is measured in: the pinned extent
 * in graph coordinates, where it lands on the canvas, and how big the canvas is. Null off the
 * timeline, where nothing is pinned.
 */
export function geometry(): {
  stretch: Stretch;
  rect: { x: number; y: number; width: number; height: number };
  canvas: { width: number; height: number };
} | null {
  const instance = showing();
  const stretch = stretchNow();
  const rect = drawnRect(instance, stretch);
  if (!rect) return null;
  return { stretch, rect, canvas: { width: canvasWidth(), height: canvasHeight() } };
}

/** panTo centres a graph x, keeping the height as it is. */
export function panTo(graphX: number): void {
  const instance = showing();
  if (!instance) return;
  const framed = instance.graphToViewport({ x: graphX, y: 0 }, cameraNow(instance));
  const { width } = instance.getDimensions();
  const origin = instance.viewportToFramedGraph({ x: 0, y: 0 });
  const moved = instance.viewportToFramedGraph({ x: framed.x - width / 2, y: 0 });
  const camera = instance.getCamera();
  const state = camera.getState();
  camera.animate({ ...state, x: state.x + (moved.x - origin.x) }, { duration: 180 });
}


/** A rectangle being drawn, in canvas coordinates. */
export interface Box {
  x0: number;
  y0: number;
  x1: number;
  y1: number;
}

/**
 * zoomToBox makes a marked rectangle fill the viewport. The camera does it — a box is a request
 * about what to look at, not about what the axes mean, so nothing is laid out again.
 *
 * The ratio takes whichever side needs more room, so the whole box is shown rather than the
 * larger part of it.
 */
export function zoomToBox(box: Box): void {
  const instance = showing();
  if (!instance) return;
  const { width, height } = instance.getDimensions();
  if (width <= 0 || height <= 0) return;

  const a = instance.viewportToFramedGraph({ x: Math.min(box.x0, box.x1), y: Math.min(box.y0, box.y1) });
  const b = instance.viewportToFramedGraph({ x: Math.max(box.x0, box.x1), y: Math.max(box.y0, box.y1) });
  const fraction = Math.max(
    Math.abs(box.x1 - box.x0) / width,
    Math.abs(box.y1 - box.y0) / height,
  );
  if (!(fraction > 0)) return;

  const camera = instance.getCamera();
  const state = camera.getState();
  camera.animate(
    { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2, ratio: state.ratio * fraction },
    { duration: 180 },
  );
}

