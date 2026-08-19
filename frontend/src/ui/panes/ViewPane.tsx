/**
 * package: ui / panes
 * type:    view
 * job:     show how the active view draws what is loaded
 * limits:  presentation; layout and rendering are elsewhere (-> core/layout, render)
 *
 * The view pane: how the active view draws what is loaded.
 *
 * Reading belongs to the Query tab; this is only presentation — layout and the render
 * toggles whose costs the measurements priced.
 */

import { relayout } from '../../core/session.ts';
import { LAYOUT_LABELS, STRATA } from '../../core/layout/layouts.ts';
import type { LayoutName } from '../../core/layout/layouts.ts';
import { activeView, useExplorer } from '../../core/store.ts';
import { clear } from '../../core/graph/universe.ts';
import { applyViewSettings, refreshSelection } from '../../render/renderer.ts';
import { Button, Field, KeyValue, Select, Toggle } from '../components/Field.tsx';
import { formatZoom } from '../format.ts';

const LAYOUT_OPTIONS = (Object.keys(LAYOUT_LABELS) as LayoutName[]).map((value) => ({
  value,
  label: LAYOUT_LABELS[value],
}));

/**
 * toggledStrata turns a stratum on or off. An empty list means every stratum, so switching
 * the first one off has to spell out the rest — otherwise "none selected" would read as
 * "all selected" and the toggle would appear to do nothing.
 */
function toggledStrata(current: string[], stratum: string): string[] {
  const showing = current.length === 0 ? [...STRATA] : current;
  const next = showing.includes(stratum)
    ? showing.filter((s) => s !== stratum)
    : [...showing, stratum];
  // Back to everything: keep it as the empty list, which is how the reducer reads "no filter".
  return next.length === STRATA.length ? [] : next;
}

export function ViewPane() {
  const view = useExplorer(activeView);
  const patchView = useExplorer((s) => s.patchView);
  const status = useExplorer((s) => s.status);
  const busy = status.busy !== null;

  return (
    <div className="pane">
      <div className="row">
        <Button
          disabled={busy || status.nodes === 0}
          onClick={() => {
            clear();
            useExplorer.getState().patchStatus({ nodes: 0, edges: 0, contributions: 0 });
            refreshSelection();
          }}
        >
          clear session
        </Button>
      </div>

      {view ? (
        <>
          <h2>strata</h2>
          <p className="note">
            The classes the timeline draws, top of the picture first. Dropping one is a view
            predicate, so it costs a re-index rather than another read or a relayout — every
            band keeps the height and position it would have with every class shown, so
            toggling one never moves anyone else's claims.
            <code> entity/*</code> and <code>relation/*</code> share a band, the semantic
            layer, each in its own colour.
            <code> contribution/*</code> is the structural layer, worth hiding when reading
            content and worth having when analysing the archive itself.
          </p>
          {[...STRATA].reverse().map((stratum) => (
            <Toggle
              key={stratum}
              label={`${stratum}/*`}
              checked={view.classes.length === 0 || view.classes.includes(stratum)}
              onChange={() => {
                patchView(view.id, { classes: toggledStrata(view.classes, stratum) });
                refreshSelection();
              }}
            />
          ))}

          {view.layout === 'timeline' ? (
            <>
              <h2>zooming</h2>
              <KeyValue
                rows={[
                  ['wheel', `zoom time · ×${formatZoom(view.xStretch)}`],
                  ['shift + wheel', `zoom the strata · ×${formatZoom(view.yStretch)}`],
                  ['shift + drag', 'mark a region and zoom into it'],
                ]}
              />
              <p className="note">
                Each axis zooms on its own, about whatever is under the pointer: the claims are
                spread further apart rather than magnified, so a zoom on one axis leaves the other
                at the scale it had.
              </p>
            </>
          ) : null}

          <Field label="layout">
            <Select
              value={view.layout}
              options={LAYOUT_OPTIONS}
              onChange={(layout) => void relayout(layout)}
            />
          </Field>

          <Toggle
            label="draw edges"
            checked={view.edges}
            onChange={(edges) => {
              patchView(view.id, { edges });
              refreshSelection();
            }}
          />
          <Toggle
            label="edges while moving"
            checked={view.edgesOnMove}
            onChange={(edgesOnMove) => {
              patchView(view.id, { edgesOnMove });
              applyViewSettings({ ...view, edgesOnMove });
            }}
          />
          <Toggle
            label="draw labels"
            checked={view.labels}
            onChange={(labels) => {
              patchView(view.id, { labels });
              applyViewSettings({ ...view, labels });
            }}
          />
          <Toggle
            label="labels while moving"
            checked={view.labelsOnMove}
            onChange={(labelsOnMove) => {
              patchView(view.id, { labelsOnMove });
              applyViewSettings({ ...view, labelsOnMove });
            }}
          />

          <p className="note">
            Edge visibility is a reducer change, so it costs a full re-index; the two
            “while moving” toggles are settings, and cost nothing.
          </p>
        </>
      ) : null}

      <h2>renderer</h2>
      <p className="note renderer-name">{status.renderer}</p>
    </div>
  );
}
