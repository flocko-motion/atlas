/**
 * The spike's rendering half: generate a Ranke-Graph shaped archive in the
 * browser, build it, lay it out, hand it to Sigma, then drive the camera along a
 * fixed path and time every frame.
 *
 * Everything is scripted rather than eyeballed so the numbers are comparable
 * between machines and between settings — `window.__spike.run(config)` returns
 * the same report the panel prints, which is what bench/browser-bench.ts drives.
 */

import Sigma from 'sigma';
import FA2Layout from 'graphology-layout-forceatlas2/worker';
import forceAtlas2 from 'graphology-layout-forceatlas2';
import { circlepack, circular, random } from 'graphology-layout';
import type { DirectedGraph } from 'graphology';
import { generate } from '../mock/generate.ts';
import { build, degreeStats, sizeByDegree } from '../mock/graph.ts';

export interface SpikeConfig {
  n: number;
  layout: 'random' | 'circular' | 'circlepack' | 'fa2-worker';
  /** Wall-clock budget for the worker layout, in ms. */
  fa2Ms?: number;
  edges: boolean;
  labels: boolean;
  hideEdgesOnMove: boolean;
  sizeByDegree: boolean;
  /** Frames to force in the full-refresh probe. */
  refreshProbes?: number;
  seed?: number;
}

interface FrameStats {
  frames: number;
  durationMs: number;
  fps: number;
  p50Ms: number;
  p95Ms: number;
  p99Ms: number;
  maxMs: number;
}

export interface SpikeReport {
  config: SpikeConfig;
  graph: { order: number; size: number; degree: ReturnType<typeof degreeStats> };
  stages: {
    generateMs: number;
    buildMs: number;
    layoutMs: number;
    /** Sigma constructor, which performs the first full render synchronously. */
    firstRenderMs: number;
  };
  camera: FrameStats;
  refresh: { probes: number; meanMs: number; p95Ms: number };
  device: {
    dpr: number;
    viewport: string;
    webglRenderer: string;
    memoryLimitMB: number | null;
    usedHeapMB: number | null;
  };
}

const el = <T extends HTMLElement>(id: string): T => document.getElementById(id) as T;
const logEl = el<HTMLPreElement>('log');
const resultsEl = el<HTMLTableElement>('results');
const container = el<HTMLDivElement>('graph');

function log(line: string): void {
  logEl.textContent += `${line}\n`;
  logEl.scrollTop = logEl.scrollHeight;
}

/** nextFrame resolves on the next animation frame. */
const nextFrame = (): Promise<number> => new Promise(requestAnimationFrame);

/** stats turns frame intervals into the percentiles that decide "interactive". */
function stats(intervals: number[], durationMs: number): FrameStats {
  const sorted = [...intervals].sort((a, b) => a - b);
  const at = (q: number) => sorted[Math.min(sorted.length - 1, Math.floor(sorted.length * q))] ?? 0;
  return {
    frames: intervals.length,
    durationMs,
    fps: intervals.length === 0 ? 0 : (intervals.length / durationMs) * 1000,
    p50Ms: at(0.5),
    p95Ms: at(0.95),
    p99Ms: at(0.99),
    maxMs: sorted[sorted.length - 1] ?? 0,
  };
}

/** webglRenderer names the GPU actually behind the canvas — software or real. */
function webglRenderer(): string {
  const canvas = document.createElement('canvas');
  const gl = canvas.getContext('webgl2') ?? canvas.getContext('webgl');
  if (!gl) return 'none';
  const dbg = gl.getExtension('WEBGL_debug_renderer_info');
  return String(dbg ? gl.getParameter(dbg.UNMASKED_RENDERER_WEBGL) : gl.getParameter(gl.RENDERER));
}

/** applyLayout positions nodes and returns how long it took. */
async function applyLayout(graph: DirectedGraph, cfg: SpikeConfig): Promise<number> {
  const t0 = performance.now();
  switch (cfg.layout) {
    case 'random':
      random.assign(graph, { scale: 1000 });
      break;
    case 'circular':
      circular.assign(graph, { scale: 1000 });
      break;
    case 'circlepack':
      circlepack.assign(graph, { hierarchyAttributes: ['cls'] });
      break;
    case 'fa2-worker': {
      // Seed with something cheap, then let the worker settle it for a budget.
      random.assign(graph, { scale: 1000 });
      const layout = new FA2Layout(graph, {
        settings: { ...forceAtlas2.inferSettings(graph), barnesHutOptimize: true },
      });
      layout.start();
      await new Promise((resolve) => setTimeout(resolve, cfg.fa2Ms ?? 5000));
      layout.kill();
      break;
    }
  }
  return performance.now() - t0;
}

