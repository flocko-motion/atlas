## Why

A client cannot discover what branches an archive holds. Every read route takes
`{branch}` as a path parameter, so a client must already know a branch name to use the
API — and nothing on the wire will tell it one. `OpBranchList` is implemented
(`internal/core/execute.go:144`), returns name and head per branch, and is tested
(`execute_test.go:130`); `core-access` grants **R** on `$branches` as *"enumerates the
table"*. Only a route is missing.

Adding one exposed the reason it was missing. `/{branch}/…` sits at the **root**, so any
first path segment can be a branch name. That leaves no free root namespace, and
everything else has had to work around it: operational routes were pushed under
`/system/` — the contract says so outright, *"a root-level `/x/{id}` and `/{branch}/head`
match the same path with neither more specific"* — and the archive-wide claim read had to
borrow a sentinel, `/$universe/claim/{id}`, to avoid being mistaken for a branch.

That sentinel is a library device on a public wire. `$` marks reserved targets in the
grants language because it is illegal in branch names and so escapes globbing. A REST
client should not have to know that. It also does not survive a shell — `curl
http://host/$universe/claim/x` sends `http://host/claim/x` — while these GET routes exist
precisely to be reachable from a browser or `curl` without JSON.

So the fix is the layout, not another special case: give branches their own collection
and the root namespace comes back.

## What Changes

- **Nest branch routes under a collection**: `/branches/{branch}/head`,
  `/branches/{branch}/claims/{id}`, `/branches/{branch}/claims/{id}/content`. No branch
  name can shadow a fixed route once `{branch}` is a second segment rather than a first.
- **Add `GET /branches`** — the branch table's branches, each by name and current head.
  It becomes the obvious member of the collection rather than a route needing a sentinel.
- **Retire `$universe` from the path**: the archive-wide claim read becomes
  `GET /claims/{id}` and `GET /claims/{id}/content`. The *absence* of a branch scope is
  what makes the read archive-wide, so the URL's shape now carries the meaning a sentinel
  used to. It still requires **R** on `$universe`.
- **Pluralise the collection member**: `claim/{id}` → `claims/{id}`, matching
  `/branches` and ordinary REST usage.
- Keep `/system/…` for operational extensions. Its original justification — dodging
  ambiguity with `/{branch}/…` — is gone, but the grouping still says something true:
  these are ranke-db extensions beyond the paper's surface, not archive content.
  `/system/verification` becomes `/system/verifications`, a collection like the others.

`$`-names stay where they belong. They remain **grant targets** in `core-access`
(`$universe`, `$archive`, `$branches`) and **scope values** inside a RankeQL body, which
is the library's vocabulary. They leave only the URL path.

## Capabilities

### Modified Capabilities

- `rest-api`: the route layout — branches as a collection, the archive-wide claim read
  without a sentinel, the branch listing, and a root namespace that no longer needs
  defending.
- `adapter-endpoint`: the GET subset it pins, restated against the new paths.

## Impact

- **`openapi/openapi.yaml`**: every read path is rewritten and one is added. Operation
  ids change with them (`getUniverseClaim` → `getClaim`). `make generate` then reproduces
  the Go server interface, the TS client and both references.
- **`adapters/endpoints/rest_http/`**: handlers rewired to the regenerated interface; one
  new handler for the listing, building a `Request` with `Op: OpBranchList` and
  `Branch: core.Branches`.
- **`frontend/`**: one call site — `core/data/source.ts:101` builds `/${branch}/head`.
- **Not affected**: `internal/core/`. Every operation, its right and its payload already
  exist; this change reaches them and renames nothing behind the port.
- **Not affected**: `core-access` and RankeQL. The grant targets and query scopes keep
  their `$` names; only the path stops echoing them.
- **No migration.** The REST API is still in design; there is no released consumer to
  carry forward, so the layout is settled on its merits alone.
