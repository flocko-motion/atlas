/**
 * Headless half of the spike: everything a 100k-claim explorer does *before* a
 * pixel is drawn — generate, build the graphology graph, price the display
 * attributes, and lay out. All of it is plain JS on V8, the same engine the
 * browser runs it on, so these numbers transfer; only the paint does not
 * (see browser-bench.ts).
 *
 * Every scale is measured under two edge profiles, because the ADT's
 * `contribution/contributor` edge makes each contributor a star over every claim
 * it signed, and whether a renderer draws those edges changes the problem.
 *
 * Run: npm run bench -- --scales=1000,10000,100000
 */

import forceAtlas2 from 'graphology-layout-forceatlas2';
import { circlepack, circular, random } from 'graphology-layout';
import { writeFileSync } from 'node:fs';
import { cpus } from 'node:os';
import v8 from 'node:v8';
import { generate } from '../core/mock/generate.ts';
import { buildGraph as build, degreeStats } from '../core/graph/build.ts';
import { depths, historyStats } from '../core/graph/shape.ts';
import type { DepthStats } from '../core/graph/shape.ts';
import { assignHistory, assignLayered } from '../core/layout/layouts.ts';
import type { DirectedGraph } from 'graphology';

const args = new Map<string, string>();
for (const arg of process.argv.slice(2)) {
  const m = /^--([^=]+)=(.*)$/.exec(arg);
  if (m) args.set(m[1], m[2]);
}

const SCALES = (args.get('scales') ?? '1000,10000,100000').split(',').map(Number);
const SEED = Number(args.get('seed') ?? 0x5eed);
const FA2_PROBE = Number(args.get('iters') ?? 6);
const OUT = args.get('out') ?? new URL('../../results/graph-bench.json', import.meta.url).pathname;
/**
 * Claims per contribution — the archive's history granularity, which sets its
 * height. Nobody can know what real usage will pick, so it is swept rather than
 * assumed: the sweep runs at one scale and reports cost as a function of it.
 */
const GRANULARITIES = (args.get('granularities') ?? '3,10,30,100,1000').split(',').map(Number);
const GRANULARITY_SCALE = Number(args.get('granularity-scale') ?? 100000);
/** Granularity used for the per-scale tables; an assumption, flagged as one. */
const DEFAULT_GRANULARITY = Number(args.get('per-contribution') ?? 10);

/** Edge profiles: as the ADT stores them, and as a renderer might show them. */
const PROFILES = [
  { name: 'all-edges', drop: [] as string[] },
  { name: 'no-contributor-edges', drop: ['contribution/contributor'] },
];

/** gc collects if node was started with --expose-gc, so heap readings mean something. */
function gc(): void {
  const g = (globalThis as { gc?: () => void }).gc;
  if (g) {
    g();
    g();
  }
}

/** heapMB is the retained heap in MiB after a collection. */
function heapMB(): number {
  gc();
  return process.memoryUsage().heapUsed / 1048576;
}

/** ms times a thunk and returns its result alongside the elapsed milliseconds. */
function ms<T>(fn: () => T): [T, number] {
  const t0 = performance.now();
  const out = fn();
  return [out, performance.now() - t0];
}

interface Fa2Cost {
  /** One-off cost of reading the graph into typed arrays and writing back. */
  setupMs: number;
  /** Marginal cost of one iteration, with setup factored out. */
  msPerIter: number;
  /** Wall clock for a 200- and a 500-iteration settle, setup included. */
  est200Ms: number;
  est500Ms: number;
}

/**
 * fa2Cost splits ForceAtlas2 into its two costs. `assign` converts the whole
 * graph to typed arrays on every call, so timing one k-iteration call conflates
 * setup with iteration; timing 1 and k separates them — which matters, because a
 * worker pays setup once and then iterates forever.
 */
function fa2Cost(graph: DirectedGraph, k: number, barnesHut: boolean): Fa2Cost {
  const settings = { ...forceAtlas2.inferSettings(graph), barnesHutOptimize: barnesHut };
  const [, one] = ms(() => forceAtlas2.assign(graph, { iterations: 1, settings }));
  const [, many] = ms(() => forceAtlas2.assign(graph, { iterations: k, settings }));
  const msPerIter = k > 1 ? (many - one) / (k - 1) : many;
  const setupMs = Math.max(0, one - msPerIter);
  return { setupMs, msPerIter, est200Ms: setupMs + msPerIter * 200, est500Ms: setupMs + msPerIter * 500 };
}

interface ProfileResult {
  profile: string;
  edges: number;
  degree: ReturnType<typeof degreeStats>;
  fa2BarnesHut: Fa2Cost;
  fa2Plain: Fa2Cost | null;
  /** Same layout, seeded from circlepack instead of random positions. */
  fa2FromCirclepack: Fa2Cost | null;
}