/**
 * measureCameraPath drives a fixed pan/zoom sequence and times every painted
 * frame. The path is the same on every run, so settings are comparable.
 */
async function measureCameraPath(sigma: Sigma): Promise<FrameStats> {
  const camera = sigma.getCamera();
  camera.setState({ x: 0.5, y: 0.5, ratio: 1, angle: 0 });
  await nextFrame();

  const marks: number[] = [];
  const onRender = () => marks.push(performance.now());
  sigma.on('afterRender', onRender);

  const path = [
    { x: 0.5, y: 0.5, ratio: 0.35, angle: 0 },
    { x: 0.35, y: 0.4, ratio: 0.35, angle: 0 },
    { x: 0.65, y: 0.6, ratio: 0.12, angle: 0 },
    { x: 0.5, y: 0.5, ratio: 1.4, angle: 0 },
    { x: 0.5, y: 0.5, ratio: 1, angle: 0 },
  ];

  const t0 = performance.now();
  for (const state of path) {
    await camera.animate(state, { duration: 900, easing: 'linear' });
  }
  const durationMs = performance.now() - t0;
  sigma.off('afterRender', onRender);

  const intervals: number[] = [];
  for (let i = 1; i < marks.length; i++) intervals.push(marks[i] - marks[i - 1]);
  return stats(intervals, durationMs);
}

/**
 * measureRefresh times full refreshes — the cost of re-indexing and re-uploading
 * every node and edge, which is what a data update (not a camera move) pays.
 */
async function measureRefresh(sigma: Sigma, probes: number): Promise<{ probes: number; meanMs: number; p95Ms: number }> {
  const samples: number[] = [];
  for (let i = 0; i < probes; i++) {
    const t0 = performance.now();
    await new Promise<void>((resolve) => {
      sigma.once('afterRender', () => resolve());
      sigma.refresh();
    });
    samples.push(performance.now() - t0);
  }
  samples.sort((a, b) => a - b);
  const mean = samples.reduce((s, v) => s + v, 0) / (samples.length || 1);
  return { probes, meanMs: mean, p95Ms: samples[Math.min(samples.length - 1, Math.floor(samples.length * 0.95))] ?? 0 };
}

let current: Sigma | null = null;
let fpsHandle = 0;

/** hudFps keeps a live frame counter in the corner while a human pans around. */
function hudFps(sigma: Sigma): void {
  cancelAnimationFrame(fpsHandle);
  let frames = 0;
  let last = performance.now();
  sigma.on('afterRender', () => frames++);
  const tick = () => {
    const now = performance.now();
    if (now - last >= 500) {
      el('hud-fps').textContent = `${((frames / (now - last)) * 1000).toFixed(0)} fps painted`;
      frames = 0;
      last = now;
    }
    fpsHandle = requestAnimationFrame(tick);
  };
  fpsHandle = requestAnimationFrame(tick);
}

