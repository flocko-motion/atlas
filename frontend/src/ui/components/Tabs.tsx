/**
 * Tabs and TabHandle — the shell's one tab implementation, used by both panes.
 *
 * Pure interface: it renders the items it is given and reports intent upward. It
 * holds no state of its own, knows nothing about views, graphs or panes, and both
 * the main pane (graph views) and the side pane (tooling) use the same component.
 */

export interface TabItem {
  id: string;
  label: string;
  /** Optional short annotation, e.g. a count, shown dimmed after the label. */
  hint?: string;
  closable?: boolean;
}

export interface TabsProps {
  items: TabItem[];
  activeId: string | null;
  onSelect: (id: string) => void;
  onClose?: (id: string) => void;
  /** Rendered after the handles — an add button, usually. */
  trailing?: React.ReactNode;
  ariaLabel: string;
}

export function TabHandle({
  item,
  active,
  onSelect,
  onClose,
}: {
  item: TabItem;
  active: boolean;
  onSelect: (id: string) => void;
  onClose?: (id: string) => void;
}) {
  return (
    <div className={`tab-handle${active ? ' is-active' : ''}`} role="presentation">
      <button
        type="button"
        role="tab"
        aria-selected={active}
        className="tab-handle-label"
        onClick={() => onSelect(item.id)}
      >
        {item.label}
        {item.hint ? <span className="tab-handle-hint">{item.hint}</span> : null}
      </button>
      {item.closable && onClose ? (
        <button
          type="button"
          className="tab-handle-close"
          aria-label={`Close ${item.label}`}
          onClick={(event) => {
            event.stopPropagation();
            onClose(item.id);
          }}
        >
          ×
        </button>
      ) : null}
    </div>
  );
}

export function Tabs({ items, activeId, onSelect, onClose, trailing, ariaLabel }: TabsProps) {
  return (
    <div className="tabs" role="tablist" aria-label={ariaLabel}>
      {items.map((item) => (
        <TabHandle
          key={item.id}
          item={item}
          active={item.id === activeId}
          onSelect={onSelect}
          onClose={onClose}
        />
      ))}
      {trailing ? <div className="tabs-trailing">{trailing}</div> : null}
    </div>
  );
}
