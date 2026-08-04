/**
 * package: ui / ticks
 * type:    logic
 * job:     choose which dates to label on the time ruler, and at what unit
 * limits:  placement and wording; the axis and its inverse are core's (-> core/layout/timescale)
 *
 * Labels sit on calendar boundaries rather than at even distances, because a reader looks
 * for "the start of 2024", not "40% of the way along". The unit is chosen by how much room
 * the boundaries actually get: the finest that fits, so a coarser unit wins whenever a finer
 * one would crowd.
 *
 * Two things make this awkward, and both are handled by measuring rather than assuming. The
 * axis compresses silences, so boundaries are not evenly spaced even at one unit — hence the
 * thinning pass. And the camera zooms, so the same archive wants years at one moment and
 * milliseconds at another — hence taking the screen positions as input.
 *
 * Year boundaries are always kept and marked, whatever unit was chosen: a year is the
 * coarsest thing a reader orients by, and losing it to a thinning pass would be the one
 * omission that matters.
 */

export type TimeUnit = 'year' | 'month' | 'day' | 'hour' | 'minute' | 'second' | 'ms';

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
}

/** Rough size of each unit, for deciding how many boundaries a span would produce. */
const UNIT_MS: Record<TimeUnit, number> = {
  year: 365.25 * 86_400_000,
  month: 30.44 * 86_400_000,
  day: 86_400_000,
  hour: 3_600_000,
  minute: 60_000,
  second: 1_000,
  ms: 1,
};

/** Coarsest first, which is the order the choice is made in. */
const UNITS: TimeUnit[] = ['year', 'month', 'day', 'hour', 'minute', 'second', 'ms'];

/** More boundaries than this are never worth generating to find out they do not fit. */
const MAX_CANDIDATES = 4000;

/**
 * timeTicks picks the labels for the visible span: the finest unit whose boundaries are not
 * crowded, thinned where the axis compresses them together, with every year boundary kept.
 */
export function timeTicks(req: TickRequest): TimeTick[] {
  const { from, to } = req;
  if (!Number.isFinite(from) || !Number.isFinite(to) || to < from) return [];

  const chosen = finestUnitThatFits(req);
  let ticks = thin(boundaries(from, to, chosen).map((at) => tickAt(at, chosen, req.xOf)), req.minGap);

  // A span can contain no boundary of any unit that fits — a few minutes inside one hour, or a
  // whole archive that starts and ends mid-day. Labelling the ends of what is visible is then
  // the only thing to say, and saying nothing is what left a large archive with no ruler at all
  // until it was zoomed right in.
  if (ticks.length === 0) {
    ticks = endpointTicks(req);
  }

  // Years are never dropped, and they carry the year rather than the unit's own wording.
  if (chosen !== 'year') {
    const years = boundaries(from, to, 'year').map((at) => tickAt(at, 'year', req.xOf));
    return merge(ticks, years, req.minGap);
  }
  return ticks;
}

/**
 * endpointTicks labels the ends of the visible span, for when no calendar boundary lies inside
 * it. The unit follows the span, so the two labels differ from each other.
 */
function endpointTicks(req: TickRequest): TimeTick[] {
  const span = req.to - req.from;
  const unit: TimeUnit =
    span < 2_000 ? 'ms' : span < 2 * 60_000 ? 'second' : span < 2 * 3_600_000 ? 'minute' : 'hour';
  const ends = span === 0 ? [req.from] : [req.from, req.to];
  return thin(
    ends.map((at) => ({ x: req.xOf(at), label: wordFor(at, unit), unit, major: false })),
    req.minGap,
  );
}

/** tickAt makes one tick, worded for its unit. */
function tickAt(at: number, unit: TimeUnit, xOf: (instant: number) => number): TimeTick {
  return { x: xOf(at), label: wordFor(at, unit), unit, major: unit === 'year' };
}

/**
 * finestUnitThatFits walks coarse to fine and takes the last unit whose boundaries still
 * clear minGap on the median. The median rather than the minimum, because one compressed
 * silence should not force the whole ruler up a unit — the thinning pass handles that.
 */
function finestUnitThatFits(req: TickRequest): TimeUnit {
  // Only a unit that actually yields a boundary can be chosen; one that yields none would give
  // an empty ruler, which is worse than a coarse one.
  let best: TimeUnit | null = null;
  for (const unit of UNITS) {
    const estimate = (req.to - req.from) / UNIT_MS[unit];
    if (estimate > MAX_CANDIDATES) continue;
    const xs = boundaries(req.from, req.to, unit).map(req.xOf);
    if (xs.length === 0) continue;
    if (xs.length === 1) {
      best = unit;
      continue;
    }
    if (medianGap(xs) < req.minGap) break;
    best = unit;
  }
  return best ?? 'ms';
}

/** medianGap is the middle distance between consecutive positions. */
function medianGap(xs: number[]): number {
  const gaps: number[] = [];
  for (let i = 1; i < xs.length; i++) gaps.push(Math.abs(xs[i] - xs[i - 1]));
  gaps.sort((a, b) => a - b);
  return gaps[gaps.length >> 1] ?? Number.POSITIVE_INFINITY;
}

/** thin drops a tick that would sit too close to the one before it. */
function thin(ticks: TimeTick[], minGap: number): TimeTick[] {
  const kept: TimeTick[] = [];
  for (const tick of ticks) {
    const last = kept[kept.length - 1];
    if (last && Math.abs(tick.x - last.x) < minGap) continue;
    kept.push(tick);
  }
  return kept;
}

/** merge keeps every year and drops whatever crowds one, since a year outranks a month. */
function merge(ticks: TimeTick[], years: TimeTick[], minGap: number): TimeTick[] {
  const kept = ticks.filter((t) => !years.some((y) => Math.abs(y.x - t.x) < minGap));
  return [...kept, ...years].sort((a, b) => a.x - b.x);
}

/**
 * boundaries enumerates the starts of a unit within a span, from the first at or after
 * `from`. Calendar units step by the calendar, so months and years stay true across their
 * uneven lengths.
 */
export function boundaries(from: number, to: number, unit: TimeUnit): number[] {
  const out: number[] = [];
  if (to < from) return out;

  if (unit === 'year' || unit === 'month') {
    const start = new Date(from);
    let year = start.getUTCFullYear();
    let month = unit === 'year' ? 0 : start.getUTCMonth();
    if (Date.UTC(year, month, 1) < from) {
      if (unit === 'year') year++;
      else if (++month > 11) {
        month = 0;
        year++;
      }
    }
    for (let at = Date.UTC(year, month, 1); at <= to; ) {
      out.push(at);
      if (out.length > MAX_CANDIDATES) break;
      if (unit === 'year') year++;
      else if (++month > 11) {
        month = 0;
        year++;
      }
      at = Date.UTC(year, month, 1);
    }
    return out;
  }

  const size = UNIT_MS[unit];
  for (let at = Math.ceil(from / size) * size; at <= to; at += size) {
    out.push(at);
    if (out.length > MAX_CANDIDATES) break;
  }
  return out;
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
