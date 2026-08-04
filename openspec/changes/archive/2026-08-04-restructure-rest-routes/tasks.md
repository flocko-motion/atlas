## 1. The contract

- [x] 1.1 Rewrite the read paths in `openapi/openapi.yaml`: `/{branch}/head` → `/branches/{branch}/head`; `/{branch}/claim/{id}` → `/branches/{branch}/claims/{id}`; the same for the `/content` forms; `/$universe/claim/{id}` → `/universe/claims/{id}`. Add `/archive/claims/{id}` and its `/content` form — a scope with no route today.
- [x] 1.1a Document each scope collection as the reserved scope it reads: `/archive/…` is `$archive` (the closure of the whole Ranke-Archive), `/universe/…` is `$universe`. Say that this is the same scope `select.branch` names in a RankeQL body and the same target the grant is written against, so route, query and policy read as one concept. Do not offer `/branches/$archive/…` as an alternative path.
- [x] 1.2 Add `GET /branches`, with a response schema matching what core already emits: `{"branches":[{"name":…,"head":…}]}`. Take the shape from `branchList`/`branchEntry` in `internal/core/execute.go` rather than inventing one, so the wire and the core agree by construction.
- [x] 1.3 Rename `/system/verification` → `/system/verifications` and its sub-paths. `POST /system/verifications/{id}/cancel` keeps its verb — a state transition with no natural noun.
- [x] 1.4 Update operation ids to match: `getUniverseClaim` → `getClaim`, `getUniverseClaimContent` → `getClaimContent`, `listVerifications` and friends onto the plural path. Add `listBranches`. The archive scope needed two more: `getArchiveClaim`, `getArchiveClaimContent`.
- [x] 1.5 Document the cache posture per route: claims immutable by id, the branch listing and branch head with revalidation.
- [x] 1.6 Document which right each route needs — **R** on the branch for branch-scoped reads, **R** on `$branches` for the listing, **R** on `$universe` for `/claims/{id}` — and confirm no path segment carries a `$`.
- [x] 1.7 `make generate`. Hand-edit no `*.gen.*`.

## 2. The handlers

- [x] 2.1 Rewire `adapters/endpoints/rest_http/endpoints_read.go` to the regenerated interface. The `Request` each handler builds is unchanged apart from the listing — only the routes moved.
- [x] 2.2 Add the listing handler: `Op: core.OpBranchList`, `Branch: core.Branches`. The scope matters — `authorize` checks the operation's right against the branch the request names, so `$branches` is what makes the **R** grant apply.
- [x] 2.3 Map each scope collection to its reserved target when building the `Request`: `/universe/…` → `Branch: core.Universe`, `/archive/…` → `Branch: core.Archive`. The grant checked is what the path names, even though the path drops the `$`.

## 2a. The scope arms in core

- [x] 2a.1 Correct the `$universe` arm of `scopedClaim`. It called `archive.GetClaim`, which is `GetFromClosure(…, BranchArchive, [bth], id)` — the archive-head read. The unconfined call is `ranke.GetClaim(ctx, c.store, id)`. Confirmed still true of ranke-go v0.12.0 (`archive.go:90`), so the diagnosis held after the version bump.
- [x] 2a.2 Add the `$archive` arm: `Branch == core.Archive` → `archive.GetClaim`, which is already exactly that scope. `core.Archive` had to be re-exported from `access` alongside `Universe` and `Branches`.
- [x] 2a.3 Same treatment for the content arm, so `/universe/claims/{id}/content` and `/archive/claims/{id}/content` resolve through the scope their path names. The Universe arm reads the claim unconfined and takes the bytes off it, there being no archive-level equivalent that skips the closure.
- [x] 2a.4 Confirm `R $archive` is grantable — `access.go` lists `$archive` in `reserved`, and only `$universe` carries the read-only restriction — so the new route has a grant to be gated by. Locked in by `TestScopeRoutesAreSeparatelyGated`.
- [x] 2.4 Set the cache headers the contract declares. Nothing set them before, so `respondImmutable` / `respondRevalidated` were added: a strong validator (the id) with an immutable lifetime for a by-id read, a weak one derived from the body for a moving target, and `304` on a matching `If-None-Match` — since `no-cache` with no validator promises a revalidation the server cannot answer.

