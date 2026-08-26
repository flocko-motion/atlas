/**
 * package: core / layout
 * type:    logic
 * job:     map claim instants onto an axis that spans milliseconds to centuries, and pick which
 *          of them are worth a ruler label
 * limits:  numbers and unit tags only — no dates formatted to text, no screen space measured,
 *          no DOM (-> ui/ticks wordFor, which turns a tag into text, and the render-time stride
 *          that decides how many of these a given zoom actually shows)
 *
 * Real time will not serve as a coordinate: a year of silence would push everything off
 * screen while a thousand claims within one second pile into a column. So a gap contributes
 * the logarithm of its duration — eleven orders of magnitude land inside a 46-fold range of
 * distance, and two claims at the same instant contribute nothing, so stacking means only
 * simultaneity. A logarithm alone would still read a tenth-millisecond gap as simultaneous,
 * hence the floor: an existing gap is at least `minStep` wide. What the axis carries is
 * ordering and magnitude, not measurable duration — the ruler reads real dates out of `atX`.
 */

/** A stretch of the axis running from one instant to the next. */
export interface Span {
  from: number;
  to: number;
  x0: number;
  x1: number;
}

export type TimeUnit = 'year' | 'month' | 'day' | 'hour' | 'minute' | 'second' | 'ms';

/** Rough duration of each unit in ms — the scale `granularityFor` buckets a gap against. */
const UNIT_MS: Record<TimeUnit, number> = {
  year: 365.25 * 86_400_000,
  month: 30.44 * 86_400_000,
  day: 86_400_000,
  hour: 3_600_000,
  minute: 60_000,
  second: 1_000,
  ms: 1,
};

/** Coarsest first — a year always outranks a month when two ticks compete for the same spot. */
export const UNITS: TimeUnit[] = ['year', 'month', 'day', 'hour', 'minute', 'second', 'ms'];

/**
 * granularityFor is the coarsest unit that still distinguishes an instant from a neighbour this
 * far away — a second-apart claim reads to the second, a year-apart one to the year, each from
 * its own spacing rather than one unit chosen for the whole ruler.
 */
export function granularityFor(deltaMs: number): TimeUnit {
  for (const unit of UNITS) if (UNIT_MS[unit] <= deltaMs) return unit;
  return 'ms';
}

/** A candidate ruler label: where it sits on the unstretched axis, and at what tag — the text
 * a unit becomes ("Mar 4", "05:06:07") is `ui/ticks.ts wordFor`'s job, not this module's. */
export interface TickPosition {
  axisX: number;
  at: number;
  unit: TimeUnit;
}

/** How much finer `tickPositions` walks than a reference view needs, so zooming in past it by
 * up to this factor still finds real positions to draw without a rebuild. */
const TICK_ZOOM_HEADROOM = 4;
/** The label spacing a first, unstretched view of the whole axis wants. */
const TICK_REFERENCE_GAP_PX = 66;
/** The canvas width a first view is assumed to fill — a documented guess, since this module
 * knows no real viewport. It feeds `TICK_BUDGET` directly: too small starves it of candidates
 * (and the zoom headroom they buy); too large spends budget a narrower real viewport won't need. */
const TICK_REFERENCE_VIEWPORT_PX = 1200;

/** The main pool `tickPositions` splits by width share (-> budgetsFor) — independent of claim
 * count, so a run's natural share never grows just because there are more runs elsewhere. */
const TICK_BUDGET = Math.ceil((TICK_REFERENCE_VIEWPORT_PX / TICK_REFERENCE_GAP_PX) * TICK_ZOOM_HEADROOM);
/** A boosted run keeps at least this many candidates — see `budgetsFor` below. */
const MIN_CANDIDATES_PER_RUN = 2;
/** A second, capped pool an under-share run may draw a boost from, on top of TICK_BUDGET — a
 * swarm of needy runs draws a fixed total from it, not one growing with how many there are. The
 * candidate set is bounded overall at roughly TICK_BUDGET + FLOOR_HEADROOM plus a small constant
 * per run (each force-keeps its own first/last item), not at TICK_BUDGET alone. */
const FLOOR_HEADROOM = TICK_BUDGET;

