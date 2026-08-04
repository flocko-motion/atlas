# Ranke Explorer

A pure browser client for a RankeDB archive: a static bundle with no application
server, no proxy and no database of its own. It talks straight to a ranke-db REST
endpoint, can hold several instances at once, and works with none at all against
generated mock data.

```sh
make run       # dev server — open the URL, then press 'load' in the header
make           # build dist/ and the self-contained explorer.html
make bench     # the headless performance numbers below
make help      # the rest
```

Dependencies install themselves on first use, so a clean checkout needs only node.

`explorer.html` is committed and self-contained — open it with no npm and no server.

## Architecture

Strict one-way layering; the boundary is the point, not a preference.

| Layer | Holds | Never |
|---|---|---|
| `src/core/` | the store (zustand), the union graph, layouts, shape statistics, mock data, connections | React, DOM, Sigma |
| `src/render/` | the one Sigma instance, at module scope, and the reducers a view is expressed through | graph mutation on a view switch |
| `src/ui/` | React components that read state and dispatch actions | graph data in state, context or props |

Two rules follow from measurement rather than taste:

- **One store holds the union; views are selections over it.** A claim's id is its
  content address, so a claim reached by two queries merges to one node with no
  reconciliation. Switching a view is a reducer swap, never a rebuild.
- **The renderer lives outside React.** Sigma is constructed once for the canvas host
  and is never torn down by a re-render.

## Server connections

Configured in the client, several at a time, switchable — the auth kinds mirror the
server's adapters and `openapi.yaml`'s security schemes: no-auth, `X-API-Key`,
`Authorization: Bearer` (JWT), and `Authorization: Macaroon`. Secrets stay in memory
for the session unless you tick *remember*, which writes them to `localStorage` where
any script on the page can read them. Reading claim *bodies* from a connection is not
wired yet: the REST query contract has merged (`rest-api`), but nothing here imports a
generated client from it so far.

## Branches, and where a query is answered

The header's picker lists what the archive holds — the whole archive and each named
branch, each with the head its closure is rooted at — read through the data-source port,
so it works against a server (`GET /branches`) and against mock data alike. `$universe`
is absent: a browsable scope needs a head, and it has none, which is also why RankeQL
refuses to read there without an explicit one. No branch name is built in; `main` is a
fact about someone else's archive, so every name comes from the listing.

**The engine answers what a branch contains, not this client.** Selecting a scope runs a
query for identities only — `select.branch` names the scope, `output.detail: id` asks for
the ids — and the view then draws the intersection with what is cached. Switching scopes
therefore costs a query for ids and a `Set.has` per claim, never a re-read of a claim body.

An earlier attempt walked the closure here instead, which is a second query engine in the
layer least able to optimise one. It also cost **398 ms at 100k claims and 1.7 s at 300k**
for the widest scope. Both are reasons; the boundary is the better one.

A scope's answer can name claims this session never read — the archive advanced, or the
load was capped — so the count is reported next to the picker rather than the overlap being
drawn silently.

# Performance

## What was measured, and what was not

| Leg | Measured | Where |
|---|---|---|
| Generate mock claims, build the graph, heap, wire size | ✅ | this container, headless |
| Height, contribution shape, all five layouts | ✅ | this container, headless |
| Sigma first paint, camera frame times, full redraw | ✅ | **a human's machine** — see below |
| VRAM | ❌ | not instrumented — needs `chrome://gpu` or a trace, by hand |

The headless numbers come from `make bench` in a container with no GPU and no
browser: `chrome-headless-shell` is missing 17 system libraries here (`libnspr4`,
`libnss3`, `libX11`, `libgbm`, …) and there is no root to install them, so
`make bench-render` **skips loudly and exits 0** — the convention the repo's adapter
tests use for podman-backed counterparts. A software rasteriser would have measured
SwiftShader rather than a GPU, so nothing was faked in its place.

**The frame numbers below were instead produced by a person running the page**, on
an integrated AMD Radeon (Renoir) through ANGLE / OpenGL ES 3.2 — a laptop-class
GPU, which is the realistic target. Their CPU-side figures also confirm that the
headless numbers transfer: at 50k they measured generate 135 ms, build 723 ms,
history layout 20 ms, against 584 / 2 274 / 91 ms for 100k here — linear, as
expected, since it is the same V8 doing the same work.

