## 1. The contract

- [ ] 1.1 Rewrite the read paths in `openapi/openapi.yaml`: `/{branch}/head` → `/branches/{branch}/head`; `/{branch}/claim/{id}` → `/branches/{branch}/claims/{id}`; the same for the `/content` forms; `/$universe/claim/{id}` → `/universe/claims/{id}`. Add `/archive/claims/{id}` and its `/content` form — a scope with no route today.
- [ ] 1.1a Document each scope collection as the reserved scope it reads: `/archive/…` is `$archive` (the closure of the whole Ranke-Archive), `/universe/…` is `$universe`. Say that this is the same scope `select.branch` names in a RankeQL body and the same target the grant is written against, so route, query and policy read as one concept. Do not offer `/branches/$archive/…` as an alternative path.
- [ ] 1.2 Add `GET /branches`, with a response schema matching what core already emits: `{"branches":[{"name":…,"head":…}]}`. Take the shape from `branchList`/`branchEntry` in `internal/core/execute.go` rather than inventing one, so the wire and the core agree by construction.
- [ ] 1.3 Rename `/system/verification` → `/system/verifications` and its sub-paths. `POST /system/verifications/{id}/cancel` keeps its verb — a state transition with no natural noun.
- [ ] 1.4 Update operation ids to match: `getUniverseClaim` → `getClaim`, `getUniverseClaimContent` → `getClaimContent`, `listVerifications` and friends onto the plural path. Add `listBranches`.
- [ ] 1.5 Document the cache posture per route: claims immutable by id, the branch listing and branch head with revalidation.
- [ ] 1.6 Document which right each route needs — **R** on the branch for branch-scoped reads, **R** on `$branches` for the listing, **R** on `$universe` for `/claims/{id}` — and confirm no path segment carries a `$`.
- [ ] 1.7 `make generate`. Hand-edit no `*.gen.*`.

## 2. The handlers

- [ ] 2.1 Rewire `adapters/endpoints/rest_http/endpoints_read.go` to the regenerated interface. The `Request` each handler builds is unchanged apart from the listing — only the routes moved.
- [ ] 2.2 Add the listing handler: `Op: core.OpBranchList`, `Branch: core.Branches`. The scope matters — `authorize` checks the operation's right against the branch the request names, so `$branches` is what makes the **R** grant apply.
- [ ] 2.3 Map each scope collection to its reserved target when building the `Request`: `/universe/…` → `Branch: core.Universe`, `/archive/…` → `Branch: core.Archive`. The grant checked is what the path names, even though the path drops the `$`.

## 2a. The scope arms in core

- [ ] 2a.1 Correct the `$universe` arm of `scopedClaim` (`internal/core/execute.go:163`). It calls `archive.GetClaim`, which is `GetFromClosure(…, BranchArchive, [bth], id)` — the archive-head read. The unconfined call is `ranke.GetClaim(ctx, u, id)`. As written, the privileged route cannot reach a claim outside the current head's closure, which is what `$universe` exists for.
- [ ] 2a.2 Add the `$archive` arm: `Branch == core.Archive` → `archive.GetClaim`, which is already exactly that scope.
- [ ] 2a.3 Same treatment for the content arm, so `/universe/claims/{id}/content` and `/archive/claims/{id}/content` resolve through the scope their path names.
- [ ] 2a.4 Confirm `R $archive` is grantable — `access.go` lists `$archive` in `reserved`, and only `$universe` carries the read-only restriction — so the new route has a grant to be gated by.
- [ ] 2.4 Set the cache headers the contract declares.

## 3. Consumers in this repo

- [ ] 3.1 `frontend/src/core/data/source.ts:101` builds `/${branch}/head`; move it to `/branches/${branch}/head`. Check for other constructed paths in the same file.
- [ ] 3.2 Check `cmd/generator` — it contributes over `POST /contribute`, which is unmoved, but confirm it reads nothing on the old paths.
- [ ] 3.3 `make verify` green, and the smoke script still passes against the moved routes.

## 4. Tests

- [ ] 4.1 A client knowing no branch name lists branches and gets name and head for each.
- [ ] 4.2 An account without `R $branches` is denied the listing, and the denial is the access decision rather than a not-found.
- [ ] 4.3 An account without `R $universe` is denied `GET /universe/claims/{id}`, and one without `R $archive` is denied `GET /archive/claims/{id}` — dropping the sentinel from the path did not drop the privilege check with it.
- [ ] 4.3a The three scopes genuinely differ: a claim in the Universe but outside the head's closure is returned by `/universe/claims/{id}` and reported not-found by `/archive/claims/{id}`. This is the test that would have caught the current miswiring, where both call the same thing.
- [ ] 4.3b An ordinary glob confers neither reserved scope: an account holding `R *` is denied both. `matchBranch` already guarantees it by requiring an exact match on a `$`-prefixed target — lock it in rather than rely on it.
- [ ] 4.3c `/branches/$archive/claims/{id}` is not-found, there being exactly one route per scope.
- [ ] 4.4 A branch named `branches` resolves at `/branches/branches/head`, and `/branches` still lists the table.
- [ ] 4.5 A branch named `health` or `claims` shadows no fixed route.
- [ ] 4.6 The listing is consistent as of one snapshot: heads do not come from different points in time within a single response. `core-execution` requires one archive snapshot per request — test it rather than assume it.
- [ ] 4.7 An empty archive returns an empty list rather than an error, so a fresh instance is explorable.
- [ ] 4.8 Every pinned GET route is typable unquoted: assert no path in the generated spec contains a character a shell would expand.
