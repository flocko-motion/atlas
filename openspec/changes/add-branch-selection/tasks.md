## 1. Discovery through the port

- [ ] 1.1 Add `branches()` to the `DataSource` port in `frontend/src/core/data/source.ts`, returning each branch by name and current head.
- [ ] 1.2 Implement it on `RestSource` against `GET /branches`. Depends on `restructure-rest-routes`.
- [ ] 1.3 Implement it on `MockSource` from what the generator already records — `branches: Record<string, ClaimId>` (`mock/model.ts:64`). One code path for both, per the port's own rule: *"one data path, not a real and a test one."*
- [ ] 1.4 Include the archive itself as a selectable scope, carrying the archive head. Exclude `$universe`: it has no head, so no closure, which is why RankeQL rejects a read there without an explicit `Select.Head`.

## 2. Delete the assumed branch name

- [ ] 2.1 Remove `branch: 'main'` from `DEFAULT_QUERY` (`core/query.ts:26`). A branch name is a fact about the archive in front of the explorer, not a constant.
- [ ] 2.2 Hold no branch name until the listing answers, and issue no scope-confined read before then.
- [ ] 2.3 Select automatically only when the listing returns exactly one branch. With several, ask — never by name, position or convention.

## 3. The scope predicate

- [ ] 3.1 Add the selected scope to `ViewState` in `core/store.ts`, alongside `contributionRange` and `classes`. Null means everything loaded.
- [ ] 3.2 Compute membership as reachability from the scope's head over the union graph, in `core/graph/`. Not a label on the claim: a claim reached from two branches is one node belonging to both.
- [ ] 3.3 Cache each closure by head id. Claims are immutable, so a closure is stable for its head; a branch that advances yields a new head and a new entry, and a view still pinned to the old head stays correct.
- [ ] 3.4 Apply it as a reducer swap in `render/`. No store clear, no refetch, no node or edge added or removed.

## 4. The picker

- [ ] 4.1 Put a scope picker in the header's tools slot, listing the archive and each branch with its current head.
- [ ] 4.2 Report a selected scope whose head is absent from the store as *not loaded*, distinct from a scope that is loaded and empty. Blanking silently hides exactly the question the user is asking.
- [ ] 4.3 Carry the selection into the query as its scope.

## 5. Tests

- [ ] 5.1 Branches list identically from a REST connection and from mock data, through the one port method.
- [ ] 5.2 Selecting a scope narrows the view while the store keeps every claim it held.
- [ ] 5.3 Switching between two loaded scopes and back adds and removes nothing, and refetches nothing.
- [ ] 5.4 A claim reachable from two branch heads is drawn under either, the store holding one node for it.
- [ ] 5.5 A closure is computed once per head and reused on reselection.
- [ ] 5.6 A listed-but-unloaded scope reports itself rather than drawing an empty view.
- [ ] 5.7 The picker offers no `$universe` entry.
- [ ] 5.8 With no listing yet answered, no scope-confined read is issued and no branch name is assumed.
