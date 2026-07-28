/**
 * The view pane: how the active view draws what is loaded.
 *
 * Reading belongs to the Query tab; this is only presentation — layout and the render
 * toggles whose costs the measurements priced.
 */

import { relayout } from '../../core/session.ts';
import { LAYOUT_LABELS } from '../../core/layout/layouts.ts';
import type { LayoutName } from '../../core/layout/layouts.ts';
import { activeView, useExplorer } from '../../core/store.ts';
import { clear } from '../../core/graph/universe.ts';
import { applyViewSettings, refreshSelection } from '../../render/renderer.ts';
import { Button, Field, Select, Toggle } from '../components/Field.tsx';

const LAYOUT_OPTIONS = (Object.keys(LAYOUT_LABELS) as LayoutName[]).map((value) => ({
  value,
  label: LAYOUT_LABELS[value],
}));

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
    </div>
  );
}
