/**
 * Cooperative yielding, so a long load does not look like a hang.
 *
 * Core stays headless: this checks for `requestAnimationFrame` rather than assuming
 * it, so the same code paths run under Node in the benches. In a browser it waits
 * for an actual paint — one frame to let the pending state render, a second to be
 * sure the frame was presented — which is the difference between a progress
 * indicator that updates and one that appears only after the work is finished.
 */

const hasRaf = typeof requestAnimationFrame === 'function';

/** yieldToPaint returns once the UI has had a chance to draw. */
export function yieldToPaint(): Promise<void> {
  if (!hasRaf) return new Promise((resolve) => setTimeout(resolve, 0));
  return new Promise((resolve) => {
    requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
  });
}

/**
 * chunked walks `items` in batches, yielding between them and reporting progress.
 * The batch size is a compromise: too small and the yielding dominates, too large
 * and a frame is missed. A few thousand claims per batch keeps both acceptable.
 */
export async function chunked<T>(
  items: T[],
  batchSize: number,
  onBatch: (batch: T[], from: number) => void,
  onProgress?: (done: number, total: number) => void,
): Promise<void> {
  for (let from = 0; from < items.length; from += batchSize) {
    onBatch(items.slice(from, from + batchSize), from);
    const done = Math.min(from + batchSize, items.length);
    onProgress?.(done, items.length);
    if (done < items.length) await yieldToPaint();
  }
}
