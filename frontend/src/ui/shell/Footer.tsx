/**
 * package: ui / shell
 * type:    view
 * job:     show what is true of this frame — where the pointer is, how the axes are scaled,
 *          and enough breakdown to attribute a bad one
 * limits:  a diagnostic, not a gate; it reads state only (-> core/store)
 *
 * What the archive holds is the Info tab's answer and is not repeated here. This line is for
 * what changes as the reader moves: the cursor, the zooms, and the cost of the last frame.
 */

import { useEffect, useState } from 'react';
import { activeView, useExplorer } from '../../core/store.ts';
import { graphXAt, graphYAt } from '../../render/camera.ts';
import { onPointer } from '../../render/renderer.ts';
import { formatZoom } from '../format.ts';

/** round writes a graph coordinate at the precision a reader can act on. */
function round(v: number): string {
  return Math.abs(v) >= 1000 ? v.toFixed(0) : v.toFixed(1);
}

/**
 * PointerCoords reads out where the pointer is in graph space — the coordinates every layout,
 * bound and extent is stated in, so a picture that sits wrong can be reported in the terms the
 * code uses rather than in "about a third of the way down".
 */
function PointerCoords() {
  const [at, setAt] = useState<{ x: number; y: number } | null>(null);

  useEffect(() => {
    return onPointer((canvas) => {
      if (canvas === null) {
        setAt(null);
        return;
      }
      const x = graphXAt(canvas.x);
      const y = graphYAt(canvas.y);
      setAt(x === null || y === null ? null : { x, y });
    });
  }, []);

  return (
    <span className="metric metric-coords" title="the pointer, in graph coordinates">
      {at === null ? '— / —' : `${round(at.x)} / ${round(at.y)}`}
    </span>
  );
}

export function Footer() {
  const status = useExplorer((s) => s.status);
  const view = useExplorer(activeView);
  const software = /swiftshader|llvmpipe|softwarepipe|mesa offscreen/i.test(status.renderer);
  const stalled = status.stallMs !== null && status.stallMs > 100;
  // Only worth saying when the view is holding something back; otherwise the Info tab has it.
  const hiding = status.nodes > 0 && status.visibleNodes < status.nodes;

  return (
    <footer className="statusbar">
      {/* Every slot is always here, saying nothing where there is nothing to say: a field that
          comes and goes moves every field beside it, and a bar that rearranges itself while the
          reader is reading it cannot be read. */}
      <span className="metric metric-shown" title="claims this view admits, of those loaded — the rest are filtered out, not missing">
        {hiding
          ? `${status.visibleNodes.toLocaleString('en-US')} of ${status.nodes.toLocaleString('en-US')} shown`
          : ''}
      </span>
      <span className="metric busy">
        {status.busy
          ? `${status.busy}${status.progress === null ? '…' : ` ${Math.round(status.progress * 100)}%`}`
          : ''}
      </span>

      <span className="statusbar-spacer" />

      <PointerCoords />
      <span className="metric metric-zoom" title="zoom, time / strata — wheel for time, shift + wheel for the strata">
        {view && view.layout === 'timeline'
          ? `zoom ${formatZoom(view.xStretch)}/${formatZoom(view.yStretch)}`
          : ''}
      </span>
      <span className="metric metric-refresh" title="cost of the last full refresh: re-index plus buffer upload, O(N)">
        {status.lastRefreshMs === null ? '' : `refresh ${status.lastRefreshMs.toFixed(0)} ms`}
      </span>
      <span
        className={`metric metric-stall${stalled ? ' warn' : ''}`}
        title="longest gap between painted frames — the main thread was blocked"
      >
        {stalled ? `stall ${status.stallMs?.toFixed(0)} ms` : ''}
      </span>
      <span className="metric fps metric-fps">
        {status.fps === null ? '— fps' : `${status.fps.toFixed(0)} fps`}
        {status.frameMs !== null ? ` · ${status.frameMs.toFixed(1)} ms` : ''}
      </span>
      {/* The GPU's name is a fact about the machine, not about what is happening — it belongs
          where it can be read once. A software renderer is different: it explains the frame
          rate, so it stays. */}
      {software ? (
        <span className="renderer warn" title={status.renderer}>
          software renderer
        </span>
      ) : null}
    </footer>
  );
}
