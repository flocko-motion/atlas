## 1. The contract

- [ ] 1.1 Add `GET /$branches` to `openapi/openapi.yaml`, alongside the other read routes rather than under `/system/` — it is a read of archive content gated by a data right, not an operational extension.
- [ ] 1.2 Give it a response schema matching what core already emits: `{"branches":[{"name":…,"head":…}]}`. Take the shape from `branchList`/`branchEntry` in `internal/core/execute.go` rather than inventing one, so the wire and the core agree by construction.
- [ ] 1.3 Document it as cacheable with revalidation — weak `ETag`, same treatment as `GET /{branch}/head`, since it carries one moving head per branch.
- [ ] 1.4 Note in the route description that it requires **R** on `$branches`, matching how the other routes reference their rights.
- [ ] 1.5 `make generate`, so the Go server interface, the TS client and the two references all come from the spec. Hand-edit no `*.gen.*`.

## 2. The handler

- [ ] 2.1 Implement the generated interface method in `adapters/endpoints/rest_http/endpoints_read.go`, building a `core.Request` with `Op: core.OpBranchList` and `Branch: core.Branches`. The scope matters: `authorize` checks the operation's right against the branch the request names, so `$branches` is what makes the **R** grant apply.
- [ ] 2.2 Accept `%24` as equivalent to `$` on the way in, since a client library may percent-encode it. The same applies to the existing `/$universe/…` routes; fix both if only one handles it.
- [ ] 2.3 Set the cache headers the contract declares.

## 3. Tests

- [ ] 3.1 A client knowing no branch name lists branches and gets name and head for each.
- [ ] 3.2 An account without `R $branches` is denied, and the denial is the access decision rather than a not-found.
- [ ] 3.3 The listing is consistent as of one snapshot: heads do not come from different points in time within a single response. `core-execution` requires one archive snapshot per request — test it rather than assume it.
- [ ] 3.4 A branch named `branches` is still addressed as `/branches/head` and does not shadow the reserved route.
- [ ] 3.5 An empty archive returns an empty list rather than an error, so a fresh instance is explorable.