interface ScaleResult {
  requested: number;
  claims: number;
  byClass: Record<string, number>;
  /** Contributions the archive was built from, and the height that implies. */
  contributions: number;
  depth: DepthStats;
  layeredMs: number;
  history: ReturnType<typeof historyStats>;
  historyMs: number;
  generateMs: number;
  /** Heap held by the claim objects — what a read has to buffer. */
  heapClaimsMB: number;
  /** JSON wire size of those claims, as a query response would carry them. */
  payloadMB: number;
  /** Cost of turning that payload back into objects — a client-side cost. */
  payloadParseMs: number;
  payloadStringifyMs: number;
  buildLeanMs: number;
  buildFullMs: number;
  /** Heap of the graphology graph alone, over and above the claims. */
  heapLeanMB: number;
  heapFullMB: number;
  mergedEdges: number;
  layout: { randomMs: number; circularMs: number; circlepackMs: number };
  profiles: ProfileResult[];
}

const pad = (v: number, w = 7, d = 0) => v.toFixed(d).padStart(w);
const out = (s: string) => process.stdout.write(s);
const results: ScaleResult[] = [];
const granularities: GranularityResult[] = [];

/**
 * warmup runs the whole pipeline once on a throwaway graph. Without it the first
 * scale measured pays V8's JIT compilation — visible as a nonsense 44 ms/iter
 * for a 1k graph that then measures 6 ms/iter on the next pass.
 */
function warmup(): void {
  const archive = generate(2000, SEED ^ 0x1234);
  const { graph } = build(archive, { attrs: 'full' });
  random.assign(graph, { scale: 1000 });
  circular.assign(graph, { scale: 1000 });
  circlepack.assign(graph, { hierarchyAttributes: [] });
  degreeStats(graph);
  JSON.stringify(archive.claims).length;
  fa2Cost(graph, 3, true);
  fa2Cost(graph, 3, false);
}

out('warming up (JIT)…\n');
warmup();

