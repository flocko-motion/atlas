# Spike: Sigma + graphology at 100k claims

**Question.** Can a browser explorer hold and draw a 100 000-claim Ranke-Graph
interactively, using [graphology](https://graphology.github.io) for the model and
[Sigma v3](https://www.sigmajs.org) (WebGL) for the paint?

**Answer.** The paint is not the problem. Getting *to* the first frame is. On this
machine, 100k claims cost **~1.5 s of CPU and ~220 MiB of heap to become a
renderable graph — before any layout or paint** — and a settled ForceAtlas2 layout
costs **~1.3 s per iteration, 4.4 minutes for 200 iterations**. An explorer
therefore cannot fetch a 100k-claim result, lay it out client-side and show it.
Either the coordinates arrive with the data, or the client only ever
force-lays-out a bounded query result (thousands, not hundreds of thousands) and
100k appears solely as an overview positioned by a layout that costs nothing
(**circular: 17 ms**).

This is a throwaway spike, not a product frontend. RankeDB has no frontend — see
`CLAUDE.md` — and nothing here is wired into the Go build, the OpenAPI pipeline or
`make verify`. It exists to produce the numbers below, which are the deliverable;
the code is scaffolding around them.

## What was measured, and what was not

| Leg | Measured here | Why |
|---|---|---|
| Generate mock claims | ✅ | pure JS |
| Build the graphology graph | ✅ | pure JS |
| Heap and wire size | ✅ | pure JS |
| Layout (random, circular, circlepack, ForceAtlas2) | ✅ | pure JS — same V8 as the browser |
| Sigma paint and camera frame times | ❌ | needs a browser with a GPU; this container has neither |
| VRAM | ❌ | not instrumented at all — it needs browser tooling, not a harness |

The container has no browser: `chrome-headless-shell` is missing 17 system
libraries (`libnspr4`, `libnss3`, `libX11`, `libgbm`, …) and there is no root to
install them. `npm run bench:render` therefore **skips loudly and exits 0** —
the same convention the repo's adapter tests use for podman-backed counterparts.
No frame-rate number is stated anywhere in this document, because none was taken:
a software rasteriser would have measured swiftshader, not the GPU path.

The unmeasured leg does not change the verdict. The costs that rule out
client-side layout at 100k are all CPU-side, and were measured.

**The render code has therefore never executed.** It type-checks (`tsc --noEmit`)
and bundles (`vite build`), and its API use was checked against Sigma's and
graphology's sources — which is how one bug was found without running it: Sigma v3
renders the first frame *synchronously inside the constructor*, so awaiting an
`afterRender` listener attached afterwards waits forever. Assume there may be
more, and treat the first run on real hardware as a debugging session.

## Running it

```sh
npm install
npm run bench                          # the headless numbers below
npm run bench -- --scales=1000,300000  # any scales you like
npm run bench:render                   # frame timings (needs a browser + GPU)
npm run dev                            # the explorer itself, by hand
```

The explorer page has the same measurement built in: pick a scale, press **Load &
render**, and it reports build, layout, first paint, a scripted camera path and a
full-refresh cost. `npm run bench:render` just drives that page through
Playwright and writes `results/render-bench.json`. Both call the identical
`window.__spike.run(config)`, so hand-run and automated numbers are comparable.

## The mock graph

Generated to the ADT's shape (paper 01, §Type Vocabulary), not to a random graph,
because topology is what layout and rendering react to:

- `source/*` roots, `derivation/*` claims citing 1–4 earlier inputs, `entity/*`
  claims distilled from derivations, `relation/*` claims joining 2–3 entities with
  `relation_direction`, periodic `contribution/head` claims consolidating open
  claims, and a `contribution/branches` table as the archive head.
- Claims only ever reference **earlier** claims, so the graph is acyclic and
  `created_at` is monotonic, as the ADT requires.
- Derivation inputs are drawn mostly from a recent window → the clustering a real
  archive has. Entity references use damped preferential attachment → hubs that
  do not run away.
- Ids are synthetic but multibase-base32 shaped and 57 chars long, so string keys
  cost what real claim ids will cost.
- Deterministic: same `--seed`, same archive.

The mix holds at every scale: 44 % derivations, 20 % sources, 20 % entities,
14 % relations, 1.5 % contribution claims — and ~3.4 edges per claim.

## Results

`node v24.18.0`, AMD Ryzen 7 PRO 5850U, 16 cores, Linux x64. Raw:
[`results/graph-bench.json`](results/graph-bench.json),
[`results/graph-bench-300k.json`](results/graph-bench-300k.json).
Heap deltas under ~10 MiB are inside the allocator's noise floor — read the 100k row.

### Getting the data in

| claims | edges | JSON wire | JSON.parse | build → graphology | graph on heap | claims on heap |
|---:|---:|---:|---:|---:|---:|---:|
| 1 000 | 3 339 | 0.5 MiB | 2 ms | 7 ms | 1 MiB | 1 MiB |
| 10 000 | 34 161 | 5.2 MiB | 14 ms | 119 ms | 14 MiB | 10 MiB |
| 100 000 | 342 632 | 52.4 MiB | 206 ms | 1 294 ms | 125 MiB | 96 MiB |
| 300 000 | 1 028 550 | 157.3 MiB | 716 ms | 4 435 ms | 369 MiB | 289 MiB |

Above 10k these costs scale linearly, at **~1.3 KiB per node** in graphology and
**~1 KiB per claim** as objects. Carrying inspector metadata on every node
(`full` vs `lean` attributes) costs only ~5 % more heap — display attributes are
not the problem. Building the graph costs **~6× more than parsing the payload**,
so the client's pre-paint bill is dominated by graphology, not by the wire.

### Laying it out

| claims | random | circular | circlepack | ForceAtlas2 (Barnes-Hut) | 200 iterations |
|---:|---:|---:|---:|---:|---:|
| 1 000 | 1 ms | 0 ms | 12 ms | ~7 ms/iter | 1.4 s |
| 10 000 | 4 ms | 4 ms | 111 ms | ~80 ms/iter | 16 s |
| 100 000 | 21 ms | 17 ms | 3 194 ms | ~1 300 ms/iter | 4.4 min |
| 300 000 | 206 ms | 25 ms | 21 549 ms | ~6 700 ms/iter | 22 min |

Single runs, and ForceAtlas2 varies ±20 % between them, so the figures are given
to two significant figures; the raw JSON keeps what each run actually measured.

- **ForceAtlas2 is the wall.** Cost grows ~11–16× per 10× nodes, so it is worse
  than linear in practice. At 100k a single iteration misses a 60 fps frame budget
  by ~80×, and a converged layout is minutes. Running it in a worker (the
  explorer's `fa2-worker` option) moves that cost off the main thread but does not
  make it finish sooner.
- **Barnes-Hut is not optional**: at 10k, disabling it costs ~610 ms/iter against
  ~80 — a 7.5× penalty that grows with N.
- **Seeding barely matters**: seeded from circlepack instead of random positions,
  100k measured ~920 ms/iter against ~1 300 — same order, no rescue.
- **circlepack** looks attractive (it groups by node class) but costs 3.2 s at
  100k and 21.5 s at 300k. It is affordable up to ~10k.
- **circular and random** are effectively free at every scale and are the only
  client-side layouts that survive 100k — at the cost of showing no structure.

### Heap headroom is a first-order cost

The 300k measurements were taken twice. Under Node's default ceiling on this
machine (**1 120 MiB**), ForceAtlas2 measured **132 s per iteration** and the
process then died with `Ineffective mark-compacts near heap limit`. With the
ceiling raised to 4 GiB, the same iteration on the same graph took **6.7 s** — a
**20× penalty purely from garbage-collection pressure**, not from the algorithm.

That is the most transferable finding here, because a browser tab's heap ceiling
sits in exactly that region: 300k claims need ~660 MiB (claims plus graph) before
layout allocates anything, so a 300k explorer would be running where the cliff is,
and would report layout costs an order of magnitude worse than the algorithm's.
Peak heap is the thing to design against, ahead of iteration count.

### The contributor star

Every claim carries a `contribution/contributor` edge, so a contributor claim's
degree equals the number of claims it signed. With 4 contributors over 100k
claims, the top node has **degree 25 235** — a quarter of the graph attached to
one point:

| claims | edges (all) | max degree | edges (contributor edges dropped) | max degree |
|---:|---:|---:|---:|---:|
| 1 000 | 3 339 | 266 | 2 343 | 33 |
| 10 000 | 34 161 | 2 558 | 24 180 | 164 |
| 100 000 | 342 632 | 25 235 | 242 668 | 479 |

Dropping those edges removes **29 % of all edges** and collapses max degree by
~50×. It buys little layout time (~1 070 against ~1 320 ms/iter at 100k, barely
outside run-to-run variance), so the reason to do it is legibility: a star over every claim in the
archive tells a reader nothing and drags every cluster towards one point. This is
a rendering decision — the edge stays in the data, where it is load-bearing for
verification.

### Client footprint

The whole renderer stack — graphology, Sigma, the layouts, the generator —
bundles to **197 KB (51 KB gzipped)**.

## What this implies for an explorer

1. **Do not lay out 100k claims in the browser.** Either the server returns
   coordinates alongside claims, or the client shows an overview positioned by a
   free layout (circular/random) and only force-lays-out a drill-down.
2. **Force layout belongs to the query result, not the archive.** A bounded read
   (the spec's filtered query with a result limit) lands in the thousands, where
   ForceAtlas2 costs 7–80 ms per iteration and a 200-iteration settle is 1.4–16 s —
   slow, but usable, and it can settle visibly in a worker. That is the interactive
   case, and it is the one to build for.
3. **Budget ~1.5 s and ~220 MiB for a 100k read** before layout or paint: 0.2 s to
   parse a 52 MiB payload, 1.3 s to build the graph, 96 MiB of claim objects and
   125 MiB of graph. Dropping the claim objects once the graph is built nearly
   halves the peak — and per the GC cliff above, peak heap is what to defend.
4. **Never draw `contribution/contributor` edges by default.** 29 % of edges, and
   they actively harm the picture.
5. **A 100k overview needs a coordinate source.** The obvious candidates are
   server-side layout (positions as a derived artifact) or an incremental
   client-side layout that only ever moves the newly arrived subgraph. Both are
   the next thing to spike.

## Open questions this spike did not answer

- Sigma's actual frame times at 100k on a real GPU, with and without edges, labels
  and `hideEdgesOnMove`. The harness is written and scripted; it needs hardware.
  Run `npm run bench:render` and commit `results/render-bench.json`. VRAM is not
  instrumented — that needs `chrome://gpu` or a Chrome trace, by hand.
- Whether 300k survives a browser tab at all — 369 MiB of graph plus 289 MiB of
  claims plus WebGL buffers, against a tab ceiling in the same region, and the GC
  cliff above shows what happens when you get close.
- Whether progressive loading (render at 10k, keep streaming) beats waiting for a
  complete result — a UX question the numbers above make worth asking.
- Whether server-side coordinates are stable enough across contributions to be
  cacheable, or whether every new head reshuffles the picture.

## Files

| Path | What |
|---|---|
| `src/mock/model.ts` | The ADT vocabulary the mock data follows |
| `src/mock/generate.ts` | Deterministic Ranke-shaped archive generator |
| `src/mock/graph.ts` | Claims → graphology, degree statistics, edge profiles |
| `src/bench/graph-bench.ts` | The headless bench (the numbers above) |
| `src/bench/browser-bench.ts` | Playwright driver for the render bench; skips cleanly |
| `src/explorer/main.ts` | The explorer and its in-page scripted measurement |
| `index.html`, `src/explorer/style.css` | The page |
| `results/*.json` | Measured output, committed on purpose |
