// package: ui / components
// type:    view
// job:     the form primitives — Field, Select, TextInput, Toggle, Button, KeyValue
// limits:  pure interface: each takes a value and reports a change, reading no store

/** Field labels a control and optionally hints at what the value means. */
export function Field({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <label className="field">
      <span className="field-label">
        {label}
        {hint ? <span className="field-hint">{hint}</span> : null}
      </span>
      {children}
    </label>
  );
}

export function Select<T extends string | number>({
  value,
  options,
  onChange,
}: {
  value: T;
  options: { value: T; label: string }[];
  onChange: (value: T) => void;
}) {
  return (
    <select
      value={String(value)}
      onChange={(event) => {
        const raw = event.target.value;
        const match = options.find((o) => String(o.value) === raw);
        if (match) onChange(match.value);
      }}
    >
      {options.map((option) => (
        <option key={String(option.value)} value={String(option.value)}>
          {option.label}
        </option>
      ))}
    </select>
  );
}

export function TextInput({
  value,
  onChange,
  placeholder,
  type = 'text',
}: {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  type?: 'text' | 'password' | 'url';
}) {
  return (
    <input
      type={type}
      value={value}
      placeholder={placeholder}
      spellCheck={false}
      onChange={(event) => onChange(event.target.value)}
    />
  );
}

export function Toggle({
  label,
  checked,
  onChange,
}: {
  label: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <label className="toggle">
      <input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} />
      <span>{label}</span>
    </label>
  );
}

export function Button({
  children,
  onClick,
  variant = 'default',
  disabled,
  title,
}: {
  children: React.ReactNode;
  onClick: () => void;
  variant?: 'default' | 'primary' | 'danger';
  disabled?: boolean;
  title?: string;
}) {
  return (
    <button type="button" className={`btn btn-${variant}`} onClick={onClick} disabled={disabled} title={title}>
      {children}
    </button>
  );
}

/** KeyValue renders a read-only table — the shape every detail pane needs. */
export function KeyValue({ rows }: { rows: [string, React.ReactNode][] }) {
  return (
    <table className="kv">
      <tbody>
        {rows.map(([key, value]) => (
          <tr key={key}>
            <th scope="row">{key}</th>
            <td>{value}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

/** ExtensionFields lists a record's own extension fields, and nothing when it carries none. */
export function ExtensionFields({ fields }: { fields: Readonly<Record<string, string>> }) {
  const names = Object.keys(fields);
  if (names.length === 0) return null;
  return (
    <>
      <h2>fields</h2>
      <KeyValue rows={names.map((name) => [name, fields[name]] as [string, React.ReactNode])} />
    </>
  );
}

/** Empty is the placeholder a pane shows before it has anything to say. */
export function Empty({ children }: { children: React.ReactNode }) {
  return <p className="empty">{children}</p>;
}

/**
 * PaneTitle names what a pane is answering about. The Info pane answers about three different
 * kinds of thing and swaps between them on a click, so the rows below it are only readable once
 * the reader knows which kind is in front of them.
 */
export function PaneTitle({ children, hint }: { children: React.ReactNode; hint?: string }) {
  return (
    <h1 className="pane-title">
      {children}
      {hint ? <span className="pane-title-hint">{hint}</span> : null}
    </h1>
  );
}
