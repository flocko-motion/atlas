/**
 * package: ui / ticks
 * type:    logic
 * job:     word the ruler's precomputed tick positions, and decide how many of them a given
 *          zoom actually shows
 * limits:  formatting and render-time selection; which instants are candidates at all, and at
 *          what granularity, is core's (-> core/layout/timescale TickPosition, tickPositions)
 *
 * Candidate selection and granularity tagging (a burst of claims reads to the second, an
 * isolated one to the year) happen once, at axis build time, in core/layout/timescale.ts — a
 * small, bounded set regardless of claim count. This module only words the ones in view and
 * thins them for the current zoom, per frame.
 */

import type { TickPosition, TimeUnit } from '../core/layout/timescale.ts';
import { UNITS } from '../core/layout/timescale.ts';

export type { TimeUnit };

export interface TimeTick {
  /** Where to draw it, in the same units `xOf` returned. */
  x: number;
  label: string;
  unit: TimeUnit;
  /** True on a year boundary, which is drawn more strongly. */
  major: boolean;
}

export interface TickRequest {
  /** The visible instants, earliest first. */
  from: number;
  to: number;
  /** Where an instant lands on screen — the axis and the camera, composed. */
  xOf: (instant: number) => number;
  /** Least distance between two labels before they are considered crowded. */
  minGap: number;
  /** The axis's precomputed candidates, sorted ascending by `at` (-> core/layout/timescale). */
  positions: readonly TickPosition[];
}

/**
 * timeTicks picks which of the axis's precomputed candidates the current zoom has room to
 * show: binary search finds those within [from, to], then all of them run the spacing
 * competition (-> select). Affordable because the candidate set is already small and bounded
 * (-> core/layout/timescale.ts), so checking every visible one beats approximating with a stride.
 */
export function timeTicks(req: TickRequest): TimeTick[] {
  const { from, to, positions, minGap, xOf } = req;
  if (!Number.isFinite(from) || !Number.isFinite(to) || to < from) return [];
  if (positions.length === 0) return endpointTicks(req);

  const lo = lowerBoundAt(positions, from);
  const hi = upperBoundAt(positions, to);
  if (hi <= lo) return endpointTicks(req);

  const candidates: TimeTick[] = [];
  for (let i = lo; i < hi; i++) candidates.push(toTick(positions[i], xOf));
  return select(candidates, minGap);
}

/** toTick words one precomputed position for the ruler at its precomputed granularity. */
function toTick(p: TickPosition, xOf: (instant: number) => number): TimeTick {
  return { x: xOf(p.at), label: wordFor(p.at, p.unit), unit: p.unit, major: p.unit === 'year' };
}

/** lowerBoundAt is the first index whose `at` is not less than the given instant. */
function lowerBoundAt(positions: readonly TickPosition[], at: number): number {
  let lo = 0;
  let hi = positions.length;
  while (lo < hi) {
    const mid = (lo + hi) >> 1;
    if (positions[mid].at < at) lo = mid + 1;
    else hi = mid;
  }
  return lo;
}

/** upperBoundAt is the first index whose `at` is greater than the given instant. */
function upperBoundAt(positions: readonly TickPosition[], at: number): number {
  let lo = 0;
  let hi = positions.length;
  while (lo < hi) {
    const mid = (lo + hi) >> 1;
    if (positions[mid].at <= at) lo = mid + 1;
    else hi = mid;
  }
  return lo;
}

/** endpointTicks labels the ends of a span holding no precomputed candidate at all — before
 * the first claim, after the last, or between two with nothing of its own. */
function endpointTicks(req: TickRequest): TimeTick[] {
  const span = req.to - req.from;
  const unit: TimeUnit =
    span < 2_000 ? 'ms' : span < 2 * 60_000 ? 'second' : span < 2 * 3_600_000 ? 'minute' : 'hour';
  const ends = span === 0 ? [req.from] : [req.from, req.to];
  return select(
    ends.map((at) => ({ x: req.xOf(at), label: wordFor(at, unit), unit, major: false })),
    req.minGap,
  );
}

/**
 * Rough px per character at the ruler's tabular-nums font — an estimate, since this module has
 * no DOM or canvas to measure a rendered string with, but enough to decide whether two labels fit.
 */
const CHAR_PX = 6;
const LABEL_PADDING_PX = 6;

/** halfWidth is roughly half a label's rendered width — a tick draws centred on its position
 * (`.time-tick { transform: translateX(-50%) }`), so this is the clearance one side needs. */
function halfWidth(label: string): number {
  return (label.length * CHAR_PX + LABEL_PADDING_PX) / 2;
}

/** requiredGap is the least distance avoiding overlap: combined half-widths, or the caller's
 * minGap, whichever is more — a floor under short labels, not a ceiling over long ones. */
function requiredGap(a: TimeTick, b: TimeTick, minGap: number): number {
  return Math.max(minGap, halfWidth(a.label) + halfWidth(b.label));
}

/**
 * select ranks candidates coarsest unit first (`UNITS` order, so a year always beats a nearby
 * month or day), then by position, keeping one only once it clears every tick already kept —
 * not just its predecessor, since two units can interleave in position (-> render/labels.ts
 * for the graph's matching node-label budget).
 */
function select(candidates: TimeTick[], minGap: number): TimeTick[] {
  const ranked = [...candidates].sort((a, b) => UNITS.indexOf(a.unit) - UNITS.indexOf(b.unit) || a.x - b.x);
  const kept: TimeTick[] = [];
  for (const tick of ranked) {
    if (kept.every((k) => Math.abs(k.x - tick.x) >= requiredGap(k, tick, minGap))) kept.push(tick);
  }
  return kept.sort((a, b) => a.x - b.x);
}

/** wordFor writes a boundary as the part that distinguishes it — a month says "Mar" since the
 * year is its own tick; a day says "Mar 4" since a bare number is ambiguous next to an hour. */
export function wordFor(at: number, unit: TimeUnit): string {
  const d = new Date(at);
  const iso = d.toISOString();
  const month = MONTHS[d.getUTCMonth()];
  switch (unit) {
    case 'year':
      return String(d.getUTCFullYear());
    case 'month':
      return month;
    case 'day':
      return `${month} ${d.getUTCDate()}`;
    case 'hour':
      return iso.slice(11, 16);
    case 'minute':
      return iso.slice(11, 16);
    case 'second':
      return iso.slice(11, 19);
    case 'ms':
      return iso.slice(11, 23);
  }
}

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
