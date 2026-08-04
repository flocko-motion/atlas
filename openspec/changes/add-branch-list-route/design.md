## Context

`OpBranchList` works and has no route. `internal/core/execute.go:144` calls
`Archive.GetBranches`, maps each to `{name, head}` and serves
`{"branches":[…]}`; `execute_test.go:130` exercises it against `$branches`.
`core-access` grants **R** on `$branches` as "enumerates the table". The only thing
absent is the transport binding.

So this is a contract change, not a feature build. The one decision worth recording is
what to call the route.

## Goals / Non-Goals

**Goals:**

- A route that lets a client discover an archive's branches without prior knowledge.

**Non-Goals:**

- **Anything in `internal/core/`.** The operation, its right and its payload exist.
- **Branch metadata beyond name and head.** Whatever else a branch carries is reachable
  by reading the branch-table claim; this route answers "what may I ask for".
- **Filtering or paging.** A branch table is small by construction — it is one claim's
  edges. If an archive ever holds enough branches to need paging, `POST /query` over
  `$archive` is the surface for that, not this.

## Decisions

### `GET /$branches`, not `GET /branches`

Both are safe under the router. `rest-api` already fixes the collision rule: a fixed
route is ambiguous only when it *"would sit at the top level with a single `{id}`
segment"*, because `/x/{id}` and `/{branch}/head` match the same shape with neither
more specific. A single fixed segment with no parameter is not that case — `/health`
and `/query` already occupy it — and a branch named `branches` would still read as
`/branches/head`, two segments, distinct.

The reason to prefer `$branches` is that it is the name the grant is written against.
A reader comparing the route to the access policy sees one string, not two that happen
to correspond: **R** on `$branches` permits `GET /$branches`. `$universe` already
establishes the pattern on the wire at `/$universe/claim/{id}`, and because `$` is
illegal in an ordinary branch name, the route is unshadowable by construction rather
than by an argument about matching rules.

`$` is a sub-delimiter and legal unescaped in a path segment, so no client-side
encoding is required.

### Cacheable with revalidation

A branch's head moves, and this returns one per branch, so the response is exactly as
volatile as `GET /{branch}/head` — which the contract already makes cacheable with
revalidation via a weak `ETag`. Same treatment, for the same reason.

### Name and head together

The core op already returns both, and a client rendering an archive wants both: the
name to address further reads, the head to know what it is looking at. Returning only
names would have every client immediately issue N requests to `GET /{branch}/head` to
recover what this call already held.

### A read, not an operational endpoint

`/system/…` is where `rest-api` puts operational extensions — layers, verification runs
— and also where it puts anything that would otherwise be ambiguous against a branch
read. Neither applies. Enumerating branches is a read of archive content, gated by a
data right (**R**), so it belongs with the read surface.

## Risks / Trade-offs

- **`$` in a path is unusual** → some HTTP tooling and proxies mangle it, and a client
  library may encode it as `%24`. `/$universe/claim/{id}` already carries that exposure,
  so this adds no new class of problem, but the server should treat `%24` as equivalent
  on the way in.
- **The head set is not atomic across branches** unless read from one snapshot →
  `core-execution` already requires one archive snapshot per request, so the listing is
  consistent as of a single head. Worth a test rather than an assumption.
