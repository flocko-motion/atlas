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

import { useState } from 'react';
import { shapeOf } from '../../core/session.ts';
import { useExplorer } from '../../core/store.ts';
import { Button, Empty, KeyValue } from '../components/Field.tsx';

export function GraphPane() {
  const status = useExplorer((s) => s.status);
  const [shape, setShape] = useState<ReturnType<typeof shapeOf> | null>(null);

  if (status.nodes === 0) return <Empty>Nothing loaded yet.</Empty>;

  return (
    <div className="pane">
      <KeyValue
        rows={[
          ['claims', status.nodes.toLocaleString('en-US')],
          ['edges', status.edges.toLocaleString('en-US')],
          ['contributions', status.contributions.toLocaleString('en-US')],
          ['edges per claim', (status.edges / (status.nodes || 1)).toFixed(2)],
        ]}
      />

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