**The render code was written without ever executing it**, and that showed. Two bugs
came out of it: Sigma v3 renders its first frame *synchronously inside the
constructor* (found by reading Sigma's source — awaiting an `afterRender` listener
attached afterwards waits forever), and a claim's ADT type was being stored in the
node attribute `type`, which Sigma reads as the name of the WebGL program to render
with — that one killed the first real run outright. Anything here still unexercised
should be assumed broken until a person has clicked it.

## The mock archive

Built **contribution by contribution**, because that is what gives a Ranke-Graph
its shape. Each contribution adds content claims, then a `contribution/head`
consolidating the previous head plus what is new, then a `contribution/branches`
revision carrying a `contribution/diff` to the table before it. Those two chains
are the commit history, and they are what makes the graph tall.

Content within a contribution is ordinary Ranke: `source/*` roots, `derivation/*`
citing recent inputs (locality → clusters), `entity/*` distilled from derivations,
`relation/*` joining entities with `relation_direction`, everything carrying a
`contribution/contributor` edge. Claims only reference *earlier* claims, so the DAG
is acyclic and `created_at` monotonic, as the ADT requires. Ids are synthetic but
multibase-base32 shaped and 57 chars long, so string keys cost what real ids cost.
Deterministic: same `--seed`, same archive.

**Claims per contribution is swept, not assumed.** It is a usage pattern nobody can
know in advance, and it sets the archive's height — so the bench varies it and
reports cost as a curve. 2 % of contributions are bulk ingests 25× the usual size,
which is why the achieved mean sits above the nominal figure.

## Results

`node v24.18.0`, AMD Ryzen 7 PRO 5850U, 16 cores, 4 GiB heap ceiling. Single runs;
ForceAtlas2 varies ±20 % between them, so it is quoted to two significant figures.
Raw: [`results/graph-bench.json`](results/graph-bench.json).

### Getting the data in

| claims | edges | JSON wire | JSON.parse | build → graphology | graph on heap | claims on heap |
|---:|---:|---:|---:|---:|---:|---:|
| 1 000 | 3 775 | 0.6 MiB | 2 ms | 28 ms | 1 MiB | 1 MiB |
| 10 000 | 37 549 | 5.7 MiB | 22 ms | 208 ms | 15 MiB | 10 MiB |
| 100 000 | 376 634 | 57.2 MiB | 288 ms | 2 274 ms | 138 MiB | 97 MiB |

**~1.4 KiB per node** in graphology, **~1 KiB per claim** as objects. Building the
graph costs ~8× more than parsing the payload, so the pre-paint bill is graphology's,
not the wire's. Inspector metadata on every node (`full` vs `lean` attributes) adds
only ~2 % — display attributes are not the problem.

### Shape: the two candidate axes

The archive is tall, but not in the way the DAG suggests. Provenance depth and
contribution order disagree, and the disagreement decides the layout:

| claims | contributions | height (depth) | widest depth layer | rows (history) | widest row | mean row |
|---:|---:|---:|---:|---:|---:|---:|
| 1 000 | 55 | 60 | 185 | 55 | 197 | 18.2 |
| 10 000 | 709 | 714 | 1 763 | 709 | 296 | 14.1 |
| 100 000 | 6 937 | 6 942 | **17 659** | 6 937 | **299** | 14.4 |

**Provenance depth is a bad axis.** Height tracks the contribution count, as
expected — but only the head/branch-table spine climbs it. A source claim citing
just its contributor sits at depth 1 however old the archive is, and a derivation
citing recent sources at depth 2–3, so 17 659 claims share one depth layer at 100k.
`y = depth` draws a tall thin spine beside a flat mat of everything else.

**Contribution order is the axis.** Same graph: 6 937 rows averaging 14 claims,
widest 299 — a 23:1 ribbon, one row per contribution, which is also what a reader
means by "when". It needs one number per claim (which contribution added it), and
that is why `contribution` is in the lean attribute profile: it is an axis, not
metadata.

