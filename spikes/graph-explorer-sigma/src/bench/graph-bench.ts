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
import { generate } from '../mock/generate.ts';
import { build, degreeStats } from '../mock/graph.ts';
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

  const archive = generate(requested, SEED);
  const heapClaimsMB = heapMB() - baseline;
  out(
    `  generate      ${pad(archive.stats.generateMs)} ms   ${archive.stats.claims.toLocaleString('en-US')} claims, ` +
      `${archive.stats.edges.toLocaleString('en-US')} edges, ${heapClaimsMB.toFixed(0)} MiB as objects\n`,
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
    results,
  };
  writeFileSync(OUT, `${JSON.stringify(report, null, 2)}\n`);
}

save();
out(`\nwrote ${OUT}\n`);
