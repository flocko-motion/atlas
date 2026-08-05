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

### Nothing here declares the model or the contract

Two dependencies own what this client used to restate, and the boundary is as strict as the
layering above:

| From | Comes | So this repo declares |
|---|---|---|
| `@flocko-motion/ranke` | `Claim`, `Edge`, the node classes, the content declaration, the sequence reader, the type-glob matcher, the RankeQL `Query` | no claim type, no vocabulary, no framing rule |
| `openapi/openapi.gen.ts` | every route, every request and response type | no path string, no wire field name |

What the explorer *does* hold is drawing state the ADT has no reason to define — the
contribution index a history layout sorts on, the branch a generated archive stamped, and a
display label. Those sit **beside** the library's claim (`core/claims.ts`), never inside a copy
of it, and a test scans the tree to keep the deleted duplication from growing back.

## Server connections

Configured in the client, several at a time, switchable — the auth kinds mirror the
server's adapters and `openapi.yaml`'s security schemes: no-auth, `X-API-Key`,
`Authorization: Bearer` (JWT), and `Authorization: Macaroon`. Secrets stay in memory
for the session unless you tick *remember*, which writes them to `localStorage` where
any script on the page can read them.

A connection is read through the client generated from `openapi.yaml`
(`src/core/data/openapi.gen.ts`), so every route, path and response type a read uses is
the contract's and this repo hand-writes none of them. Auth stays the client's: which
kinds an instance accepts follows from how it was configured, so the generated client is
handed the headers the kind calls for rather than asked to negotiate one.

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

The archive scope needs the branch-table head, which `GET /archive/info` reports — the only
route that does. Against an instance too old to answer it, or a subject without `R $archive`,
the picker offers the branches alone rather than a scope with no head.

A scope's answer can name claims this session never read — the archive advanced, or the
load was capped — so the count is reported next to the picker rather than the overlap being
drawn silently.

## Open: stretching x without stretching y

The timeline already spreads the visible strata across the whole height, so a uniform zoom
wastes half of what it does — stretching y achieves nothing and only shrinks the bands when
zooming out. What is wanted is x alone.

Sigma has one uniform `ratio` and no per-axis zoom, so this means rewriting x and leaving y
where it is. Two properties decide the design:

- **The switch point is fixed for a given archive and viewport.** The most x is ever
  compressed is the ratio at which the whole axis fits the width; below that, further
  zooming out has to become the classic uniform zoom, because there is nothing left to
  compress. That maximum is computed once at startup and again on resize — not per frame.
- **The cost is in the nodes loaded, not the nodes visible.** A stretch step rewrites x for
  every node and then pays an O(N) Sigma refresh, whether ten are on screen or ten thousand.
  So the gate is the loaded count. Measured refresh cost: **733 ms at 50k**, against a layout
  pass of 17 ms at 2k and ~100 ms at 100k — the refresh dominates by an order of magnitude.

### The lens, which makes the cost go away

Rather than gating the stretch by a threshold, change the size of the problem: while zoomed
in, draw only the claims in the window. Two modes for two uses — seeing the whole archive, or
navigating it with a lens.

Sigma is handed a small graph in lens mode, so every stretch step is O(window) rather than
O(union), and rewriting x per zoom step becomes affordable.

Nothing is rebuilt on the way back: the union lives at module scope and the lens only reads
it. What a naive switch would cost is Sigma's own work — uploading N nodes into its buffers
and re-indexing them — which is avoidable by giving each graph its own Sigma instance and
switching which canvas is shown. The union's buffers stay valid while lensing, so neither
direction pays an upload. The price is two WebGL contexts, the union's GPU memory resident
throughout (as it already was), and camera state kept in step so the switch does not jump.

The bar is frictionless, and two things are what meet it: the lens is built and uploaded
*before* the canvases swap, so no frame is ever shown empty; and both instances are handed
the same camera state at the moment of the swap, so the picture does not move. Neither is
expensive — the lens is small — but getting the order wrong is what a reader would see.

Two things to settle first:

- **It bends the one-graph invariant, honestly.** The union stays authoritative; the lens is a
  derived, disposable copy for rendering, discarded on exit. Ids are content addresses, so
  selection and hover survive the switch without mapping.
- **Edges leaving the window are counted, not followed.** Bringing the far end along as a
  stub was the first design, and measuring settled it: provenance reaches back arbitrarily far
  in time, so a 4% window of 100k claims pulled in **8,333 stubs against 3,994 real claims**.
  A lens two-thirds composed of placeholders is not a lens. Instead each claim carries how many
  of its references leave the view, so the signal sits on the claim that has them.

