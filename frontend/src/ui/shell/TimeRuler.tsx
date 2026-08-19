/**
 * package: ui / shell
 * type:    view
 * job:     read dates off the time axis under the canvas
 * limits:  presentation; the axis and its inverse are core's (-> core/layout/timescale)
 *
 * The axis compresses silences, so distance on it is not duration — which is exactly why
 * the dates have to be written down. The ruler samples evenly across the *screen* and asks
 * what instant is under each sample, so panning and zooming need no arithmetic here: the
 * camera has already done it, and what shows is always the span actually in view.
 */

import { useEffect, useState } from 'react';
import { timeAxis } from '../../core/timeline.ts';
import { useExplorer } from '../../core/store.ts';
import { graphXAt, viewportXAt } from '../../render/camera.ts';
import { canvasWidth } from '../../render/instances.ts';
import { onRender } from '../../render/renderer.ts';
import { timeTicks } from '../ticks.ts';
import type { TimeTick } from '../ticks.ts';

/** How close two labels may come before one is dropped, and how often to recompute. */
const MIN_GAP_PX = 66;
const THROTTLE_MS = 120;

export function TimeRuler() {
  const nodes = useExplorer((s) => s.status.nodes);
  const layout = useExplorer((s) => s.views.find((v) => v.id === s.activeViewId)?.layout);
  const [ticks, setTicks] = useState<TimeTick[]>([]);

  // Recomputed after a frame rather than on a timer: the camera moves in frames, and a
  // throttle keeps a pan from re-rendering React sixty times a second.
  useEffect(() => {
    if (layout !== 'timeline') {
      setTicks([]);
      return;
    }
    let last = 0;
    let previous = '';
    const recompute = () => {
      const now = performance.now();
      if (now - last < THROTTLE_MS) return;
      last = now;
      const next = readTicks();
      const signature = next.map((t) => `${t.label}@${t.x.toFixed(0)}`).join('|');
      if (signature === previous) return;
      previous = signature;
      setTicks(next);
    };
    recompute();
    return onRender(recompute);
  }, [layout, nodes]);

  if (ticks.length === 0) return null;

  return (
    <div className="time-ruler" role="presentation">
      {ticks.map((tick) => (
        <span
          key={`${tick.unit}:${tick.label}:${tick.x.toFixed(0)}`}
          className={`time-tick${tick.major ? ' is-major' : ''}`}
          style={{ left: `${tick.x}px` }}
        >
          {tick.label}
        </span>
      ))}
    </div>
  );
}

/**
 * readTicks asks the camera which instants the canvas currently shows, then puts the labels
 * on the calendar boundaries inside that span. Both edges come from the camera, so a pan or
 * a zoom changes the labels without the ruler knowing what a camera is.
 */
function readTicks(): TimeTick[] {
  const axis = timeAxis();
  const width = canvasWidth();
  if (!axis || axis.instants === 0 || width <= 0) return [];

  const left = graphXAt(0);
  const right = graphXAt(width);
  if (left === null || right === null) return [];

  return timeTicks({
    from: axis.atX(Math.min(left, right)),
    to: axis.atX(Math.max(left, right)),
    xOf: (instant) => viewportXAt(axis.toX(instant)) ?? 0,
    minGap: MIN_GAP_PX,
    positions: axis.tickPositions,
  }).filter((tick) => tick.x >= 0 && tick.x <= width);
}
