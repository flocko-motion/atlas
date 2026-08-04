## Why

`GET /branches/{branch}/head` answers one field, and nothing answers the rest. A client
that wants to say anything about a branch — how deep it is, when it last moved — has no
route to ask, so the explorer's scope picker shows a truncated head id because that is all
there is to show.

Worse, **no route reports the branch-table head**. `GET /branches` gives each branch's head
and `POST /contribute` returns the new archive head to whoever wrote it, but a reader
cannot obtain it. So `$archive` — a scope the access model reserves, RankeQL accepts, and
`/archive/claims/{id}` reads within — is a scope no client can name.

## What Changes

- Add **`GET /branches/{branch}/info`** beside `/head`: the branch's name, head, the head
  claim's **height**, and **when it last moved**.
- Add **`GET /archive/info`**: the **branch-table head**, its height, when it last moved,
  and how many branches the table holds. This is the only route that reports that head.
- Keep `/head` as it is — the narrow, cacheable id a client polls. `/info` is the richer
  read for the same resource, cacheable with revalidation.
- A **claim count is deliberately absent.** Counting a branch's claims is a walk of its
  closure, which is a query (`POST /query`), not a field on a resource.

## Capabilities

### Modified Capabilities

- `rest-api`: adds the two `/info` reads, and with the archive head becoming readable, the
  `$archive` scope becomes nameable by a client for the first time.

## Impact

- **`openapi/openapi.yaml`**: two paths, two schemas (`BranchInfo`, `ArchiveInfo`).
- **`internal/core/`**: `OpBranchInfo` and `OpArchiveInfo`, each one library call plus the
  head claim's typed fields.
- **`adapters/endpoints/rest_http/`**: two handlers, answering with revalidation.
- **`frontend/`**: `RestSource.branches()` reads `/archive/info`, so the picker offers the
  archive scope against a server as it already did against mock data.
- **Not affected**: `core-access`. Each route is gated by the scope it reads — **R** on the
  branch, **R** on `$archive` — with no new grant target.