### Laying it out

| claims | by history | by depth | random | circular | circlepack | ForceAtlas2 (Barnes-Hut) | 200 iterations |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 1 000 | 2 ms | 3 ms | 2 ms | 1 ms | 22 ms | ~13 ms/iter | 2.5 s |
| 10 000 | 11 ms | 8 ms | 2 ms | 2 ms | 153 ms | ~130 ms/iter | 26 s |
| 100 000 | **91 ms** | 83 ms | 9 ms | 8 ms | 3 916 ms | ~1 500 ms/iter | 5.0 min |

- **The history axis is ~16 000× cheaper than a settled force layout at 100k**
  (91 ms against 300 s) and it is deterministic — same archive, same picture, every
  time, which a force layout never gives you.
- **ForceAtlas2 is the wall.** ~11× per 10× nodes. At 100k one iteration misses a
  60 fps frame budget by ~90×. A worker (`make run` → *ForceAtlas2 (worker)*) moves
  the cost off the main thread without making it finish.
- **Barnes-Hut is not optional**: at 10k, off costs ~820 ms/iter against ~130.
- **Seeding barely matters**: from circlepack rather than random, 100k measured
  ~1 010 ms/iter against ~1 500 — same order, no rescue.
- **circlepack** groups by class attractively but costs 3.9 s at 100k. Fine to ~10k.
- **depth pass**: computing every claim's depth is one sweep, 266 ms at 100k — an
  O(V+E) property, which is why the layered layout is nearly free.

### Granularity: the parameter nobody can know

Same 100k claims, varying claims per contribution:

| claims/contribution | contributions | bookkeeping share | height | widest depth layer | rows | widest row | FA2 |
|---:|---:|---:|---:|---:|---:|---:|---:|
| ~3 | 28 952 | **58 %** | 28 954 | 8 633 | 28 952 | 39 | ~1 360 ms/iter |
| ~10 | 6 937 | 14 % | 6 942 | 17 659 | 6 937 | 299 | ~1 410 ms/iter |
| ~30 | 2 368 | 5 % | 2 376 | 19 532 | 2 368 | 1 031 | ~1 480 ms/iter |
| ~100 | 731 | 1 % | 742 | 20 204 | 731 | 3 630 | ~1 530 ms/iter |
| ~1000 | 39 | 0 % | 56 | 20 487 | 39 | 36 952 | ~1 460 ms/iter |

Three things fall out, and the first is not about rendering at all:

1. **Fine-grained contributions make the archive mostly bookkeeping.** At ~3 claims
   per contribution, **58 % of all claims are heads and branch tables** — the archive
   is more history than content. At ~30 it is 5 %, at ~100 it is 1 %. That is a cost
   the server pays too (storage, closure walks, sequencer revisions), not just a
   picture problem.
2. **Layout cost is indifferent to granularity** (1 360–1 530 ms/iter across a
   700× range). Edge count drives ForceAtlas2; height does not.
3. **The history axis degrades gracefully into uselessness.** At ~1000 claims per
   contribution there are 39 rows of ~2 500 — no longer a ribbon, and a renderer
   would need to lay out *within* a row. Below ~100 claims per contribution the
   axis works well; above that it needs a second dimension.

### Frame times on real hardware

Measured by a person driving the page, not by this container. Machine: integrated AMD
Radeon (Renoir) via ANGLE / OpenGL ES 3.2, Linux. Layout `history`, edges drawn during
motion (`hideEdgesOnMove` off), scripted camera path via **Run camera bench**.

| claims | edges | labels | edges while moving | first paint | camera | frame p50 | frame p95 | full redraw |
|---:|---:|---|---|---:|---:|---:|---:|---:|
| 50 000 | 188 114 | off | drawn | 1 005 ms | **6.8 fps** | 125 ms | 376 ms | 733 ms |
| 50 000 | 188 114 | on | drawn | 985 ms | — | — | — | — |
| 50 000 | 188 114 | off | **hidden** | 602 ms | **59.6 fps** | 16.7 ms | 16.9 ms | 560 ms |
| 50 000 | 188 114 | **on** | **hidden** | 781 ms | **58.1 fps** | 16.7 ms | 20.4 ms | 663 ms |
| 10 000 | 37 549 | on | drawn | — | ~20 fps | — | — | — |

