/**
 * package: ui / panes
 * type:    view
 * job:     show what to read from the active source, and run it
 * limits:  presentation; the query and the run are core's (-> core/query, core/session)
 *
 * A mock archive honours the query directly. Against a real instance it is the shape of
 * the REST read, which is not wired yet — the pane says so rather than pretending.
 */

import { activeConnection, useConnections } from '../../core/connections.ts';
import { CLAIM_CLASSES, useQuery } from '../../core/query.ts';
import { load } from '../../core/session.ts';
import { useExplorer } from '../../core/store.ts';
import { Button, Field, Select, Toggle } from '../components/Field.tsx';

const LIMITS = [1000, 10000, 50000, 100000, 300000];

export function QueryPane() {
  const { query, toggleClass } = useQuery();
  const connections = useConnections((s) => s.connections);
  const activeId = useConnections((s) => s.activeId);
  const status = useExplorer((s) => s.status);
  const busy = status.busy !== null;
  const connection = connections.find((c) => c.id === activeId) ?? activeConnection();
  const isMock = connection?.kind === 'mock';

  return (
    <div className="pane">
      <p className="lede">
        Reading from <strong>{connection ? connection.name || connection.baseUrl : 'no source'}</strong>
        {isMock ? ' — a generated archive' : ' — a ranke-db instance'}.
      </p>

      <Field label="scope" hint="select.branch">
        <span className="static-value">
          {query.branch ?? 'none selected — pick one in the header'}
        </span>
      </Field>

      <Field
        label="limit"
        hint={isMock ? `archive holds ≤ ${connection?.mock.claims.toLocaleString('en-US')}` : 'limit.results'}
      >
        <Select
          value={query.limit}
          options={LIMITS.map((n) => ({ value: n, label: n.toLocaleString('en-US') }))}
          onChange={(limit) => useQuery.getState().patchQuery({ limit })}
        />
      </Field>

      <h2>classes</h2>
      {CLAIM_CLASSES.map((cls) => (
        <Toggle
          key={cls}
          label={cls}
          checked={query.classes.length === 0 || query.classes.includes(cls)}
          onChange={() => toggleClass(cls)}
        />
      ))}
      <p className="note">
        All classes are shown when none is selected. The filter is applied as the view’s
        selection over the union, so it costs a re-index rather than another read.
      </p>

      <div className="row">
        <Button variant="primary" disabled={busy} onClick={() => void load()}>
          {busy ? (status.busy ?? 'running') : 'run'}
        </Button>
        <Button disabled={busy} onClick={() => void load({ asNewView: true })}>
          run in new view
        </Button>
      </div>

      <p className="note">
        Each run merges into the union rather than replacing it, so running twice
        accumulates — which is what a session of queries does.
        {isMock
          ? ''
          : ' Reading claims from a server is not wired yet: the REST query contract has merged, but the explorer binds to no generated client so far.'}
      </p>
    </div>
  );
}
