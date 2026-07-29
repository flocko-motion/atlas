/**
 * package: ui / shell
 * type:    view
 * job:     show essential status, and enough breakdown to attribute a bad frame
 * limits:  a diagnostic, not a gate; it reads state only (-> core/store)
 *
 * When interaction degrades the readout should say which cost dominates — edges drawn,
 * the last full refresh, or a stalled main thread — rather than prompting a guess.
 */

import { activeView, useExplorer } from '../../core/store.ts';

export function Footer() {
  const status = useExplorer((s) => s.status);
  const view = useExplorer(activeView);
  const software = /swiftshader|llvmpipe|softwarepipe|mesa offscreen/i.test(status.renderer);
  const stalled = status.stallMs !== null && status.stallMs > 100;

  return (
    <footer className="statusbar">
      <span>
        {status.nodes.toLocaleString('en-US')} nodes · {status.edges.toLocaleString('en-US')} edges
      </span>
      <span>{status.contributions.toLocaleString('en-US')} contributions</span>
      {status.visibleNodes > 0 ? (
        <span title="admitted by the active view's reducers at the last full refresh">
          drawn {status.visibleNodes.toLocaleString('en-US')} / {status.visibleEdges.toLocaleString('en-US')}
        </span>
      ) : null}
      {status.busy ? (
        <span className="busy">
          {status.busy}
          {status.progress === null ? '…' : ` ${Math.round(status.progress * 100)}%`}
        </span>
      ) : null}

      <span className="statusbar-spacer" />

      {view ? (
        <span className="attribution" title="what the renderer is being asked to draw">
          edges {view.edges ? (view.edgesOnMove ? 'on, while moving' : 'on, hidden while moving') : 'off'}
          {' · '}
          labels {view.labels ? (view.labelsOnMove ? 'on, while moving' : 'on, hidden while moving') : 'off'}
        </span>
      ) : null}
      {status.lastRefreshMs !== null ? (
        <span title="cost of the last full refresh: re-index plus buffer upload, O(N)">
          refresh {status.lastRefreshMs.toFixed(0)} ms
        </span>
      ) : null}
      {stalled ? (
        <span className="warn" title="longest gap between painted frames — the main thread was blocked">
          stall {status.stallMs?.toFixed(0)} ms
        </span>
      ) : null}
      <span className="fps">
        {status.fps === null ? '— fps' : `${status.fps.toFixed(0)} fps`}
        {status.frameMs !== null ? ` · ${status.frameMs.toFixed(1)} ms` : ''}
      </span>
      <span className={`renderer${software ? ' warn' : ''}`} title={status.renderer}>
        {software ? 'software renderer' : status.renderer.slice(0, 40)}
      </span>
    </footer>
  );
}
