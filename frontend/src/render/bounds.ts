/**
 * package: render / bounds
 * type:    logic
 * job:     the one bound on where the camera may go — how much viewport may miss the graph
 * limits:  arithmetic only; applying it to the camera is the renderer's (-> render/renderer)
 *
 * The picture moves under two independent instruments: the shift-wheel stretches time in graph
 * space, and the plain wheel, show-all and fit-height move the camera. A bound on either one
 * alone is a bound the other can walk straight past, which is how a branch selection used to
 * land at an x-zoom the wheel could neither reach nor undo.
 *
 * So the bound is stated on what a reader can actually see: the drawn graph against the
 * viewport. Zooming out too far and panning off the edge are then the same violation, and one
 * clamp answers both, whichever instrument moved.
 */

/** At most this much of the viewport may fall outside the drawn graph. */
export const MAX_OUTSIDE = 0.4;

/** So this much of it must find graph — which on x is the width fit the wheel compresses to. */
export const MIN_COVER = 1 - MAX_OUTSIDE;

/**
 * needed is how many pixels of a viewport span must land on graph. A graph drawn smaller than
 * that can only be asked to stay wholly in view, since no camera position would do better; on x
 * the ratio ceiling keeps it from ever being that small.
 */
export function needed(size: number, viewport: number): number {
  return Math.min(MIN_COVER * viewport, size);
}

/** covered is how many pixels of the viewport the graph reaches, given where its near edge is. */
export function covered(start: number, size: number, viewport: number): number {
  return Math.max(0, Math.min(start + size, viewport) - Math.max(start, 0));
}

/**
 * holdRange is where the graph's near edge may sit and still meet the bound. Below the floor it
 * has slid off one way, above the ceiling the other; between them the reader keeps enough graph.
 */
export function holdRange(size: number, viewport: number): { min: number; max: number } {
  const must = needed(size, viewport);
  return { min: must - size, max: viewport - must };
}

/** hold puts a near edge back inside that range, and reports how far it had to move. */
export function hold(start: number, size: number, viewport: number): number {
  const { min, max } = holdRange(size, viewport);
  // A viewport narrower than the graph inverts the two, and then any position is as good.
  if (min > max) return 0;
  return Math.min(max, Math.max(min, start)) - start;
}

/**
 * ratioCeiling is the largest camera ratio that still covers the viewport, read off what the
 * graph measures at the ratio in force. Drawn size runs inversely with ratio, so one measurement
 * fixes the whole relation and nothing here needs to know how the renderer projects.
 */
export function ratioCeiling(size: number, ratio: number, viewport: number): number | null {
  if (size <= 0 || ratio <= 0 || viewport <= 0) return null;
  return (size * ratio) / (MIN_COVER * viewport);
}

/**
 * stretchFloor is the least stretch of the time axis that still covers the viewport. Drawn size
 * runs with the stretch rather than against it, which is the only way this differs from the
 * ceiling above.
 */
export function stretchFloor(size: number, stretch: number, viewport: number): number | null {
  if (size <= 0 || stretch <= 0 || viewport <= 0) return null;
  return (MIN_COVER * viewport * stretch) / size;
}
