/**
 * package: ui / ticks
 * type:    logic
 * job:     word the ruler's precomputed tick positions, and decide how many of them a given
 *          zoom actually shows
 * limits:  formatting and render-time selection; which instants are candidates at all, and at
 *          what granularity, is core's (-> core/layout/timescale TickPosition, tickPositions)
 *
 * The heavy work — walking every claim once to decide which instants are worth a label and how
 * finely to word each — happens once, at axis build time, in core/layout/timescale.ts: a
 * candidate every so often as the axis is walked left to right, so the total is a small
 * constant regardless of how many claims fed it, each tagged with the coarsest unit that still
 * distinguishes it from its predecessor, so a burst of nearby claims earns fine detail (seconds)
 * and a claim after a long silence settles for whatever its actual distance says (a year). This
 * module only turns the few candidates that fall in view into text and thins them for the
 * current zoom's on-screen spacing — a bounded, per-frame spacing check, not a recomputation of
 * the whole candidate set.
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
 * timeTicks picks which of the axis's precomputed candidates the current zoom has room to show.
 * The candidates within [from, to] are found by binary search, then all of them run through
 * the same spacing competition (-> select) a reader would get from raw calendar boundaries —
 * cheap here in a way it never was there, because the build-time thinning in
 * core/layout/timescale.ts already bounds how many candidates exist at all (a small constant
 * regardless of how many claims fed the axis), so there is no large set left to stride or
 * sample around: running every visible candidate through the real check is the simple choice
 * once that count is already small, and it is exact for any distribution of density across the
 * visible range rather than an approximation of one.
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

/**
 * endpointTicks labels the ends of the visible span, for when it holds no precomputed
 * candidate at all — zoomed into a stretch before the first claim, after the last, or between
 * two candidates with nothing of its own. The unit follows the span, so the two labels differ.
 */
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
 * Rough px per character at the ruler's own font — tabular-nums, so every digit is the same
 * width (-> ui/style.css .time-ruler). An estimate rather than a measurement: this module
 * places and words ticks and knows no DOM or canvas to measure a rendered string with, and an
 * estimate is enough to decide *whether two labels fit*, which is all a spacing decision needs.
 */
const CHAR_PX = 6;
const LABEL_PADDING_PX = 6;

/**
 * halfWidth is roughly half a label's rendered width. A tick is drawn centred on its position
 * (`.time-tick { transform: translateX(-50%) }`), so this much clearance on each side is what
 * keeps its text off a neighbour's — a flat minGap alone said nothing about how wide the label
 * actually drawn there was.
 */
function halfWidth(label: string): number {
  return (label.length * CHAR_PX + LABEL_PADDING_PX) / 2;
}

/**
 * requiredGap is the least distance two ticks may sit at without their (centred) text
 * overlapping: their combined half-widths, or the caller's own minGap, whichever asks for more
 * room — a floor under short labels, not a ceiling over long ones.
 */
function requiredGap(a: TimeTick, b: TimeTick, minGap: number): number {
  return Math.max(minGap, halfWidth(a.label) + halfWidth(b.label));
}

/**
 * select runs every candidate through one spacing competition, the same shape as the graph's
 * own node-label budget (-> render/labels.ts): ranked coarsest unit first — a year always
 * outranks a month, a month a day, and so on down `UNITS` — so a coarse tick never loses its
 * place to a finer one that merely happened to land nearby, then by position ascending as the
 * tie-breaker within a tier, which is what reading order already is and needs no further one.
 * A candidate is kept only once it clears every tick already kept, not merely the one before
 * it, since two units can interleave in position and a list processed in position order alone
 * would let one crowd the other regardless of rank.
 */
function select(candidates: TimeTick[], minGap: number): TimeTick[] {
  const ranked = [...candidates].sort((a, b) => UNITS.indexOf(a.unit) - UNITS.indexOf(b.unit) || a.x - b.x);
  const kept: TimeTick[] = [];
  for (const tick of ranked) {
    if (kept.every((k) => Math.abs(k.x - tick.x) >= requiredGap(k, tick, minGap))) kept.push(tick);
  }
  return kept.sort((a, b) => a.x - b.x);
}

/**
 * wordFor writes a boundary as the part that distinguishes it. A month tick says "Mar"
 * because the year is on its own tick; a day says "Mar 4" because a bare number would be
 * ambiguous next to an hour.
 */
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
