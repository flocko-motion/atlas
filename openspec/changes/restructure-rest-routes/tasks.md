## 1. The contract

- [ ] 1.1 Rewrite the read paths in `openapi/openapi.yaml`: `/{branch}/head` → `/branches/{branch}/head`; `/{branch}/claim/{id}` → `/branches/{branch}/claims/{id}`; the same for the `/content` forms; `/$universe/claim/{id}` → `/claims/{id}` and its `/content` form.
- [ ] 1.2 Add `GET /branches`, with a response schema matching what core already emits: `{"branches":[{"name":…,"head":…}]}`. Take the shape from `branchList`/`branchEntry` in `internal/core/execute.go` rather than inventing one, so the wire and the core agree by construction.
- [ ] 1.3 Rename `/system/verification` → `/system/verifications` and its sub-paths. `POST /system/verifications/{id}/cancel` keeps its verb — a state transition with no natural noun.
- [ ] 1.4 Update operation ids to match: `getUniverseClaim` → `getClaim`, `getUniverseClaimContent` → `getClaimContent`, `listVerifications` and friends onto the plural path. Add `listBranches`.
- [ ] 1.5 Document the cache posture per route: claims immutable by id, the branch listing and branch head with revalidation.
- [ ] 1.6 Document which right each route needs — **R** on the branch for branch-scoped reads, **R** on `$branches` for the listing, **R** on `$universe` for `/claims/{id}` — and confirm no path segment carries a `$`.
- [ ] 1.7 `make generate`. Hand-edit no `*.gen.*`.

## 2. The handlers

- [ ] 2.1 Rewire `adapters/endpoints/rest_http/endpoints_read.go` to the regenerated interface. The `Request` each handler builds is unchanged apart from the listing — only the routes moved.
- [ ] 2.2 Add the listing handler: `Op: core.OpBranchList`, `Branch: core.Branches`. The scope matters — `authorize` checks the operation's right against the branch the request names, so `$branches` is what makes the **R** grant apply.
- [ ] 2.3 Keep `Branch: core.Universe` on the unscoped claim routes, so the grant checked is still **R** on `$universe` even though the path no longer says so.
- [ ] 2.4 Set the cache headers the contract declares.

## 3. Consumers in this repo

- [ ] 3.1 `frontend/src/core/data/source.ts:101` builds `/${branch}/head`; move it to `/branches/${branch}/head`. Check for other constructed paths in the same file.
- [ ] 3.2 Check `cmd/generator` — it contributes over `POST /contribute`, which is unmoved, but confirm it reads nothing on the old paths.
- [ ] 3.3 `make verify` green, and the smoke script still passes against the moved routes.

## 4. Tests

- [ ] 4.1 A client knowing no branch name lists branches and gets name and head for each.
- [ ] 4.2 An account without `R $branches` is denied the listing, and the denial is the access decision rather than a not-found.
- [ ] 4.3 An account without `R $universe` is denied `GET /claims/{id}`, so dropping the sentinel from the path did not drop the privilege check with it.
- [ ] 4.4 A branch named `branches` resolves at `/branches/branches/head`, and `/branches` still lists the table.
- [ ] 4.5 A branch named `health` or `claims` shadows no fixed route.
- [ ] 4.6 The listing is consistent as of one snapshot: heads do not come from different points in time within a single response. `core-execution` requires one archive snapshot per request — test it rather than assume it.
- [ ] 4.7 An empty archive returns an empty list rather than an error, so a fresh instance is explorable.
- [ ] 4.8 Every pinned GET route is typable unquoted: assert no path in the generated spec contains a character a shell would expand.
