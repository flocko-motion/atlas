/**
 * Render half of the spike: drives the explorer page in a real browser and
 * collects the frame timings the headless bench cannot produce.
 *
 * It follows the repo's convention for external counterparts (see the
 * architecture spec on adapters and podman): drive the real thing, and skip
 * cleanly — exit 0, loud message — when it is unavailable, so a machine without
 * a browser or a GPU never reports a red gate or, worse, invented numbers.
 *
 * Run: npm run bench:render -- --scales=1000,10000,100000
 */

import { spawn } from 'node:child_process';
import { writeFileSync } from 'node:fs';
import { createServer } from 'node:net';

const args = new Map<string, string>();
for (const arg of process.argv.slice(2)) {
  const m = /^--([^=]+)=(.*)$/.exec(arg);
  if (m) args.set(m[1], m[2]);
}

const SCALES = (args.get('scales') ?? '1000,10000,100000').split(',').map(Number);
const LAYOUT = args.get('layout') ?? 'circlepack';
const OUT = args.get('out') ?? new URL('../../results/render-bench.json', import.meta.url).pathname;
const HEADED = args.get('headed') === '1';

/** SKIP prints why the render bench could not run and exits successfully. */
function skip(why: string, detail?: string): never {
  process.stdout.write(`\nSKIP render bench — ${why}\n`);
  if (detail) process.stdout.write(`  ${detail.split('\n')[0]}\n`);
  process.stdout.write(
    '  This measures a GPU. Run it where there is one:\n' +
      '    npx playwright install --with-deps chromium\n' +
      '    npm run bench:render\n' +
      '  Or open the page by hand: npm run dev, then press "Load & render".\n',
  );
  process.exit(0);
}

/** freePort asks the OS for a port nobody is using. */
async function freePort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const srv = createServer();
    srv.on('error', reject);
    srv.listen(0, '127.0.0.1', () => {
      const addr = srv.address();
      const port = typeof addr === 'object' && addr ? addr.port : 0;
      srv.close(() => resolve(port));
    });
  });
}

let chromium: typeof import('playwright').chromium;
try {
  ({ chromium } = await import('playwright'));
} catch (err) {
  skip('playwright is not installed', String(err));
}

const root = new URL('../..', import.meta.url).pathname;
const port = await freePort();
// Spawn vite's bin directly rather than through npx: killing an npx wrapper
// leaves the real server orphaned, holding the port for the next run.
const vite = spawn(
  process.execPath,
  [`${root}node_modules/vite/bin/vite.js`, '--host', '127.0.0.1', '--port', String(port), '--strictPort'],
  { cwd: root, stdio: ['ignore', 'pipe', 'pipe'] },
);
const stop = () => {
  if (!vite.killed) vite.kill('SIGTERM');
};
process.on('exit', stop);

const ready = await new Promise<boolean>((resolve) => {
  const timer = setTimeout(() => resolve(false), 30000);
  const watch = (buf: Buffer) => {
    if (buf.toString().includes(String(port))) {
      clearTimeout(timer);
      resolve(true);
    }
  };
  vite.stdout.on('data', watch);
  vite.stderr.on('data', watch);
  vite.on('exit', () => {
    clearTimeout(timer);
    resolve(false);
  });
});
if (!ready) skip('the vite dev server did not start');

let browser: import('playwright').Browser;
try {
  browser = await chromium.launch({
    headless: !HEADED,
    args: ['--use-gl=angle', '--use-angle=swiftshader', '--enable-unsafe-swiftshader', '--disable-dev-shm-usage'],
  });
} catch (err) {
  skip('chromium could not launch (system libraries or browser missing)', String(err));
}

const page = await browser.newPage({ viewport: { width: 1600, height: 1000 } });
page.on('console', (msg) => {
  if (msg.type() === 'error') process.stdout.write(`  [page error] ${msg.text()}\n`);
});
await page.goto(`http://127.0.0.1:${port}/`, { waitUntil: 'load' });
await page.waitForFunction('window.__spike?.ready === true', null, { timeout: 30000 });

const renderer = await page.evaluate(() => {
  const gl = document.createElement('canvas').getContext('webgl2');
  if (!gl) return 'none';
  const dbg = gl.getExtension('WEBGL_debug_renderer_info');
  return String(dbg ? gl.getParameter(dbg.UNMASKED_RENDERER_WEBGL) : gl.getParameter(gl.RENDERER));
});
if (renderer === 'none') {
  await browser.close();
  skip('the browser has no WebGL context at all');
}
const software = /swiftshader|softwarepipe|llvmpipe|mesa offscreen/i.test(renderer);
process.stdout.write(`renderer: ${renderer}${software ? '  (SOFTWARE — frame times are a floor, not a verdict)' : ''}\n`);

const reports = [];
for (const n of SCALES) {
  process.stdout.write(`\n— ${n.toLocaleString('en-US')} claims, ${LAYOUT}\n`);
  const report = await page.evaluate(
    async ([n, layout]) => {
      const spike = (window as unknown as { __spike: { run: (c: unknown) => Promise<unknown> } }).__spike;
      // Measured at the explorer's defaults: edges drawn while moving, labels kept.
      return spike.run({
        n,
        layout,
        edges: true,
        labels: false,
        labelsOnMove: true,
        hideEdgesOnMove: false,
        sizeByDegree: true,
      });
    },
    [n, LAYOUT] as [number, string],
  );
  const r = report as {
    stages: { buildMs: number; layoutMs: number; firstRenderMs: number };
    camera: { fps: number; p50Ms: number; p95Ms: number };
    refresh: { meanMs: number };
  };
  process.stdout.write(
    `  build ${r.stages.buildMs.toFixed(0)} ms · layout ${r.stages.layoutMs.toFixed(0)} ms · ` +
      `first paint ${r.stages.firstRenderMs.toFixed(0)} ms\n` +
      `  camera ${r.camera.fps.toFixed(1)} fps (p50 ${r.camera.p50Ms.toFixed(1)} ms, p95 ${r.camera.p95Ms.toFixed(1)} ms) · ` +
      `full refresh ${r.refresh.meanMs.toFixed(1)} ms\n`,
  );
  reports.push(report);
}

await browser.close();
stop();

writeFileSync(
  OUT,
  `${JSON.stringify(
    {
      what: 'Sigma render + camera frame timings for a Ranke-Graph shaped archive',
      renderer,
      softwareRasterised: software,
      caveat: software
        ? 'Rendered by a software rasteriser: these frame times are an upper bound on cost, ' +
          'not a measurement of the GPU path. Re-run on real hardware before concluding anything about fps.'
        : 'Rendered on a real GPU.',
      layout: LAYOUT,
      reports,
    },
    null,
    2,
  )}\n`,
);
process.stdout.write(`\nwrote ${OUT}\n`);