The 10k figure is eyeballed from the live HUD rather than the scripted path, so treat
it as an order of magnitude. The first-paint and full-redraw differences between rows
one and two are machine warmth, not the mitigation — `hideEdgesOnMove` only affects
frames drawn *during* camera motion.

**Hiding edges while moving is the whole difference: 125 ms → 16.7 ms per frame, an
18× improvement that lands exactly on the 60 fps vsync cap.** p95 of 16.9 ms says the
renderer is idle-waiting for the display, so the frame budget is not merely met — it
has room to spare.

That room is the useful part, because hiding all edges **looks worse**: the picture
loses its structure exactly while you are moving through it. Taking the two rows as
two points, edges cost about (125 − 16.7) ms ÷ 188 114 ≈ **0.6 µs per edge drawn**.
Spending half a frame — 8 ms — on edges would then buy roughly **14 000 of them**,
about 7 % of the graph, held during motion instead of none. That is an extrapolation
from two measurements, not a measurement, and the obvious next experiment.

Three further things follow, and the first corrects a guess of mine:

1. **Labels are not the bottleneck; edges are — and labels are close to free.** I
   predicted the opposite, on the theory that Sigma draws labels as 2D canvas text
   every frame while nodes and edges go through WebGL. The controlled pair (rows three
   and four, identical but for labels) says otherwise: **58.1 fps with labels against
   59.6 without, the same 16.7 ms p50**, and only the p95 tail moves (20.4 against
   16.9 ms). First paint likewise barely notices them (985 against 1 005 ms).

   The likely reason is that Sigma's label culling is doing its job: labels only
   render above `labelRenderedSizeThreshold` (8 here) and the label grid caps their
   density, so at 50k zoomed out only a handful are ever drawn. That is a hypothesis
   about *why*, and it predicts labels get expensive when zoomed far enough in for
   many nodes to clear the threshold — which nobody has measured.
2. **First paint is ~1 s at half the target scale**, and a **full redraw is 733 ms** —
   so any data update that re-indexes the graph is a visible stall, independent of
   panning.
3. **The binary choice is avoidable.** Sigma offers all-edges or no-edges during
   motion; the first is 6.8 fps, the second loses the structure a reader navigates by.
   The frame budget says a middle exists — a few thousand edges held during motion,
   chosen by degree, zoom relevance, selection neighbourhood, or sampling.

   Better still, **gate edges on zoom level**: zoomed far out, 188k edges overlap into
   near-solid colour and carry no information, so hiding them there costs the reader
   nothing and buys the whole 18×. That also suggests the per-edge model above is too
   simple — a flooded view is fill-rate bound, so cost tracks the *pixels* edges cover,
   which makes long edges (the contributor star, spanning the entire picture)
   disproportionately expensive and worth dropping first.

   Note the implementation asymmetry: `hideEdgesOnMove` is an internal per-frame skip
   and free to toggle, whereas an `edgeReducer` marking edges `hidden` filters during
   indexation, so a zoom-threshold crossing would pay the full-refresh cost measured
   above (560–733 ms) unless a per-frame skip can be reached instead.

### The contributor star

Every claim carries a `contribution/contributor` edge, so a contributor's degree is
the number of claims it signed. With 4 contributors over 100k claims the top node
reaches **degree 25 120** — a quarter of the archive on one point — and it stays
~25k across the whole granularity sweep:

| claims | edges (all) | max degree | edges without contributor edges | max degree |
|---:|---:|---:|---:|---:|
| 1 000 | 3 775 | 268 | 2 779 | 198 |
| 10 000 | 37 549 | 2 549 | 27 551 | 297 |
| 100 000 | 376 634 | 25 120 | 275 636 | 494 |

Dropping them at render time removes **27 % of all edges** and cuts max degree ~50×.
It buys some layout time (~1 100 against ~1 500 ms/iter at 100k) but the real reason
is legibility: a star over every claim in the archive tells a reader nothing while
dragging every cluster towards one point. A rendering decision only — the edge stays
in the data, where verification needs it.