Measured, with a viewport showing 4% of the axis:

| union | lens | cut once | stretch step: lens | stretch step: union |
|---|---|---|---|---|
| 20,000 claims | 801 (4.0%) | 54 ms | **2.8 ms** | 91 ms |
| 100,000 claims | 3,994 (4.0%) | 351 ms | **16.9 ms** | 591 ms |

So a stretch step costs about a thirtieth of what it would over the union, which is what makes
x-only zooming affordable at all. The cut is paid once per window rather than per step, and
only when a pan leaves the margin the window was widened by.

A measured node-count threshold with uniform zoom above it remains the fallback if the lens
proves more than is wanted.

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

**The container is capped at 2 GiB**, and that cap is now the binding constraint rather than a
footnote: claims are built by the ADT library, so 100k of them are **770 MiB of objects** where
the old hand-rolled shape was 97 MiB. Two things follow. The granularity sweep runs at 30k
claims rather than 100k — five 100k archives in one process was OOM-killed at the third
granularity. And the separate 300k run is gone: its claims alone measured **2 301 MiB** (113 s
to build 300 002 claims and 1 129 121 edges), so it cannot complete here at all. The stale
`results/graph-bench-300k.json` was deleted rather than left looking current; a machine with
memory to spare can restore it with `make bench ARGS="--scales=300000 --granularities=skip"`.

**The frame numbers below were instead produced by a person running the page**, on
an integrated AMD Radeon (Renoir) through ANGLE / OpenGL ES 3.2 — a laptop-class
GPU, which is the realistic target. Their CPU-side figures also confirmed that the
headless numbers transfer: at 50k they measured build 723 ms and history layout 20 ms,
against 2 274 / 91 ms for 100k in this container at the time — linear, as expected, since
it is the same V8 doing the same work. (Their generate figure, 135 ms at 50k, is no longer
comparable to anything: the generator has since moved onto the ADT library's builder, which
is ~25× the work. Build and layout are unaffected — same topology, same graphology.)

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
is acyclic and `created_at` monotonic, as the ADT requires.

**The claims are real claims.** They are built by the ADT library's `claim_builder`, which
means the ids are the content addresses of the canonical encoding — `id = H(S(v))`, 56 chars of
multibase base32, identity-signed since no key is held here (§5.7) — the edge rules are
enforced as they would be on a server, and the type a generated claim has is the type a read
returns. That costs an encode and a SHA-256 per record, which is why generation is ~0.4 ms per
claim rather than ~0.01 ms: see the table below, and the note on the default archive size.
Deterministic: same `--seed`, same archive.

Content is the one thing a generator cannot invent: it declares an external content address
and a size, so the sizes are realistic, but there are no bytes behind them — reading a
generated claim's content says so rather than returning something made up.

**Claims per contribution is swept, not assumed.** It is a usage pattern nobody can
know in advance, and it sets the archive's height — so the bench varies it and
reports cost as a curve. 2 % of contributions are bulk ingests 25× the usual size,
which is why the achieved mean sits above the nominal figure.

## Results

`node v24.18.0`, AMD Ryzen 7 PRO 5850U, 16 cores, 3 GiB heap ceiling under a 2 GiB container
cap. Single runs; ForceAtlas2 varies ±20 % between them, so it is quoted to two significant
figures — and, as the granularity table shows, it varies far more than that with what the heap
already holds. Raw: [`results/graph-bench.json`](results/graph-bench.json).

### Getting the data in

| claims | edges | generate | JSON of the claims | JSON.parse | build → graphology | graph on heap | claims on heap |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 1 000 | 3 787 | 0.3 s | 1.1 MiB | 8 ms | 19 ms | 1 MiB | 8 MiB |
| 10 000 | 37 709 | 3.3 s | 11.4 MiB | 72 ms | 200 ms | 12 MiB | 77 MiB |
| 100 000 | 377 121 | 30.3 s | 114.3 MiB | 1 030 ms | 2 778 ms | 111 MiB | **770 MiB** |

**~1.1 KiB per node** in graphology, and **~7.9 KiB per claim** as objects. That second
figure is the one that changed: a claim is now the ADT library's `Claim`, which carries the
split type, both timestamp forms, a content declaration and frozen edge records, where the old
hand-rolled shape carried the four fields a renderer read. It is ~8× the heap, and at 100k it
is 770 MiB sitting *beside* the graph for the whole session.

