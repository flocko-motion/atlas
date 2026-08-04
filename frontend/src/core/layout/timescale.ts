/**
 * package: core / layout
 * type:    logic
 * job:     map claim instants onto an axis that spans milliseconds to centuries
 * limits:  numbers only — no dates formatted, no ticks placed, no DOM (-> ui)
 *
 * An archive is historical, so time is the axis a reader wants. Real time will not serve as
 * a coordinate: a year of silence would push everything off screen while a thousand claims
 * within one second piled into a single column, and the same archive holds both.
 *
 * So a gap contributes the logarithm of its duration. Eleven orders of magnitude of time
 * land inside a 46-fold range of distance, ordering is never violated, and nothing needs a
 * threshold or a special case. Two claims at the *same* instant contribute nothing and so
 * coincide, which is what makes stacking mean "simultaneous" and nothing else.
 *
 * A logarithm alone would put two instants a tenth of a millisecond apart within a node's
 * width, which would read as simultaneous when it is not. Hence the floor: a gap that
 * exists at all is at least `minStep` wide.
 *
 * What the axis therefore carries is ordering and the order of magnitude of every gap. It
 * does not carry measurable duration — twice the distance is not twice the elapsed time —
 * and it does not need to, because the ruler above it reads real dates out of `atX`.
 */

/** A stretch of the axis running from one instant to the next. */
export interface Span {
  from: number;
  to: number;
  x0: number;
  x1: number;
}

export interface TimeScale {
  /** Total width of the axis, 0 when every claim shares one instant. */
  width: number;
  /** The instants the axis spans. */
  from: number;
  to: number;
  /** How many distinct instants carry claims. */
  instants: number;
  /** toX maps an instant onto the axis, clamped to its ends. */
  toX(at: number): number;
  /** atX maps an axis position back to the instant it stands for — what a ruler reads. */
  atX(x: number): number;
  /**
   * nearestInstant is the instant closest to `at` that actually carries claims, which is
   * what a cursor snaps to. Snapping in time rather than in distance keeps the answer the
   * same however the axis is compressed or the camera zoomed.
   */
  nearestInstant(at: number): number;
  /**
   * stepInstant is the next instant carrying claims after `at`, or the previous one before it.
   * Distinct from `nearestInstant`, which can answer with the instant you are already on — a
   * step has to move.
   */
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

  return {
    width,
    from,
    to,
    instants,
    toX(at) {
      if (!Number.isFinite(at) || at <= from) return 0;
      if (at >= to) return width;
      const span = find(at, (s) => s.to);
      const through = span.to === span.from ? 0 : (at - span.from) / (span.to - span.from);
      return span.x0 + through * (span.x1 - span.x0);
    },
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