### Client footprint

The whole stack — graphology, Sigma, the layouts, the generator — bundles to
**198 KB (51 KB gzipped)**, or one 199 KB self-contained `explorer.html`.

## What this implies for an explorer

1. **Don't force-lay-out an archive.** Use contribution order: 91 ms at 100k,
   deterministic, and semantically right. Keep ForceAtlas2 for a drill-down of a
   few thousand claims, where it costs 13–130 ms/iter and settles in seconds.
2. **Budget edges, not layout.** Layout is solved at 21 ms; drawing 188k edges every
   frame is not, at 6.8 fps. Hide edges while the camera moves, decimate them by
   zoom level, or drop whole edge classes (starting with the contributor star, which
   is 27 % of them). This is where the next spike's effort belongs.
4. **Provenance depth is not the history axis**, even though it correlates with it.
   Measure before believing otherwise — I did believe otherwise.
5. **Budget ~2.5 s and ~235 MiB for a 100k read** before layout or paint: 0.3 s
   parse, 2.3 s build, 97 MiB claims, 138 MiB graph. Drop the claim objects once the
   graph is built. Add ~1 s of first paint on top, measured at 50k.
6. **Never draw `contribution/contributor` edges by default** — 27 % of edges,
   actively harmful to the picture, and now also implicated in the frame cost.
7. **Contribution granularity is a first-class design parameter**, and it belongs to
   whoever writes to the archive, not to the renderer. At the fine end the archive is
   mostly bookkeeping; at the coarse end the history axis stops separating anything.

## Open questions

- **Partial edges while moving.** Hiding all of them hits 60 fps; hiding none gives
  6.8 fps; ~0.6 µs per edge suggests ~14 000 could stay visible inside the budget.
  Nobody has measured a partial draw, and the interesting variants differ in what they
  keep: highest-degree edges, edges incident to the selection, edges within the rows
  currently on screen, or a flat random sample. Sigma has no setting for this — it
  needs an `edgeReducer` that hides by predicate, and re-measuring.
- **Dropping the contributor edges** (27 % of them) as a cheaper first cut, which
  should be worth ~1.6 of the 6.8 fps case on the per-edge figure — probably not
  enough alone, which is itself worth confirming.
- Frame times at 100k, and the same runs on a discrete GPU — though if the cost is
  edge upload and fill rate, an integrated GPU is the honest target anyway.
- Whether 300k survives a browser tab. An earlier run on a 1 120 MiB heap ceiling
  measured ForceAtlas2 at **132 s/iteration** and then died with `Ineffective
  mark-compacts near heap limit`; with 4 GiB the same graph took **6.7 s/iteration**
  — a 20× penalty purely from GC pressure. Those figures predate the contribution
  chains, so they are indicative only, but the lesson is not: **peak heap is a
  first-order performance factor**, and a browser tab's ceiling sits right there.
- **Labels when zoomed in.** They cost nothing at 50k zoomed out because Sigma culls
  almost all of them; the same camera path zoomed into a dense contribution row, where
  many nodes clear `labelRenderedSizeThreshold`, is the case that would actually price
  label drawing.
- Whether rows should be contributions or time buckets once contributions get large.
- Whether progressive rendering (draw at 10k, keep streaming) beats waiting.

## Files

| Path | What |
|---|---|
| `Makefile` | The entry point — `run`, `build`, `single`, `bench`, `check` |
| `explorer.html` | Generated, committed: open it with no npm and no server |
| `src/mock/model.ts` | The ADT vocabulary the mock data follows |
| `src/mock/generate.ts` | Contribution-by-contribution archive generator |
| `src/mock/graph.ts` | Claims → graphology; depth, history and degree statistics |
| `src/bench/graph-bench.ts` | The headless bench (every number above) |
| `src/bench/browser-bench.ts` | Playwright driver for frame timings; skips cleanly |
| `src/explorer/main.ts` | The explorer and its in-page scripted measurement |
| `results/*.json` | Measured output, committed on purpose |