Building the graph still costs ~2.7× parsing a payload of the same claims, so the pre-paint
bill is graphology's rather than the wire's. Inspector metadata on every node (`full` vs `lean`
attributes) adds ~7 % of heap and ~28 % of build time — more than it used to look, still not
the problem.

The JSON column is the *decoded* claims stringified, which bounds a response from above rather
than being one: a wire record carries `type` once, not split, and no epoch milliseconds.

### Shape: the two candidate axes

The archive is tall, but not in the way the DAG suggests. Provenance depth and
contribution order disagree, and the disagreement decides the layout:

| claims | contributions | height (depth) | widest depth layer | rows (history) | widest row | mean row |
|---:|---:|---:|---:|---:|---:|---:|
| 1 000 | 55 | 61 | 186 | 55 | 246 | 18.2 |
| 10 000 | 709 | 715 | 1 763 | 709 | 286 | 14.1 |
| 100 000 | 6 814 | 6 820 | **17 710** | 6 814 | **301** | 14.7 |

**Provenance depth is a bad axis.** Height tracks the contribution count, as
expected — but only the head/branch-table spine climbs it. A source claim citing
just its contributor sits at depth 1 however old the archive is, and a derivation
citing recent sources at depth 2–3, so 17 710 claims share one depth layer at 100k.
`y = depth` draws a tall thin spine beside a flat mat of everything else.

**Contribution order is the axis.** Same graph: 6 814 rows averaging 14.7 claims,
widest 301 — a 23:1 ribbon, one row per contribution, which is also what a reader
means by "when". It needs one number per claim (which contribution added it), and
that is why `contribution` is in the lean attribute profile: it is an axis, not
metadata. It is also the one number a *read* cannot supply — an archive expresses that
order as the head chain and no output field reports it — so it is drawing state the
explorer carries beside the claim rather than a field of one.

### Laying it out

| claims | by history | by depth | random | circular | circlepack | ForceAtlas2 (Barnes-Hut) | 200 iterations |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 1 000 | 1 ms | 2 ms | 1 ms | <1 ms | 16 ms | ~12 ms/iter | 2.5 s |
| 10 000 | 6 ms | 5 ms | 2 ms | 2 ms | 148 ms | ~140 ms/iter | 27 s |
| 100 000 | **110 ms** | 104 ms | 20 ms | 15 ms | 5 034 ms | ~2 600 ms/iter | 8.8 min |

- **The history axis is ~4 800× cheaper than a settled force layout at 100k**
  (110 ms against 527 s) and it is deterministic — same archive, same picture, every
  time, which a force layout never gives you.
- **ForceAtlas2 is the wall.** ~11× per 10× nodes to 10k, ~19× on the next decade. At
  100k one iteration misses a 60 fps frame budget by ~160×. A worker (`make run` →
  *ForceAtlas2 (worker)*) moves the cost off the main thread without making it finish.
- **Barnes-Hut is not optional**: at 10k, off costs ~1 050 ms/iter against ~140.
- **Seeding barely matters**: from circlepack rather than random, 100k measured
  ~2 100 ms/iter against ~2 600 — same order, no rescue.
- **circlepack** groups by class attractively but costs 5.0 s at 100k. Fine to ~10k.
- **depth pass**: computing every claim's depth is one sweep, 368 ms at 100k — an
  O(V+E) property, which is why the layered layout is nearly free.

The force numbers are **~1.7× higher than the previous run** of the same topology, and the
likeliest reason is the row above: 770 MiB of claim objects are live while the layout runs, so
every iteration allocates against a heap eight times fuller than before. A shared host under
memory pressure is the other candidate, and this container cannot tell the two apart — which is
itself the argument for not holding decoded claims once they are in the graph.

### Granularity: the parameter nobody can know

Same 30 000 claims, varying claims per contribution. (30k rather than 100k: five archives of
100k library-built claims in one process do not fit the container — see above.)

| claims/contribution | contributions | bookkeeping share | height | widest depth layer | rows | widest row | edges | FA2 |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| ~3 | 8 441 | **56 %** | 8 443 | 2 692 | 8 441 | 39 | 94 321 | ~5 200 ms/iter † |
| ~10 | 2 031 | 14 % | 2 037 | 5 321 | 2 031 | 300 | 113 350 | ~480 ms/iter |
| ~30 | 710 | 5 % | 719 | 5 862 | 710 | 968 | 116 975 | ~510 ms/iter |
| ~100 | 190 | 1 % | 203 | 6 076 | 190 | 3 515 | 118 765 | ~440 ms/iter |
| ~1000 | 11 | 0 % | 30 | 6 149 | 11 | 20 388 | 118 972 | ~520 ms/iter |

