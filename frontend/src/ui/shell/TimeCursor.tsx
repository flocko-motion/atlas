/**
 * package: ui / shell
 * type:    view
 * job:     a vertical cursor that snaps to the nearest claim and names its date
 * limits:  presentation; the snap is the axis's (-> core/layout/timescale)
 *
 * The axis compresses silences, so the pointer's position does not read as a date. The
 * cursor answers that: it snaps to the nearest instant that carries claims, draws the line
 * that column sits on, and writes the date at the top — so the picture and the ruler are
 * connected by something a reader put there.
 *
 * Snapping happens in *time*, not in distance, so the answer does not change with the
 * camera. What moves with the camera is only where the line is drawn.
 */

import { useEffect, useState } from 'react';
import { timeAxis } from '../../core/session.ts';
import { useExplorer } from '../../core/store.ts';
import { canvasWidth, graphXAt, onPointer, onRender, viewportXAt } from '../../render/renderer.ts';
import { formatExact } from '../format.ts';

interface CursorAt {
  /** Where the snapped column is on the canvas, in px. */
  x: number;
  label: string;
}

export function TimeCursor() {
  const layout = useExplorer((s) => s.views.find((v) => v.id === s.activeViewId)?.layout);
  const nodes = useExplorer((s) => s.status.nodes);
  const [at, setAt] = useState<CursorAt | null>(null);

  useEffect(() => {
    if (layout !== 'timeline' || nodes === 0) {
      setAt(null);
      return;
    }
    // The pointer's instant is remembered rather than its position, so a pan or zoom
    // redraws the line where that same claim now is.
    let instant: number | null = null;

    const redraw = () => {
      const axis = timeAxis();
      if (instant === null || !axis) {
        setAt(null);
        return;
      }
      const x = viewportXAt(axis.toX(instant));
      const width = canvasWidth();
      if (x === null || x < 0 || x > width) {
        setAt(null);
        return;
      }
      setAt({ x, label: formatExact(instant) });
    };

    const stopPointer = onPointer((px) => {
      const axis = timeAxis();
      if (px === null || !axis) {
        instant = null;
        setAt(null);
        return;
      }
      const graphX = graphXAt(px);
      if (graphX === null) return;
      instant = axis.nearestInstant(axis.atX(graphX));
      redraw();
    });
    const stopRender = onRender(redraw);
    return () => {
      stopPointer();
      stopRender();
    };
  }, [layout, nodes]);

  if (!at) return null;

  return (
    <div className="time-cursor" style={{ left: `${at.x}px` }} role="presentation">
      <span className="time-cursor-bubble">{at.label}</span>
    </div>
  );
}
