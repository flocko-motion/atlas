## Decision taken during execution: membership is the engine's answer, not the client's

The plan had the client compute membership — a BFS from the scope's head over the loaded
graph (3.2), cached per head (3.3). Built and measured, that costs **398 ms at 100k claims
and 1.7 s at 300k** for the widest scope, once per head.

The cost was not the reason to change it. The boundary was: a closure walk is a query, and
queries are ranke-go's. A client that walks closures is a second query engine, in the layer
furthest from the data and least able to optimise it.

So selection asks the source instead, with the query contract's ids-only result —
`select.branch` names the scope, `output.detail: id` asks for identities only. The engine
answers what a branch contains; the client intersects that id set with what it has cached
and draws the overlap. Switching scopes still costs no re-read of claim bodies, which is
what the view-predicate design was protecting.

Tasks 3.2, 3.3 and 3.4 are therefore **superseded**, and the tests that pinned a
client-side walk are replaced by tests of the answered id set. `design.md`'s "Membership is
reachability from the head, not a label" is right about the *model* and wrong about *who
computes it*.

## 1. Discovery through the port

- [x] 1.1 Add `branches()` to the `DataSource` port in `frontend/src/core/data/source.ts`, returning each branch by name and current head.
- [x] 1.2 Implement it on `RestSource` against `GET /branches`. Verified against a live instance: `branches()` → `main@b5uau5xqkv…`.
- [x] 1.3 Implement it on `MockSource` from what the generator already records — `branches: Record<string, ClaimId>`. One code path for both.
- [x] 1.4 Include the archive itself as a selectable scope, carrying the archive head. Excluding `$universe` needed no special case: the rule is *a scope is browsable iff it has a head*, and `$universe` has none — which is also why RankeQL rejects a read there without an explicit `Select.Head`. The REST backend cannot offer the archive scope either, since **no route reports the branch-table head** — see "Found while executing".

## 2. Delete the assumed branch name

- [x] 2.1 Remove `branch: 'main'` from `DEFAULT_QUERY`. It is `string | null`, defaulting to null.
- [x] 2.2 Hold no branch name until the listing answers: nothing scope-confined is issued before `discoverScopes()` returns, and the query pane reports "none selected" rather than a placeholder.
- [x] 2.3 Select automatically only when the listing returns exactly one branch. Several are left unset — no name, position or convention decides.

## 3. The scope predicate

- [x] 3.1 Add the selected scope to `ViewState`. Only the name and head live in the store; the ids it contains stay at module scope, since an id set of a large branch is graph data and graph data never enters React state.
- [~] 3.2 **Superseded.** Membership is not computed here. `scopeIds(scope)` asks the source, which answers with the query contract's `output.detail: id`.
- [~] 3.3 **Superseded.** There is no closure to cache. The answered id set is held per scope name and reused on reselection; a new answer replaces it.
- [x] 3.4 Apply it as a reducer swap in `render/`. No store clear, no refetch, no node or edge added or removed — the reducer does a `Set.has` against the answer.

## 4. The picker

- [x] 4.1 Scope picker in the header's tools slot, listing the archive and each branch with the head its closure is rooted at. What it offers is composed in `core/scope.ts`, so the rule lives in core and is testable without a browser.
- [x] 4.2 Report the shortfall: claims a scope's answer names that this session has not cached. That is the state a graph loaded once against a moving archive produces, and it is counted rather than silently absent.
- [x] 4.3 Carry the selection into the query as its scope.

## 5. Tests

- [x] 5.1 Branches list identically from a REST connection and from mock data, through the one port method.
- [x] 5.2 A scope narrows what is admitted while the cache keeps every claim — the branch answer is a proper subset of the archive answer.
- [x] 5.3 Switching between two answered scopes and back keeps both answers and re-asks nothing.
- [x] 5.4 A claim two scopes both contain is one claim in the cache, reported under either.
- [~] 5.5 **Replaced.** "Computed once per head" no longer applies. Pinned instead: a scope with no answer yet admits everything, so a view is never blank for want of one.
- [x] 5.6 Claims a scope names but the cache lacks are countable.
- [x] 5.7 The picker offers no `$universe` entry — and no backend returns a scope without a head.
- [x] 5.8 With no listing answered, no scope-confined read is issued and no branch name is assumed.
- [x] 5.9 Added: the ids-only wire shape is read correctly out of a JSON sequence, bare-string and object records alike, ignoring the execution report a query may append.

## Found while executing

- **`RestSource.headOf` returned the JSON envelope, not the id.** It read the body as text
  and trimmed it, so it answered `{"head":"b5ua…"}` where a caller expects `b5ua…`. Found by
  driving it against a live instance; fixed to read the field.
- **No route reports the branch-table head**, so a REST client cannot name the archive
  scope. `GET /branches` gives each branch's head and `POST /contribute` returns the new
  archive head, but nothing reads it. `/archive/claims/{id}` therefore exists as a read
  whose scope a client cannot address as a whole. Worth a route.
- **Node's TypeScript support forbids parameter properties.** `source.ts` used
  `constructor(private params: MockParams)`, which strip-only mode cannot transform, so core
  could not be imported by a test. Rewritten as plain fields — core stays runnable under
  node, which is how the benches already work and what makes a test runner unnecessary.
- **The mock had to answer a scoped query without becoming an engine.** The generator now
  stamps each claim with the branch whose contribution introduced it, which costs nothing at
  generation and needs no walk. Cross-branch references are then absent from a scoped answer
  — which is what the contract does too, returning out-of-scope references as hash-only
  stubs.
- **Tests run on `node --test`, no framework added.** Node strips the types, as the benches
  already rely on. `make -C frontend test` runs them.
