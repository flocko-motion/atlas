/**
 * package: ui / format
 * type:    logic
 * job:     write numbers and instants the way a reader reads them
 * limits:  presentation only; the values are core's (-> core)
 *
 * Formatting is the interface's business, so it lives here rather than in core — but in a
 * plain module rather than a component, so it can be tested without a DOM.
 */

/**
 * formatInstant writes an instant at the precision the visible span justifies. Reading
 * milliseconds across a decade says nothing, and reading a bare date across one minute says
 * nothing either.
 */
export function formatInstant(at: number, span: number): string {
  if (!Number.isFinite(at)) return '—';
  const iso = new Date(at).toISOString();
  if (span < 2_000) return iso.slice(11, 23);
  if (span < 2 * 60_000) return iso.slice(11, 19);
  if (span < 2 * 86_400_000) return iso.slice(11, 16);
  if (span < 60 * 86_400_000) return `${iso.slice(5, 10)} ${iso.slice(11, 16)}`;
  if (span < 3 * 365 * 86_400_000) return iso.slice(0, 10);
  return iso.slice(0, 7);
}

/**
 * formatExact writes an instant in full, to the millisecond. The ruler's labels are a scale
 * and shed precision the span does not justify; the cursor is a readout of one claim, so it
 * says exactly when — which is the only reason to point at a claim and ask.
 */
export function formatExact(at: number): string {
  if (!Number.isFinite(at)) return '—';
  const iso = new Date(at).toISOString();
  return `${iso.slice(0, 10)} ${iso.slice(11, 23)}`;
}

/** How many bytes a hex dump shows per line. */
const HEX_COLUMNS = 16;

/**
 * hexDump writes bytes the way a reader inspects them: offset, hex, and the printable ASCII
 * beside it. For content whose encoding says nothing about how to read it, this is the only
 * honest rendering — a byte is a byte.
 */
export function hexDump(bytes: Uint8Array): string {
  const lines: string[] = [];
  for (let at = 0; at < bytes.length; at += HEX_COLUMNS) {
    const row = bytes.subarray(at, at + HEX_COLUMNS);
    const hex = [...row].map((b) => b.toString(16).padStart(2, '0')).join(' ');
    const ascii = [...row].map((b) => (b >= 0x20 && b < 0x7f ? String.fromCharCode(b) : '·')).join('');
    lines.push(`${at.toString(16).padStart(6, '0')}  ${hex.padEnd(HEX_COLUMNS * 3 - 1)}  ${ascii}`);
  }
  return lines.join('\n');
}

/** isTextual reports whether an encoding says the bytes are meant to be read as characters. */
export function isTextual(encoding: string | undefined): boolean {
  if (!encoding) return false;
  return (
    encoding.startsWith('text/') ||
    encoding === 'application/json' ||
    encoding === 'application/xml' ||
    encoding.endsWith('+json') ||
    encoding.endsWith('+xml')
  );
}

/** asText decodes bytes as UTF-8, replacing anything that is not. */
export function asText(bytes: Uint8Array): string {
  return new TextDecoder('utf-8', { fatal: false }).decode(bytes);
}
