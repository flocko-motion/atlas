/**
 * package: ui / shell
 * type:    view
 * job:     show the logo, the essential tools, and the badge naming the current source
 * limits:  presentation; the source is the connection store's (-> core/connections)
 *
 * Running a read belongs to the Query tab: a query has parameters a toolbar button
 * cannot carry. The pick/select/drag tools are declared but disabled until core has
 * interaction modes — inert is more honest than pretending.
 */

import { useConnections } from '../../core/connections.ts';
import { discoverScopes, scopeShortfall, selectScope } from '../../core/session.ts';
import { scopeOptions } from '../../core/scope.ts';
import { activeView, useExplorer } from '../../core/store.ts';

const TOOLS = [
  { id: 'pick', glyph: '⌖', label: 'Pick' },
  { id: 'select', glyph: '▭', label: 'Select' },
  { id: 'drag', glyph: '✥', label: 'Drag' },
];

/**
 * ScopePicker offers the scopes the archive holds. Selecting one narrows the view through
 * a reducer, so it costs a re-index rather than a read.
 *
 * A scope whose head never loaded is marked rather than left to draw as an empty view.
 */
function ScopePicker() {
  const scopes = useExplorer((s) => s.scopes);
  const view = useExplorer(activeView);
  const shortfall = scopeShortfall(view?.scope ?? null);
  const selected = view?.scope?.name ?? '';

  if (scopes.state === 'unknown' || scopes.state === 'error') {
    return (
      <div className="scope-picker">
        <button
          type="button"
          className="scope-load"
          onClick={() => void discoverScopes()}
          title={scopes.error ?? 'list the branches this archive holds'}
        >
          branches…
        </button>
        {scopes.state === 'error' ? <span className="scope-note">{scopes.error}</span> : null}
      </div>
    );
  }

  if (scopes.state === 'loading') {
    return (
      <div className="scope-picker">
        <span className="scope-note">listing branches…</span>
      </div>
    );
  }

  return (
    <div className="scope-picker">
      <select
        className="scope-select"
        value={selected}
        aria-label="Scope"
        onChange={(e) => {
          const next = scopes.scopes.find((s) => s.name === e.target.value) ?? null;
          void selectScope(next);
        }}
      >
        {scopeOptions(scopes.scopes, view?.scope ?? null).map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
      {shortfall > 0 ? (
        <span
          className="scope-note scope-absent"
          title="claims this scope contains that the session has not read — the archive moved, or the load was capped"
        >
          {shortfall.toLocaleString('en-US')} not cached
        </span>
      ) : null}
    </div>
  );
}

export function Header() {
  const { connections, activeId } = useConnections();
  const active = connections.find((c) => c.id === activeId) ?? connections[0];
  const setSidePane = useExplorer((s) => s.setSidePane);

  return (
    <header className="topbar">
      <div className="brand">
        <span className="brand-mark" aria-hidden="true">
          ◆
        </span>
        <span className="brand-name">Ranke Explorer</span>
      </div>

      <div className="toolbar" role="toolbar" aria-label="Tools">
        {TOOLS.map((tool) => (
          <button
            key={tool.id}
            type="button"
            className="tool"
            disabled
            title={`${tool.label} — not implemented yet`}
            aria-label={tool.label}
          >
            {tool.glyph}
          </button>
        ))}
      </div>

      <ScopePicker />

      <div className="topbar-spacer" />

      {/* The badge names the current source and is the way to change it. */}
      <button
        type="button"
        className="connection-chip"
        onClick={() => setSidePane('server')}
        title={
          active?.kind === 'mock'
            ? 'generated locally — no server. Click to configure sources.'
            : `${active?.baseUrl ?? ''} — click to configure sources`
        }
      >
        <span className={`dot ${active ? 'state-ok' : 'state-unknown'}`} aria-hidden="true" />
        {active ? active.name || active.baseUrl || 'mock' : 'no source'}
      </button>
    </header>
  );
}
