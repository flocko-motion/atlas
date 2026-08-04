## Why

A client cannot discover what branches an archive holds. Every read route takes
`{branch}` as a path parameter, so a client must already know a branch name to use the
API at all — and nothing on the wire will tell it one.

The capability is built and orphaned at the transport. `OpBranchList` is implemented
(`internal/core/execute.go:144`), returns name and head per branch, and is tested
against `$branches` (`execute_test.go:130`). `core-access` grants **R** on `$branches`
as *"enumerates the table"*. What is missing is a route: `endpoints_read.go` reaches
`OpBranchHead` and nothing reaches `OpBranchList`.

Nor does the query surface substitute. RankeQL scopes are `$universe`, `$archive`, or a
branch name (`ranke/query.go:24-38`); `$branches` is a ranke-db grant target, not a
query scope, so a query naming it resolves through `GetBranch("$branches")` and comes
back not-found.

The gap is in the contract rather than the code: `rest-api` and `adapter-endpoint` both
pin the GET route set, and neither includes a branch listing, so `openapi.yaml` is
faithfully generated from a spec that omits it.

This blocks any client that does not have a branch name configured in advance — the
Ranke Explorer cannot populate a branch picker, and a new archive cannot be explored at
all.

## What Changes

- Add **`GET /$branches`** to the REST contract: the branch table's branches, each by
  name and current head, requiring **R** on `$branches`.
- Name it for the grant it is held against. `$` is illegal in an ordinary branch name,
  so the route cannot be shadowed, and `/$universe/claim/{id}` already puts a reserved
  target on the wire — so the route and the grant read identically.
- Make it **cacheable with revalidation**, as `GET /{branch}/head` is: a head is a
  moving target, and this returns one per branch.
- Return **head alongside name**, which the core op already does. A client that wants
  to render an archive's branches needs both, and carrying the head here saves a
  request per branch against `GET /{branch}/head`.

## Capabilities

### Modified Capabilities

- `rest-api`: adds `GET /$branches` to the pinned read surface, and records that a
  reserved-target route at the top level cannot collide with a branch read.
- `adapter-endpoint`: adds the route to the GET subset the REST endpoint pins.

## Impact

- **`openapi/openapi.yaml`**: one new path and a response schema matching what
  `branchList` already emits — `{"branches":[{"name":…,"head":…}]}`. `make generate`
  then reproduces the Go server interface, the TS client and the references.
- **`adapters/endpoints/rest_http/`**: one handler, building a `Request` with
  `Op: OpBranchList` and `Branch: core.Branches`, since the grant is checked against
  the scope the request names.
- **Not affected**: `internal/core/`. The operation, its access right and its payload
  all exist; this change reaches them.
- **Not affected**: `core-access`, which already defines **R** on `$branches` as
  enumerating the table, and `core-execution`, which already covers dispatch and
  serving.
