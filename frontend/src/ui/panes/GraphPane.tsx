/**
 * package: ui / panes
 * type:    view
 * job:     show what the union graph currently is — size, shape, degree spread
 * limits:  presentation; the measuring is core's (-> core/graph/shape)
 *
 * The graph pane: what the union currently is — its size, shape and degree spread.
 *
 * Shape is recomputed on demand rather than kept in the store: it is derived data,
 * and holding derived data in state is how stores start lying.
 */

import { useEffect, useState } from 'react';
import { scopeCounts, selectScope, shapeOf } from '../../core/session.ts';
import { isArchive, shortHead } from '../../core/scope.ts';
import type { Scope } from '../../core/scope.ts';
import { useExplorer } from '../../core/store.ts';
import { timelineExtent } from '../../core/timeline.ts';
import { geometry } from '../../render/camera.ts';
import { onRender } from '../../render/renderer.ts';
import { Button, Empty, KeyValue, PaneTitle } from '../components/Field.tsx';
import { formatZoom } from '../format.ts';

/**
 * ScopeInfo is where a head id belongs: the picker names a scope, and this says what that
 * scope is. Under `$archive` the branch table is the interesting thing, so the branches are
 * listed — each selectable, since a reader looking at the list is choosing from it.
 */
function ScopeInfo() {
  const scopes = useExplorer((s) => s.scopes);
  const selected = scopes.selected;

  if (!selected) {
    return (
      <>
        <h2>scope</h2>
        <Empty>No branch selected — pick one in the header.</Empty>
      </>
    );
  }

  const counts = scopeCounts(selected);
  const rows: [string, string][] = [
    ['name', selected.name],
    ['head', shortHead(selected.head)],
  ];
  if (counts) {
    rows.push(['claims', counts.contains.toLocaleString('en-US')]);
    if (counts.loaded < counts.contains) {
      rows.push(['read', `${counts.loaded.toLocaleString('en-US')} of them`]);
    }
  }

  return (
    <>
      <h2>scope</h2>
      <KeyValue rows={rows} />
      {isArchive(selected) ? <BranchTable scopes={scopes.scopes} selected={selected} /> : null}
    </>
  );
}

/** BranchTable lists what the branch table holds, with each branch's head. */
function BranchTable({ scopes, selected }: { scopes: Scope[]; selected: Scope }) {
  const branches = scopes.filter((s) => !isArchive(s));
  if (branches.length === 0) return <Empty>The branch table holds no branches.</Empty>;
  return (
    <>
      <h2>branches</h2>
      <ul className="refs">
        {branches.map((scope) => (
          <li key={scope.name}>
            <button
              type="button"
              className="ref-id"
              onClick={() => void selectScope(scope)}
              title={`select ${scope.name}`}
            >
              {scope.name}
            </button>
            <span className="ref-type">{shortHead(scope.head)}</span>
          </li>
        ))}
      </ul>
      <p className="note">
        {branches.length.toLocaleString('en-US')} branch(es) under{' '}
        <code>{selected.name}</code>.
      </p>
    </>
  );
}

/**
 * Geometry says where the picture is, in the terms the bound is stated in: the extent the layout
 * was given, the stretch each axis carries, and the rectangle those two land on the canvas. A
 * picture that sits wrong is then a set of numbers rather than an impression of one.
 */
function Geometry() {
  const [at, setAt] = useState(geometry);
  useEffect(() => onRender(() => setAt(geometry())), []);
  const extent = timelineExtent();
  if (!at || !extent) return null;

  const round = (v: number) => Math.round(v).toLocaleString('en-US');
  return (
    <>
      <h2>geometry</h2>
      <KeyValue
        rows={[
          ['extent x', `0 … ${round(extent.x1 * at.stretch.x)}`],
          ['extent y', `0 … ${round(extent.y1 * at.stretch.y)}`],
          ['stretch', `${formatZoom(at.stretch.x)} / ${formatZoom(at.stretch.y)}`],
          ['drawn at', `${round(at.rect.x)}, ${round(at.rect.y)} px`],
          ['drawn size', `${round(at.rect.width)} × ${round(at.rect.height)} px`],
          ['canvas', `${round(at.canvas.width)} × ${round(at.canvas.height)} px`],
        ]}
      />
    </>
  );
}

export function GraphPane() {
  const status = useExplorer((s) => s.status);
  const [shape, setShape] = useState<ReturnType<typeof shapeOf> | null>(null);

  if (status.nodes === 0) {
    return (
      <div className="pane">
        <ScopeInfo />
        <Empty>Nothing loaded yet.</Empty>
      </div>
    );
  }

  return (
    <div className="pane">
      <PaneTitle hint="nothing selected">graph</PaneTitle>
      <ScopeInfo />

      <h2>loaded</h2>
      <KeyValue
        rows={[
          ['nodes', status.nodes.toLocaleString('en-US')],
          ['edges', status.edges.toLocaleString('en-US')],
          ['contributions', status.contributions.toLocaleString('en-US')],
          ['edges per node', (status.edges / (status.nodes || 1)).toFixed(2)],
        ]}
      />

      <Geometry />

      <div className="row">
        <Button onClick={() => setShape(shapeOf())}>measure shape</Button>
      </div>

      {shape ? (
        <>
          <h2>by provenance depth</h2>
          <KeyValue
            rows={[
              ['height', shape.depthStats.height.toLocaleString('en-US')],
              ['layers', shape.depthStats.layers.toLocaleString('en-US')],
              ['widest layer', shape.depthStats.widestLayer.toLocaleString('en-US')],
              ['pass', `${shape.depthStats.computeMs.toFixed(0)} ms`],
            ]}
          />
          <h2>by contribution</h2>
          <KeyValue
            rows={[
              ['rows', shape.history.rows.toLocaleString('en-US')],
              ['widest row', shape.history.widestRow.toLocaleString('en-US')],
              ['mean row', shape.history.meanRow.toFixed(1)],
              ['aspect', `${shape.history.aspect.toFixed(1)}:1`],
            ]}
          />
          <h2>degree</h2>
          <KeyValue
            rows={[
              ['max', shape.degree.max.toLocaleString('en-US')],
              ['mean', shape.degree.mean.toFixed(1)],
              ['p99', shape.degree.p99.toLocaleString('en-US')],
              ['hubs ≥ 100', shape.degree.hubs.toLocaleString('en-US')],
            ]}
          />
          <p className="note">
            The largest hub is a contributor claim: every claim carries a
            <code> contribution/contributor</code> edge, so its degree is the number of claims
            it signed.
          </p>
        </>
      ) : null}
    </div>
  );
}