/** A maximal, time-ordered stretch of instants that all tag the same unit. */
interface Run {
  unit: TimeUnit;
  items: { at: number; x: number }[];
}

/** coverageOf is each run's share of RESPONSIBILITY for the axis — from its own first instant to
 * the next run's first instant (axis end, for the last run), not its own internal span, which
 * leaves the gap BETWEEN runs credited to nobody. This partition always sums to the full width. */
function coverageOf(runs: readonly Run[], axisWidth: number): number[] {
  const starts = runs.map((r) => r.items[0].x);
  return runs.map((_, i) => (i + 1 < runs.length ? starts[i + 1] : axisWidth) - starts[i]);
}

/**
 * budgetsFor splits TICK_BUDGET across runs by each one's own share of the axis (-> coverageOf),
 * bounded by construction since shares of one pool sum to it. A share rounding below the floor
 * competes for a slot in `FLOOR_HEADROOM` instead of an unconditional bump (unbounded once many
 * need it) or a sub-floor fraction (worth zero). Slots go by AXIS POSITION, not share: real
 * archives rarely tie exactly, so a share-ranked sample draws from an order unrelated to where a
 * run sits, leaving some stretches bare while others get several.
 */
function budgetsFor(runs: readonly Run[], axisWidth: number): number[] {
  const widths = coverageOf(runs, axisWidth);
  const totalWidth = widths.reduce((sum, w) => sum + w, 0);
  const natural = totalWidth > 0 ? widths.map((w) => (w / totalWidth) * TICK_BUDGET) : widths.map(() => 0);

  const budgets = new Array<number>(runs.length).fill(0);
  const underShare: number[] = [];
  natural.forEach((share, i) => {
    if (Math.round(share) >= MIN_CANDIDATES_PER_RUN) budgets[i] = Math.round(share);
    else underShare.push(i);
  });

  const affordable = Math.floor(FLOOR_HEADROOM / MIN_CANDIDATES_PER_RUN);
  const byPosition = underShare.sort((a, b) => runs[a].items[0].x - runs[b].items[0].x);
  const boosted =
    byPosition.length <= affordable
      ? byPosition
      : Array.from({ length: affordable }, (_, k) => byPosition[Math.floor((k * byPosition.length) / affordable)]);
  for (const i of boosted) budgets[i] = Math.min(MIN_CANDIDATES_PER_RUN, runs[i].items.length);
  return budgets;
}

/**
 * tickPositions tags every RAW instant by its left-neighbour distance (-> granularityFor) purely
 * to form and budget RUNS (-> budgetsFor) — that tag describes claim density, not drawn spacing,
 * so it is provisional: every kept candidate is reworded after (-> retagKept). The true first and
 * last instant are always kept even where their own run drew no budget.
 */
function tickPositions(times: readonly number[], toX: (at: number) => number, axisWidth: number): TickPosition[] {
  if (times.length === 0) return [];
  const runs: Run[] = [];
  for (let i = 0; i < times.length; i++) {
    const at = times[i];
    const neighbour = i > 0 ? times[i - 1] : (times[i + 1] ?? at);
    const unit = granularityFor(Math.abs(at - neighbour));
    const item = { at, x: toX(at) };
    const current = runs[runs.length - 1];
    if (current && current.unit === unit) current.items.push(item);
    else runs.push({ unit, items: [item] });
  }

  const budgets = budgetsFor(runs, axisWidth);
  const out: { at: number; x: number }[] = [];
  runs.forEach(({ items }, r) => {
    const budget = budgets[r];
    if (budget <= 0) return;
    const width = items[items.length - 1].x - items[0].x;
    const threshold = width > 0 ? width / budget : 0;
    let lastX = -Infinity;
    for (let i = 0; i < items.length; i++) {
      const g = items[i];
      if (g.x - lastX < threshold && i !== items.length - 1) continue;
      out.push(g);
      lastX = g.x;
    }
  });

  const first = times[0];
  const last = times[times.length - 1];
  if (!out.some((p) => p.at === first)) out.push({ at: first, x: toX(first) });
  if (!out.some((p) => p.at === last)) out.push({ at: last, x: toX(last) });
  out.sort((a, b) => a.at - b.at);
  return retagKept(out);
}

