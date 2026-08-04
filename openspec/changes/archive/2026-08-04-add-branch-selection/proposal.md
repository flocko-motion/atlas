## Why

The Ranke Explorer cannot name a branch. `core/query.ts:26` hardcodes
`branch: 'main'` as the default and nothing offers an alternative, because until
`restructure-rest-routes` there was no route that lists branches — the same gap the
server had, mirrored in its first client.

Meanwhile the explorer already holds everything a branch view needs. `store.ts` states
the model outright: *"One store holds the union; a view is a predicate over it, not a
graph, so switching views swaps reducers."* `ViewState` already carries predicates —
`contributionRange`, `classes` — and the mock generator already produces an archive with
several named branches (`mock/model.ts:64`, `branches: Record<string, ClaimId>`). What is
missing is a branch predicate and a way to pick one.

So an archive with several branches currently renders as one undifferentiated union, and
a user cannot ask the question the data model is built around: what does *this* branch
say.

## What Changes

- Grow the data-source port with **`branches()`**, answered by both backends. `RestSource`
  reads `GET /branches`; `MockSource` returns what the generator already recorded. The
  port's own principle — *"one data path, not a real and a test one"* — keeps the picker
  working with no server, which is how the explorer is developed.
- Add a **scope predicate to `ViewState`**: a view admits the closure of one selected
  scope's head, or everything loaded when unset. Selecting swaps a reducer; it does not
  refetch and does not mutate the graph.
- Offer as scopes the **whole archive and each named branch** — everything with a head.
  `$universe` is excluded: it has none, which is why RankeQL demands an explicit head to
  read there. It stays a by-id read, not a scope a view can sit in.
- Compute membership as **reachability from that head**. A claim may sit in several
  branches, so membership is a test, not a label the claim carries.
- Put a **scope picker in the header's tools slot**, listing the archive and what
  `branches()` returned, with the current head shown per entry.
- **Delete the hardcoded default.** `branch: 'main'` is a guess about someone else's
  archive, wrong whenever it is wrong and silently so. Every branch name the explorer
  uses comes from the listing, so discovery precedes any branch-scoped read. One branch
  may be selected outright; several require a choice.

## Capabilities

### New Capabilities

- `graph-explorer`: the branch surface — discovery, selection, and a view confined to one
  branch's closure. Narrow by intent: the explorer has shipped without a capability, and
  this specifies the branch behaviour rather than retro-specifying the whole client.

## Impact

- **`frontend/src/core/data/source.ts`**: the `DataSource` port gains `branches()`; both
  backends implement it.
- **`frontend/src/core/store.ts`**: `ViewState` gains the branch predicate.
- **`frontend/src/core/graph/`**: closure-from-head, cached per branch head.
- **`frontend/src/ui/shell/Header.tsx`**: the picker.
- **`frontend/src/core/query.ts`**: `branch` stops defaulting to a guess and follows the
  selection.
- **Depends on `restructure-rest-routes`**, which adds `GET /branches` and moves
  `headOf`'s path. Without it there is nothing to populate the picker from.
- **Not affected**: the server. This is a client consuming a contract that change defines.