for (const requested of SCALES) {
  out(`\n=== ${requested.toLocaleString('en-US')} claims ===\n`);
  const baseline = heapMB();

  const archive = generate(requested, { seed: SEED, claimsPerContribution: DEFAULT_GRANULARITY });
  const heapClaimsMB = heapMB() - baseline;
  out(
    `  generate      ${pad(archive.stats.generateMs)} ms   ${archive.stats.claims.toLocaleString('en-US')} claims, ` +
      `${archive.stats.edges.toLocaleString('en-US')} edges, ${heapClaimsMB.toFixed(0)} MiB as objects\n` +
      `                ${archive.stats.contributions.toLocaleString('en-US')} contributions ` +
      `(~${DEFAULT_GRANULARITY} claims each)\n`,
  );

  // Both directions: the server pays stringify, the client pays parse, and the
  // client's cost is the one that sits in front of the first frame.
  const [payload, stringifyMs] = ms(() => JSON.stringify(archive.claims));
  const payloadMB = payload.length / 1048576;
  const [, parseMs] = ms(() => (JSON.parse(payload) as unknown[]).length);
  out(
    `  json payload  ${pad(payloadMB, 7, 1)} MiB  (stringify ${stringifyMs.toFixed(0)} ms, ` +
      `parse ${parseMs.toFixed(0)} ms)\n`,
  );

  // Price the display attributes: same topology, two attribute profiles, each
  // measured against the post-generate heap so only the graph is counted.
  const claimsHeap = heapMB();
  let full = build(archive, { attrs: 'full' });
  const heapFullMB = heapMB() - claimsHeap;
  const buildFullMs = full.buildMs;
  full = undefined as unknown as typeof full;
  gc();

  const lean = build(archive, { attrs: 'lean' });
  const heapLeanMB = heapMB() - claimsHeap;
  const bytesPerNode = (mb: number) => ((mb * 1048576) / lean.graph.order).toFixed(0);
  out(
    `  build lean    ${pad(lean.buildMs)} ms   ${heapLeanMB.toFixed(0)} MiB graph (${bytesPerNode(heapLeanMB)} B/node)\n` +
      `  build full    ${pad(buildFullMs)} ms   ${heapFullMB.toFixed(0)} MiB graph (${bytesPerNode(heapFullMB)} B/node)\n`,
  );

  const graph = lean.graph;

  // Height, and the layout it makes available. Depth is one pass over the graph,
  // so a layered layout costs about what a random one does.
  const { depth, stats: depthStats } = depths(graph);
  const [, layeredMs] = ms(() => assignLayered(graph, depth));

  // The two candidate axes, measured against each other. Provenance depth is what
  // the DAG gives; contribution order is what a reader wants. They disagree.
  const contributionOf = (node: string) => graph.getNodeAttribute(node, 'contribution') as number;
  const hist = historyStats(graph, contributionOf);
  const [, historyMs] = ms(() => assignHistory(graph, contributionOf));
  out(
    `  by depth      height ${depthStats.height.toLocaleString('en-US')}, ` +
      `layers ${depthStats.layers.toLocaleString('en-US')}, widest ${depthStats.widestLayer.toLocaleString('en-US')}, ` +
      `aspect ${depthStats.aspect.toFixed(2)}:1  (pass ${depthStats.computeMs.toFixed(0)} ms, ` +
      `layout ${layeredMs.toFixed(0)} ms)\n` +
      `  by history    rows ${hist.rows.toLocaleString('en-US')}, ` +
      `widest ${hist.widestRow.toLocaleString('en-US')}, mean ${hist.meanRow.toFixed(1)}, ` +
      `aspect ${hist.aspect.toFixed(2)}:1  (layout ${historyMs.toFixed(0)} ms)\n`,
  );

  const [, randomMs] = ms(() => random.assign(graph, { scale: 1000 }));
  const [, circularMs] = ms(() => circular.assign(graph, { scale: 1000 }));
  const [, circlepackMs] = ms(() => circlepack.assign(graph, { hierarchyAttributes: [] }));
  out(
    `  random        ${pad(randomMs)} ms\n` +
      `  circular      ${pad(circularMs)} ms\n` +
      `  circlepack    ${pad(circlepackMs)} ms\n`,
  );

  // Hold at most one graph at a time: at 300k, claims plus two graphs exceed a
  // 1 GiB heap, and the second profile dies mid-probe.
  let leanGraph: DirectedGraph | null = graph;
  const profiles: ProfileResult[] = [];
  for (const { name, drop } of PROFILES) {
    if (drop.length > 0 && lean.graph.order > 150000) {
      out(`  ${name}: skipped above 150k claims (one graph at a time fits the heap)\n`);
      continue;
    }
    let g: DirectedGraph;
    if (drop.length === 0) {
      g = leanGraph as DirectedGraph;
    } else {
      leanGraph = null;
      gc();
      g = build(archive, { attrs: 'lean', dropEdgeTypes: drop }).graph;
    }
    random.assign(g, { scale: 1000 }); // identical starting point for every probe
    const degree = degreeStats(g);
    // Past ~150k an iteration costs seconds, so probe with fewer of them: the
    // point is the per-iteration cost, and a bounded run is a run people repeat.
    const probe = g.order > 150000 ? 2 : FA2_PROBE;
    const fa2BarnesHut = fa2Cost(g, probe, true);
    const fa2Plain = g.order <= 20000 ? fa2Cost(g, Math.min(probe, 5), false) : null;

    // Seeding matters: Barnes-Hut cost follows the quadtree the positions imply,
    // so a tightly packed seed is not the same problem as a spread-out one.
    let fa2FromCirclepack: Fa2Cost | null = null;
    if (name === 'all-edges' && g.order <= 150000) {
      circlepack.assign(g, { hierarchyAttributes: [] });
      fa2FromCirclepack = fa2Cost(g, FA2_PROBE, true);
    }

    out(
      `  ${name}\n` +
        `    edges       ${pad(g.size)}      degree mean ${degree.mean.toFixed(1)}, p99 ${degree.p99}, ` +
        `max ${degree.max}, hubs≥100 ${degree.hubs}\n` +
        `    fa2 (bh)    ${pad(fa2BarnesHut.msPerIter, 7, 1)} ms/iter + ${fa2BarnesHut.setupMs.toFixed(0)} ms setup → ` +
        `${(fa2BarnesHut.est200Ms / 1000).toFixed(1)}s / 200 iters, ${(fa2BarnesHut.est500Ms / 1000).toFixed(1)}s / 500\n` +
        (fa2Plain ? `    fa2 (no bh) ${pad(fa2Plain.msPerIter, 7, 1)} ms/iter\n` : '') +
        (fa2FromCirclepack
          ? `    fa2 packed  ${pad(fa2FromCirclepack.msPerIter, 7, 1)} ms/iter (seeded from circlepack)\n`
          : ''),
    );
    profiles.push({ profile: name, edges: g.size, degree, fa2BarnesHut, fa2Plain, fa2FromCirclepack });
  }

  results.push({
    requested,
    claims: archive.stats.claims,
    byClass: archive.stats.byClass,
    contributions: archive.stats.contributions,
    depth: depthStats,
    layeredMs,
    history: hist,
    historyMs,
    generateMs: archive.stats.generateMs,
    heapClaimsMB,
    payloadMB,
    payloadParseMs: parseMs,
    payloadStringifyMs: stringifyMs,
    buildLeanMs: lean.buildMs,
    buildFullMs,
    heapLeanMB,
    heapFullMB,
    mergedEdges: lean.mergedEdges,
    layout: { randomMs, circularMs, circlepackMs },
    profiles,
  });

  // Written per scale, not at the end: the largest scale is the one that can run
  // the heap out, and losing the scales that already succeeded would be silly.
  save();
}

