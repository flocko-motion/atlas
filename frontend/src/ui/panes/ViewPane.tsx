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
            The bands the timeline draws, top of the picture first. Dropping one is a view
            predicate, so it costs a re-index rather than another read —
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
                // The timeline shares its height between the bands that are shown, so hiding
                // one gives its room to the rest — which is a new layout, not a re-filter.
                if (view.layout === 'timeline') void relayout(view.layout);
                else refreshSelection();
              }}
            />
          ))}

          {view.layout === 'timeline' ? (
            <>
              <h2>zooming</h2>
              <KeyValue
                rows={[
                  ['wheel', 'zoom, as everywhere'],
                  ['shift + wheel', `stretch or compress time · ×${view.xStretch < 1 ? view.xStretch.toFixed(2) : view.xStretch.toFixed(0)}`],
                  ['shift + drag', 'mark a region and zoom into it'],
                ]}
              />
              <p className="note">
                Time is the axis, so stretching it spreads the claims out without magnifying
                them, keeping whatever is under the pointer where it is. Compressing time after
                a zoom leaves the height alone, which is a vertical stretch in all but name.
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
