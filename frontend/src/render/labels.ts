/**
 * package: render / labels
 * type:    adapter
 * job:     draw a node's caption, which is two lines — what the claim is, then what it says
 * limits:  canvas drawing only; what the lines say is core's (-> core/claims)
 *
 * Sigma draws one line and stops at the first newline, so a caption that names the claim and
 * quotes its content would run both together — "note 42" reading as one phrase rather than as a
 * kind and a thing. Splitting them is the whole reason this exists.
 */

import type { Settings } from 'sigma/settings';
import type { NodeDisplayData, PartialButFor } from 'sigma/types';

/** Room between the baselines of two lines, on top of the font size. */
const LINE_GAP = 2;

type LabelData = PartialButFor<NodeDisplayData, 'x' | 'y' | 'size' | 'label' | 'color'>;

/** lines splits a caption into what is drawn, dropping the empties a stray newline leaves. */
function lines(label: string | undefined | null): string[] {
  if (!label) return [];
  return String(label)
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => line.length > 0);
}

/** face is the font Sigma was configured with, applied to the context before measuring. */
function face(context: CanvasRenderingContext2D, settings: Settings): number {
  context.font = `${settings.labelWeight} ${settings.labelSize}px ${settings.labelFont}`;
  return settings.labelSize;
}

/**
 * drawNodeLabel writes the caption beside its node. A block of lines is centred on the baseline
 * Sigma would have used for one, so adding a second line does not shift the first off the claim
 * it belongs to.
 */
export function drawNodeLabel(context: CanvasRenderingContext2D, data: LabelData, settings: Settings): void {
  const rows = lines(data.label);
  if (rows.length === 0) return;

  const size = face(context, settings);
  const attribute = settings.labelColor.attribute;
  context.fillStyle = attribute
    ? ((data as Record<string, unknown>)[attribute] as string) || settings.labelColor.color || '#000'
    : (settings.labelColor.color ?? '#000');

  const step = size + LINE_GAP;
  const first = data.y + size / 3 - ((rows.length - 1) * step) / 2;
  rows.forEach((row, i) => context.fillText(row, data.x + data.size + 3, first + i * step));
}

/**
 * drawNodeHover puts the caption on a plate, so a claim under the pointer is readable over
 * whatever it happens to be drawn on top of. The plate is the widest line by however many there
 * are, which is the only thing Sigma's own could not do.
 */
export function drawNodeHover(context: CanvasRenderingContext2D, data: LabelData, settings: Settings): void {
  const rows = lines(data.label);
  const size = face(context, settings);

  context.fillStyle = '#FFF';
  context.shadowOffsetX = 0;
  context.shadowOffsetY = 0;
  context.shadowBlur = 8;
  context.shadowColor = '#000';

  const PADDING = 2;
  if (rows.length > 0) {
    const step = size + LINE_GAP;
    const width = Math.round(Math.max(...rows.map((row) => context.measureText(row).width)) + 5);
    const height = Math.round(rows.length * step - LINE_GAP + 2 * PADDING);
    const radius = Math.max(data.size, height / 2) + PADDING;
    const angle = Math.asin(height / 2 / radius);
    const xDelta = Math.sqrt(Math.abs(radius ** 2 - (height / 2) ** 2));

    context.beginPath();
    context.moveTo(data.x + xDelta, data.y + height / 2);
    context.lineTo(data.x + radius + width, data.y + height / 2);
    context.lineTo(data.x + radius + width, data.y - height / 2);
    context.lineTo(data.x + xDelta, data.y - height / 2);
    context.arc(data.x, data.y, radius, angle, -angle);
    context.closePath();
    context.fill();
  } else {
    context.beginPath();
    context.arc(data.x, data.y, data.size + PADDING, 0, Math.PI * 2);
    context.closePath();
    context.fill();
  }

  context.shadowOffsetX = 0;
  context.shadowOffsetY = 0;
  context.shadowBlur = 0;
  drawNodeLabel(context, data, settings);
}