Three things fall out, and the first is not about rendering at all:

1. **Fine-grained contributions make the archive mostly bookkeeping.** At ~3 claims
   per contribution, **56 % of all claims are heads and branch tables** — the archive
   is more history than content. At ~30 it is 5 %, at ~100 it is 1 %. That is a cost
   the server pays too (storage, closure walks, sequencer revisions), not just a
   picture problem.
2. **Layout cost is indifferent to granularity** (440–520 ms/iter across a 300× range, reading
   the † row for what it is). Edge count drives ForceAtlas2, and the edge count barely moves;
   height does not drive it at all. A clean process measures the same five archives at
   443–721 ms/iter — same conclusion, same order.

   † The first sweep row reads **5 200 ms/iter**, and that figure is about the *run* rather
   than the granularity: the sweep starts immediately after the 100k scale row, so its first
   ForceAtlas2 probe pays for collecting that row's 770 MiB. The same archive in a clean
   process is **721 ms/iter**. It is left in the table because it is what a full `make bench`
   measures, and because it is the sharpest illustration of the heap lesson going: the same
   layout over the same graph, ~7× slower for what was allocated before it started.
3. **The history axis degrades gracefully into uselessness.** At ~1000 claims per
   contribution there are 11 rows of ~2 700 — no longer a ribbon, and a renderer
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

The whole stack — React, graphology, Sigma, the layouts, the ADT library, the generated REST
client, the generator — bundles to **489 KB (140 KB gzipped)**, or one 489 KB self-contained
`explorer.html`. The two libraries this client stopped hand-writing cost ~34 KB of that, for
the claim model, the codec, the query type and every route.

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
- Whether 300k survives a browser tab. It looks worse than it did: the claims alone are
  **2 301 MiB** now that they are real claims, which is over half a tab's usual 4 GiB
  ceiling before the graph exists. An earlier run on a 1 120 MiB ceiling measured
  ForceAtlas2 at **132 s/iteration** and then died with `Ineffective mark-compacts near
  heap limit`; with 4 GiB the same graph took **6.7 s/iteration** — a 20× penalty purely
  from GC pressure. Those figures predate both the contribution chains and library-built
  claims, so they are indicative only, but the lesson is not: **peak heap is a
  first-order performance factor**, and a browser tab's ceiling sits right there. The
  first thing to try is not holding the decoded claims at all — merge each into the graph
  as it arrives and drop it, which the streaming reader already makes possible.
- **Labels when zoomed in.** They cost nothing at 50k zoomed out because Sigma culls
  almost all of them; the same camera path zoomed into a dense contribution row, where
  many nodes clear `labelRenderedSizeThreshold`, is the case that would actually price
  label drawing.
- Whether rows should be contributions or time buckets once contributions get large.
- Whether progressive rendering (draw at 10k, keep streaming) beats waiting.

## Files

| Path | What |
|---|---|
| `Makefile` | The entry point — `run`, `build`, `single`, `bench`, `test`, `check` |
| `explorer.html` | Generated, committed: open it with no npm and no server |
| `src/main.tsx` | Mounts React, which renders the shell |
| `src/core/connections.ts` | The instances this client can talk to, and their credentials |
| `src/core/data/source.ts` | The data-source port and its two backends: a connection, and the generator |
| `src/core/data/openapi.gen.ts` | Generated, committed: the REST client, from the root `openapi.yaml` |
| `src/core/claims.ts` | The library's claim, plus the drawing state it has no reason to define |
| `src/core/mock/model.ts` | The subtypes a generator invents, and the shape of a generated archive |
| `src/core/mock/generate.ts` | Contribution-by-contribution archive generator |
| `src/core/graph/` | Claims → graphology: the union, the lens, scope membership, depth and degree statistics |
| `src/core/layout/` | The layouts, and the timescale the history layout stretches |
| `src/core/store.ts`, `session.ts` | View state, and the actions a read or a scope change runs |
| `src/render/renderer.ts` | The one Sigma instance and the reducers a view is expressed through |
| `src/ui/` | The shell, its panes, and the header |
| `src/bench/graph-bench.ts` | The headless bench (every number above) |
| `src/bench/browser-bench.ts` | Playwright driver for frame timings; skips cleanly |
| `src/**/*.test.ts` | The headless core tests — `make test` |
| `results/*.json` | Measured output, committed on purpose |