/**
 * retagKept words each surviving candidate by its distance to the PREVIOUS SURVIVING one — the
 * gap thinning actually drew, once the runs that formed it are gone. The raw-sequence tag (what
 * run-forming needs) would describe density instead, reading a claim every few hours as an hour
 * however far its surviving neighbour landed — every tick on a busy whole-axis view a bare time
 * with no date. A dense stretch surviving thinning at second-or-finer spacing is meant to read
 * bare even zoomed out: a burst reads to the second because it IS one.
 */
function retagKept(kept: readonly { at: number; x: number }[]): TickPosition[] {
  return kept.map((p, i) => {
    const neighbour = i > 0 ? kept[i - 1].at : (kept[1]?.at ?? p.at);
    return { axisX: p.x, at: p.at, unit: granularityFor(Math.abs(p.at - neighbour)) };
  });
}

export interface TimeScale {
  /** Total width of the axis, 0 when every claim shares one instant. */
  width: number;
  /** The instants the axis spans. */
  from: number;
  to: number;
  /** How many distinct instants carry claims. */
  instants: number;
  /** Where a first look should open — the whole axis, less a run of remote outliers separated
   * by vast emptiness (-> denseRange). Nothing outside it is hidden: toX/atX and the ruler
   * still reach every claim. */
  denseExtent: { x0: number; x1: number };
  /** Candidate ruler labels, thinned once at build time, sorted by `at` — a small bounded set
   * the ruler zooms/pans within rather than recomputing every frame (-> tickPositions, ui/ticks.ts). */
  tickPositions: readonly TickPosition[];
  /** toX maps an instant onto the axis, clamped to its ends. */
  toX(at: number): number;
  /** atX maps an axis position back to the instant it stands for — what a ruler reads. */
  atX(x: number): number;
  /** nearestInstant is the instant closest to `at` that carries claims — what a cursor snaps
   * to. Snapping in time, not distance, keeps the answer stable under compression or zoom. */
  nearestInstant(at: number): number;
  /** stepInstant is the next (or previous) instant carrying claims — distinct from
   * nearestInstant, which may answer with the instant you're already on; a step must move. */
  stepInstant(at: number, direction: 1 | -1): number;
}

export interface TimeScaleOptions {
  /** Distance per unit of natural log of the gap in milliseconds. */
  perLog?: number;
  /** Least distance a non-zero gap gets, so distinct instants never coincide. */
  minStep?: number;
}

const DEFAULTS: Required<TimeScaleOptions> = {
  // At 6 per log-ms: 1 ms ≈ 4, one second ≈ 41, a day ≈ 110, a century ≈ 186.
  perLog: 6,
  // About a node's diameter, so a gap that exists is a gap that shows.
  minStep: 8,
};

/** gapWidth is the distance one gap contributes. Zero for simultaneity, floored above it. */
export function gapWidth(ms: number, options: TimeScaleOptions = {}): number {
  const { perLog, minStep } = { ...DEFAULTS, ...options };
  if (!(ms > 0)) return 0;
  return Math.max(minStep, perLog * Math.log1p(ms));
}

/** A gap must claim at least this share of its window before trimming — relative, since
 * absolute width is already bounded (a century tops ~186 units), so only a gap genuinely large
 * next to its neighbours reads as vast emptiness. */
const OUTLIER_GAP_FRACTION = 0.3;

/**
 * denseRange repeatedly drops the window's widest gap, keeping whichever side has more
 * instants, while that gap still claims more than `OUTLIER_GAP_FRACTION` of the window. Catches
 * a lone remote claim: its side has nothing to compete with, so the gap wins every comparison
 * until the trim reaches the archive's real substance.
 */
function denseRange(spans: Span[]): { x0: number; x1: number } {
  if (spans.length === 0) return { x0: 0, x1: 0 };
  let lo = 0;
  let hi = spans.length;
  while (hi - lo > 1) {
    let widest = lo;
    for (let i = lo + 1; i < hi; i++) if (spans[i].x1 - spans[i].x0 > spans[widest].x1 - spans[widest].x0) widest = i;
    const windowWidth = spans[hi - 1].x1 - spans[lo].x0;
    if (windowWidth <= 0 || (spans[widest].x1 - spans[widest].x0) / windowWidth < OUTLIER_GAP_FRACTION) break;
    const leftCount = widest - lo;
    const rightCount = hi - (widest + 1);
    // hi = widest, not widest + 1: the widest gap must leave the window either way, or one
    // sitting last would leave hi unchanged and loop forever.
    if (leftCount >= rightCount) hi = widest;
    else lo = widest + 1;
  }
  return { x0: spans[lo].x0, x1: spans[hi - 1].x1 };
}