## 3. Consumers in this repo

- [x] 3.1 `frontend/src/core/data/source.ts` builds `/${branch}/head`; moved to `/branches/${branch}/head` (and the name is now percent-encoded, a branch name being free text in a path segment). No other constructed paths in the file. `make -C frontend check` clean.
- [x] 3.2 `cmd/generator` contributes over `POST /contribute`, unmoved — but it *does* read one path: `client.go`'s `head` built `/{branch}/head`. Moved with the rest.
- [x] 3.3 `make verify` green, and the smoke script passes against the moved routes. Smoke now also asserts `GET /branches` lists what it seeded, so the new route is covered end to end over real HTTP.

## 4. Tests

- [x] 4.1 A client knowing no branch name lists branches and gets name and head for each.
- [x] 4.2 An account without `R $branches` is denied the listing, and the denial is the access decision rather than a not-found.
- [x] 4.3 An account without `R $universe` is denied `GET /universe/claims/{id}`, and one without `R $archive` is denied `GET /archive/claims/{id}` — dropping the sentinel from the path did not drop the privilege check with it.
- [x] 4.3a The three scopes genuinely differ: a claim in the Universe but outside the head's closure is returned by `/universe/claims/{id}` and reported not-found by `/archive/claims/{id}`. Verified it bites — reverting 2a.1 fails this test with a 404.
- [x] 4.3b An ordinary glob confers neither reserved scope: an account holding `R *` is denied both.
- [x] 4.3c `/branches/$archive/claims/{id}` is not-found, there being exactly one route per scope. Covers all three reserved names.
- [x] 4.4 A branch named `branches` resolves at `/branches/branches/head`, and `/branches` still lists the table.
- [x] 4.5 A branch named `health` or `claims` shadows no fixed route — with `query` and `system` too, and `/health` asserted still live while a branch of that name exists.
- [x] 4.6 The listing is consistent as of one snapshot. Tested as the invariant itself, in `internal/core/snapshot_test.go`: a counting sequencer asserts a listing opens exactly one archive. No assertion on the returned values could catch re-opening per branch, since a mix only appears when a merge lands mid-listing.
- [x] 4.7 An empty archive returns an empty list rather than an error, so a fresh instance is explorable.
- [x] 4.8 Every pinned GET route is typable unquoted: the contract's own path keys are asserted free of anything a POSIX shell would expand.

## Found while executing

- **The example stack could not see its own new route.** `examples/minimal/config.json` granted `CR *`, and an ordinary glob reaches no reserved target — so `GET /branches` answered `403` on the one stack the docs tell a reader to launch. The example now grants the reserved reads by name (`R $branches`, `R $archive`, `R $universe`) and `C $branches`. Adding a route gated on a reserved target means granting it somewhere, or the feature ships invisible.
- **The `adapter-endpoint` delta still pinned `GET /claims/{id}`** — the bare-path shape `design.md` considered and rejected in favour of naming every scope. Corrected to the two scope routes, so the two deltas agree.
- **The deltas would not archive as written**, and each refusal was a real defect. "Every claim read names one of the three scopes" sat under MODIFIED with no such requirement in the spec — it is new, so it moved to ADDED. The rewritten "Reserved top-level path segments" requirement dropped the two scenarios the spec already carried; both are still true under the new layout (the second more strongly — an operational route cannot be ambiguous with a branch read *because* branch reads left the root), so they were restated rather than dropped.
- **Two requirements outside the delta named moved routes**: the 404 rule's out-of-closure scenario cited `GET /{branch}/claim/{id}`, and "Operational endpoints are ranke-db extensions" still listed `/system/verification` throughout. Corrected in the spec and recorded in this change's delta, so no requirement pins a route that no longer exists.
