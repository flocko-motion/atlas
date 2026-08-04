/**
 * package: ui / shell
 * job:     draw the rectangle a shift-drag is marking
 * type:    view
 * limits:  presentation; the gesture and the camera are the renderer's (-> render)
 *
 * Marking a region and being shown it is the one navigation that needs no aim: a reader points
 * at what they want rather than working out how far to zoom and where to pan. The rectangle has
 * to be visible while it is drawn, or the gesture is a guess.
 */

import { useEffect, useState } from 'react';
import { onBox } from '../../render/renderer.ts';
import type { Box } from '../../render/renderer.ts';

export function ZoomBox() {
  const [box, setBox] = useState<Box | null>(null);

  useEffect(() => onBox(setBox), []);

  if (!box) return null;
  const left = Math.min(box.x0, box.x1);
  const top = Math.min(box.y0, box.y1);
  return (
    <div
      className="zoom-box"
      role="presentation"
      style={{
        left: `${left}px`,
        top: `${top}px`,
        width: `${Math.abs(box.x1 - box.x0)}px`,
        height: `${Math.abs(box.y1 - box.y0)}px`,
      }}
    />
  );
}