interface GranularityResult {
  claimsPerContribution: number;
  claims: number;
  contributions: number;
  edges: number;
  /** Share of claims that are heads and branch tables rather than content. */
  bookkeepingShare: number;
  depth: DepthStats;
  history: ReturnType<typeof historyStats>;
  maxDegree: number;
  buildMs: number;
  heapGraphMB: number;
  layeredMs: number;
  randomMs: number;
  fa2MsPerIter: number;
}

/**
 * sweepGranularity measures cost against history granularity at one scale.
 * Claims per contribution is the one parameter nobody can know in advance — it is
 * a usage pattern, not a property of the ADT — and it sets the archive's height.
 * So it is reported as a curve, and the reader looks up whatever their usage is.
 */
function sweepGranularity(): void {
  out(`\n=== granularity sweep at ${GRANULARITY_SCALE.toLocaleString('en-US')} claims ===\n`);
  for (const perContribution of GRANULARITIES) {
    const baseline = heapMB();
    const archive = generate(GRANULARITY_SCALE, { seed: SEED, claimsPerContribution: perContribution });
    const { graph, buildMs } = build(archive, { attrs: 'full' });
    const heapGraphMB = heapMB() - baseline;
    const { depth, stats } = depths(graph);
    const [, layeredMs] = ms(() => assignLayered(graph, depth));
    const hist = historyStats(graph, (node) => graph.getNodeAttribute(node, 'contribution') as number);
    const [, randomMs] = ms(() => random.assign(graph, { scale: 1000 }));
    const degree = degreeStats(graph);
    const fa2 = fa2Cost(graph, graph.order > 150000 ? 2 : FA2_PROBE, true);
    const contribution = (archive.stats.byClass['contribution'] ?? 0) / archive.stats.claims;

    out(
      `  ~${String(perContribution).padStart(4)} claims/contribution  ` +
        `${archive.stats.contributions.toLocaleString('en-US').padStart(7)} contributions  ` +
        `bookkeeping ${(contribution * 100).toFixed(0).padStart(2)}%  ` +
        `maxdeg ${String(degree.max).padStart(6)}  fa2 ${fa2.msPerIter.toFixed(0).padStart(5)} ms/iter\n` +
        `        by depth: height ${String(stats.height).padStart(6)}, widest ${String(stats.widestLayer).padStart(6)}` +
        `  ·  by history: rows ${String(hist.rows).padStart(6)}, widest ${String(hist.widestRow).padStart(5)}, ` +
        `mean ${hist.meanRow.toFixed(1)}\n`,
    );

    granularities.push({
      claimsPerContribution: perContribution,
      claims: archive.stats.claims,
      contributions: archive.stats.contributions,
      edges: graph.size,
      bookkeepingShare: contribution,
      depth: stats,
      history: hist,
      maxDegree: degree.max,
      buildMs,
      heapGraphMB,
      layeredMs,
      randomMs,
      fa2MsPerIter: fa2.msPerIter,
    });
    save();
  }
}

if (args.get('granularities') !== 'skip') sweepGranularity();

/** save writes everything measured so far, so a crash keeps the earlier scales. */
function save(): void {
  const report = {
    what: 'graphology build + layout cost for a Ranke-Graph shaped archive',
    caveat:
      'Heap deltas under ~10 MiB sit in the allocator noise floor; read the 100k row. ' +
      'Layout and build are pure JS on V8, so they transfer to the browser; paint does not.',
    node: process.version,
    platform: `${process.platform}/${process.arch}`,
    cpu: cpus()[0]?.model ?? 'unknown',
    cores: cpus().length,
    heapLimitMB: Math.round(v8.getHeapStatistics().heap_size_limit / 1048576),
    seed: SEED,
    fa2ProbeIterations: FA2_PROBE,
    gcExposed: Boolean((globalThis as { gc?: () => void }).gc),
    claimsPerContribution: DEFAULT_GRANULARITY,
    results,
    granularitySweep: { atClaims: GRANULARITY_SCALE, results: granularities },
  };
  writeFileSync(OUT, `${JSON.stringify(report, null, 2)}\n`);
}

save();
out(`\nwrote ${OUT}\n`);
