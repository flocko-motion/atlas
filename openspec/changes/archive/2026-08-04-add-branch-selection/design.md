## Context

The explorer holds one graphology instance — the union of everything loaded — and
expresses each main-pane tab as a predicate over it. `store.ts` is explicit: *"a view is
a predicate over it, not a graph, so switching views swaps reducers."* `ViewState`
already carries `contributionRange` and `classes` in exactly that spirit.

It has no notion of a branch. `query.ts:26` defaults to `branch: 'main'`, a guess, and
`RestSource` exposes only `headOf`; `fetch()` still throws `NotWiredError`.

## Goals / Non-Goals

**Goals:**

- Discover the branches an archive holds.
- Confine a view to one branch.

**Non-Goals:**

- **Retro-specifying the explorer.** The capability covers the branch surface. What the
  rest of the client does is already built and out of scope here.
- **Multi-branch comparison.** Showing two branches side by side is a second tab, which
  the view model already gives for free. No new mechanism.
- **Wiring `fetch()`.** Reading claims from a server is a separate piece of work; the
  picker and the predicate work against whatever is loaded, mock or REST.

## Decisions

### A branch is a view predicate, not a fetch

Selecting a branch swaps a reducer. It does not clear the store, refetch, or mutate the
graph — the same treatment `classes` and `contributionRange` already get.

This falls out of the store/view split rather than being a choice, and it buys the
behaviour a user expects: switching between branches is instant because both are already
loaded, and switching back does not re-download. It also keeps the invariant the
architecture rests on — the graph is written once, on load, and read many times.

### Only scopes with a head are browsable

Three scopes exist in the access model, and only two can be *browsed*. A branch has a
head; `$archive` has the archive head; `$universe` has none — it is the whole store,
unrooted. RankeQL says so structurally: `validateSelect` rejects a `$universe` read
without an explicit `Select.Head` (`ErrQueryNoHead`), because there is nothing to bound
it otherwise.

A view predicate is a closure test, and a closure needs a root. So the picker offers the
archive and the named branches, and `$universe` stays what it is: a by-id read for
reaching a claim the branch table does not, not a scope a view can sit in.

### Membership is reachability from the head — computed by the engine

*Revised during execution.* This section originally had the client walk the closure. It is
right that membership *is* reachability and wrong about who computes it: a closure walk is a
query, and queries belong to ranke-go. The explorer asks a scoped query for identities only
(`output.detail: id`) and intersects the answer with its cache. Measured cost of the walk it
no longer does: 398 ms at 100k claims, 1.7 s at 300k.

The model argument below stands unchanged — a claim reached from two branches belongs to
both, so membership is a set and never a label.


A branch *is* the closure of its head, and `$archive` the closure of the archive head.
So "is this claim in scope `s`" is a reachability question from `s`'s head, answered
against the union graph.

The alternative — tagging each claim with a branch as it loads — is wrong on the model.
A claim reached from two branches belongs to both, and content addressing means it is
*one* node in the store, not two. A single label would have to pick a winner. A set of
labels would be a hand-maintained copy of reachability that drifts the moment a
contribution lands.

Claims are immutable, so the closure of a given head never changes: compute it once per
head id and cache it. A branch advancing gives a new head, hence a new cache entry, and
the old one stays valid for any view still pinned to it.

### The port answers `branches()`, both backends

`source.ts` states the rule this follows: *"A mock's 'server details' are the generator's
parameters, so configuring either is the same act: one data path, not a real and a test
one."*

`MockSource` already has the answer — the generator records `branches: Record<string,
ClaimId>` (`mock/model.ts:64`). `RestSource` reads `GET /branches`, which returns the
same pairing of name and head. So the picker is one code path, and the explorer keeps
working against no server at all, which is how it is developed and demonstrated.

### An unloaded branch shows empty, and says so

The picker lists what the archive holds; the store holds what has been loaded. Those
differ, and a branch whose head is absent from the store would render as an empty view.

Silently blank is the failure worth avoiding. A selected branch whose head is not loaded
reports that rather than drawing nothing — the distinction between "this branch is empty"
and "this branch is not here yet" is the whole question a user is asking.

## Risks / Trade-offs

- **Closure traversal at scale** → a BFS over the union graph per branch head. It is
  linear in the closure and runs off the render path, and the cache makes it once per
  head. The benchmark harness already measures graph work at 300k claims, so the cost is
  observable rather than guessed at.
- **The picker can outrun the store** → the branch list comes from the source and the
  graph comes from what was loaded, so a branch can be listed but unloadable into view.
  Handled above by reporting rather than blanking, but it is a state the UI has to carry.
- **A head advancing invalidates nothing, which is the point** → an old closure stays
  correct for the head it was computed from. What it does mean is that a view pinned to a
  stale head keeps showing stale membership until refreshed, which is correct behaviour
  and may still surprise.
