/**
 * package: render / bounds
 * type:    logic
 * job:     the one bound on where the camera may go — how much viewport may miss the graph
 * limits:  arithmetic only; applying it to the camera is the renderer's (-> render/renderer)
 *
 * The picture moves under two independent instruments: the wheel stretches an axis in graph
 * space, and show-all, a marked box and a drag move the camera. A bound on either one alone is a
 * bound the other can walk straight past, which is how a branch selection used to land at an
 * x-zoom the wheel could neither reach nor undo.
 *
 * So the bound is stated on what a reader can actually see: the drawn graph against the
 * viewport. It is two rules over the same measurement — how small a zoom may draw the graph, and
 * where a pan may then put it — and one clamp answers both, whichever instrument moved.
 */

/**
 * At most this much of the viewport may be empty at an edge. What is bounded is the emptiness,
 * never the graph: panning into nothing is no use to a reader, while pushing part of the picture
 * off the canvas is the whole point of zooming into it.
 */
export const MAX_OUTSIDE = 0.4;

/** So this much of it must find graph, wherever the reader has pushed the picture to. */
export const MIN_COVER = 1 - MAX_OUTSIDE;

/**
 * needed is how many pixels of a viewport span must land on graph. A graph drawn smaller than
 * that can only be asked to stay wholly in view, since no position would do better; a larger one
 * may hang off either edge, so long as the edge it leaves behind is not mostly empty.
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
  return Math.min(max, Math.max(min, start)) - start;
}

/**
 * fillRatio is the camera ratio at which a drawn span comes to exactly the span asked of it, read
 * off what it measures at the ratio in force. Drawn size runs inversely with ratio, so one
 * measurement fixes the whole relation and nothing here needs to know how the renderer projects.
 */
export function fillRatio(size: number, ratio: number, viewport: number): number | null {
  if (size <= 0 || ratio <= 0 || viewport <= 0) return null;
  return (size * ratio) / viewport;
}

/** ratioCeiling is the largest camera ratio that still covers the bound's share of the viewport. */
export function ratioCeiling(size: number, ratio: number, viewport: number): number | null {
  return fillRatio(size, ratio, MIN_COVER * viewport);
}

/**
 * fillStretch is the stretch at which a drawn span comes to the span asked of it. Drawn size runs
 * with the stretch rather than against it, which is the only way this differs from fillRatio.
 */
export function fillStretch(size: number, stretch: number, viewport: number): number | null {
  if (size <= 0 || stretch <= 0 || viewport <= 0) return null;
  return (viewport * stretch) / size;
}

/** stretchFloor is the least stretch of an axis that still covers the bound's share of the viewport. */
export function stretchFloor(size: number, stretch: number, viewport: number): number | null {
  return fillStretch(size, stretch, MIN_COVER * viewport);
}
