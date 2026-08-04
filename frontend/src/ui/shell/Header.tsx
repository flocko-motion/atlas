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
import { useExplorer } from '../../core/store.ts';

const TOOLS = [
  { id: 'pick', glyph: '⌖', label: 'Pick' },
  { id: 'select', glyph: '▭', label: 'Select' },
  { id: 'drag', glyph: '✥', label: 'Drag' },
];

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