/**
 * timeScale builds the axis from the instants that carry claims. Duplicates collapse, so
 * claims sharing an instant share a position and the layout stacks them.
 */
export function timeScale(instants: number[], options: TimeScaleOptions = {}): TimeScale {
  const times = [...new Set(instants.filter((t) => Number.isFinite(t)))].sort((a, b) => a - b);
  if (times.length === 0) return pointScale(0);
  if (times.length === 1) return pointScale(times[0]);

  const spans: Span[] = [];
  let x = 0;
  for (let i = 1; i < times.length; i++) {
    const width = gapWidth(times[i] - times[i - 1], options);
    spans.push({ from: times[i - 1], to: times[i], x0: x, x1: x + width });
    x += width;
  }
  return spanScale(spans, times[0], times[times.length - 1], x, times.length);
}

/** pointScale is the axis of an archive whose claims share one instant: a single position. */
function pointScale(at: number): TimeScale {
  return {
    width: 0,
    from: at,
    to: at,
    instants: at === 0 ? 0 : 1,
    denseExtent: { x0: 0, x1: 0 },
    // No neighbour to compare against, so no gap to read a granularity off — a single-instant
    // archive gets the coarsest, safest wording rather than a guess.
    tickPositions: at === 0 ? [] : [{ axisX: 0, at, unit: 'year' }],
    toX: () => 0,
    atX: () => at,
    nearestInstant: () => at,
    stepInstant: () => at,
  };
}

/** spanScale wraps spans as a TimeScale, adding the two lookups a caller needs. */
function spanScale(spans: Span[], from: number, to: number, width: number, instants: number): TimeScale {
  // Every instant that carries claims, ascending: the spans' left edges, plus the last.
  const times = [...spans.map((s) => s.from), spans[spans.length - 1].to];

  /** find returns the span holding a value, by the accessor the caller searches on. */
  const find = (value: number, hi: (s: Span) => number): Span => {
    let lo = 0;
    let up = spans.length - 1;
    while (lo < up) {
      const mid = (lo + up) >> 1;
      if (hi(spans[mid]) < value) lo = mid + 1;
      else up = mid;
    }
    return spans[lo];
  };

  /** Pulled out of the returned `toX` method so `tickPositions` can reuse it directly. */
  const toXOf = (at: number): number => {
    if (!Number.isFinite(at) || at <= from) return 0;
    if (at >= to) return width;
    const span = find(at, (s) => s.to);
    const through = span.to === span.from ? 0 : (at - span.from) / (span.to - span.from);
    return span.x0 + through * (span.x1 - span.x0);
  };

  return {
    width,
    from,
    to,
    instants,
    denseExtent: denseRange(spans),
    tickPositions: tickPositions(times, toXOf, width),
    toX: toXOf,
    atX(x) {
      if (!Number.isFinite(x) || x <= 0) return from;
      if (x >= width) return to;
      const span = find(x, (s) => s.x1);
      const across = span.x1 === span.x0 ? 0 : (x - span.x0) / (span.x1 - span.x0);
      return span.from + across * (span.to - span.from);
    },
    nearestInstant(at) {
      if (!Number.isFinite(at) || at <= from) return from;
      if (at >= to) return to;
      let lo = 0;
      let up = times.length - 1;
      while (lo < up) {
        const mid = (lo + up) >> 1;
        if (times[mid] < at) lo = mid + 1;
        else up = mid;
      }
      const after = times[lo];
      const before = lo > 0 ? times[lo - 1] : after;
      return at - before <= after - at ? before : after;
    },
    stepInstant(at, direction) {
      if (!Number.isFinite(at)) return from;
      if (direction > 0) {
        for (const t of times) if (t > at) return t;
        return to;
      }
      for (let i = times.length - 1; i >= 0; i--) if (times[i] < at) return times[i];
      return from;
    },
  };
}