/** run performs one full measurement pass and returns its report. */
export async function run(cfg: SpikeConfig): Promise<SpikeReport> {
  current?.kill();
  current = null;
  container.innerHTML = '';

  log(`\n— ${cfg.n.toLocaleString('en-US')} claims · ${cfg.layout} · edges=${cfg.edges} labels=${cfg.labels}`);

  const archive = generate(cfg.n, cfg.seed ?? 0x5eed);
  log(`generate    ${archive.stats.generateMs.toFixed(0)} ms`);

  const built = build(archive, { attrs: 'full' });
  const graph = built.graph;
  log(`build       ${built.buildMs.toFixed(0)} ms · ${graph.order} nodes, ${graph.size} edges`);

  if (cfg.sizeByDegree) sizeByDegree(graph);
  const degree = degreeStats(graph);

  const layoutMs = await applyLayout(graph, cfg);
  log(`layout      ${layoutMs.toFixed(0)} ms`);

  // Sigma v3 renders the first frame *inside* the constructor (synchronously,
  // via refresh() → render()), so the constructor's own duration is the cost of
  // reaching the first frame. Awaiting an 'afterRender' listener attached
  // afterwards would wait for an event that has already fired.
  const t0 = performance.now();
  const sigma = new Sigma(graph, container, {
    renderLabels: cfg.labels,
    renderEdgeLabels: false,
    hideEdgesOnMove: cfg.hideEdgesOnMove,
    hideLabelsOnMove: true,
    labelRenderedSizeThreshold: 8,
    enableEdgeEvents: false,
    allowInvalidContainer: true,
    edgeReducer: cfg.edges ? undefined : () => ({ hidden: true }),
  });
  const firstRenderMs = performance.now() - t0;
  current = sigma;
  log(`first paint ${firstRenderMs.toFixed(0)} ms`);

  const camera = await measureCameraPath(sigma);
  log(`camera      ${camera.fps.toFixed(1)} fps · p50 ${camera.p50Ms.toFixed(1)} ms · p95 ${camera.p95Ms.toFixed(1)} ms`);

  const refresh = await measureRefresh(sigma, cfg.refreshProbes ?? 12);
  log(`refresh     ${refresh.meanMs.toFixed(1)} ms mean full redraw`);

  hudFps(sigma);
  el('hud-scale').textContent = `${graph.order.toLocaleString('en-US')} nodes · ${graph.size.toLocaleString('en-US')} edges`;

  const perf = performance as Performance & {
    memory?: { jsHeapSizeLimit: number; usedJSHeapSize: number };
  };

  const report: SpikeReport = {
    config: cfg,
    graph: { order: graph.order, size: graph.size, degree },
    stages: { generateMs: archive.stats.generateMs, buildMs: built.buildMs, layoutMs, firstRenderMs },
    camera,
    refresh,
    device: {
      dpr: window.devicePixelRatio,
      viewport: `${container.clientWidth}x${container.clientHeight}`,
      webglRenderer: webglRenderer(),
      memoryLimitMB: perf.memory ? perf.memory.jsHeapSizeLimit / 1048576 : null,
      usedHeapMB: perf.memory ? perf.memory.usedJSHeapSize / 1048576 : null,
    },
  };

  renderTable(report);
  return report;
}

/** renderTable shows the report next to the graph, for a human running this by hand. */
function renderTable(r: SpikeReport): void {
  const rows: [string, string][] = [
    ['nodes / edges', `${r.graph.order.toLocaleString('en-US')} / ${r.graph.size.toLocaleString('en-US')}`],
    ['max degree', String(r.graph.degree.max)],
    ['generate', `${r.stages.generateMs.toFixed(0)} ms`],
    ['build', `${r.stages.buildMs.toFixed(0)} ms`],
    ['layout', `${r.stages.layoutMs.toFixed(0)} ms`],
    ['first paint', `${r.stages.firstRenderMs.toFixed(0)} ms`],
    ['camera fps', r.camera.fps.toFixed(1)],
    ['frame p50 / p95', `${r.camera.p50Ms.toFixed(1)} / ${r.camera.p95Ms.toFixed(1)} ms`],
    ['full refresh', `${r.refresh.meanMs.toFixed(1)} ms`],
    ['renderer', r.device.webglRenderer],
  ];
  resultsEl.innerHTML =
    '<tr><th>metric</th><th>value</th></tr>' +
    rows.map(([k, v]) => `<tr><td>${k}</td><td>${v}</td></tr>`).join('');
}

/** configFromPanel reads the controls into a config. */
function configFromPanel(): SpikeConfig {
  return {
    n: Number(el<HTMLSelectElement>('n').value),
    layout: el<HTMLSelectElement>('layout').value as SpikeConfig['layout'],
    edges: el<HTMLInputElement>('edges').checked,
    labels: el<HTMLInputElement>('labels').checked,
    hideEdgesOnMove: el<HTMLInputElement>('hide-edges-on-move').checked,
    sizeByDegree: el<HTMLInputElement>('size-by-degree').checked,
  };
}

async function guarded(fn: () => Promise<unknown>): Promise<void> {
  const buttons = [el<HTMLButtonElement>('load'), el<HTMLButtonElement>('bench')];
  buttons.forEach((b) => (b.disabled = true));
  try {
    await fn();
  } catch (err) {
    log(`ERROR ${String(err)}`);
    throw err;
  } finally {
    buttons.forEach((b) => (b.disabled = false));
  }
}

el('load').addEventListener('click', () => void guarded(() => run(configFromPanel())));
el('bench').addEventListener('click', () =>
  void guarded(async () => {
    if (!current) await run(configFromPanel());
    else {
      const camera = await measureCameraPath(current);
      log(`camera      ${camera.fps.toFixed(1)} fps · p50 ${camera.p50Ms.toFixed(1)} ms · p95 ${camera.p95Ms.toFixed(1)} ms`);
    }
  }),
);

// The automation seam: browser-bench.ts calls this.
(window as unknown as { __spike: unknown }).__spike = { ready: true, run };

log('ready — pick a scale and press "Load & render"');
