# rest-api Specification

## Purpose
The REST/HTTP binding of the read-and-contribute surface: the routes the paper pins,
the media types claims and content cross the wire in, how a RankeQL query is carried
and its results framed, the status and error model, and the operational endpoints
ranke-db adds beyond the paper. `openapi/openapi.yaml` is its single source of truth,
and every artifact — the Go server, the TS client, the references — is generated from
it. Authorization is deliberately not part of the contract: the endpoint carries the
request's credential to core, and `core-access` decides.
## Requirements
### Requirement: OpenAPI is the single source of truth for the REST contract
The REST/HTTP contract SHALL be defined in `openapi/openapi.yaml`, from which the Go
server interface and models, the TS/JS client, and the HTML and Markdown references
are generated (`make generate`). Generated artifacts (`*.gen.*`) SHALL NOT be
hand-edited; the contract changes only by editing the spec and regenerating.

#### Scenario: The generated server matches the spec
- **WHEN** `make generate` runs against `openapi/openapi.yaml`
- **THEN** `openapi/openapi.gen.go`, the TS client, and the two references are produced from it, and no generated file is edited by hand

### Requirement: REST binds the paper's read-and-contribute surface
The REST endpoint SHALL expose the full surface as `POST /query` (a RankeQL query)
and `POST /contribute` (an atomic contribution of one or more signed claims), and
SHALL pin the cacheable GET subset:

- `GET /branches` — the branch table's branches, each by name and current head
- `GET /branches/{branch}/head` — that branch's current head id
- `GET /branches/{branch}/claims/{id}` — a claim within that branch's closure
- `GET /branches/{branch}/claims/{id}/content` — that claim's content
- `GET /archive/claims/{id}` — a claim within the archive head's closure, across all branches
- `GET /archive/claims/{id}/content` — that claim's content
- `GET /universe/claims/{id}` — a claim anywhere the Universe holds, reached without any closure
- `GET /universe/claims/{id}/content` — that claim's content

The by-id claim forms SHALL be HTTP-cacheable immutably, their ids content-addressing
the bytes; the branch listing and the branch head SHALL be cacheable with revalidation,
both carrying heads that move. Content SHALL be addressed by the claim holding it rather
than by a raw hash, so the read stays scoped as the claim's route is scoped.

`GET /branches` SHALL be reachable without prior knowledge of any branch name, so a
client discovers what it may address, and SHALL require the **R** right on the reserved
`$branches` target (see `core-access`).

#### Scenario: The full read surface is a query POST
- **WHEN** a client reads via `POST /query` with a RankeQL body
- **THEN** the response is the result set the query defines

#### Scenario: A by-id claim GET is cacheable without a JSON body
- **WHEN** a client issues `GET /branches/{branch}/claims/{id}`
- **THEN** the claim is returned without a query body and is cacheable immutably by id

#### Scenario: The branch head GET is revalidated
- **WHEN** a client issues `GET /branches/{branch}/head`
- **THEN** the current head id (a moving target) is returned as a cacheable-with-revalidation resource

#### Scenario: Content is fetched through the claim that holds it
- **WHEN** a client issues `GET /branches/{branch}/claims/{id}/content`
- **THEN** the claim's content is streamed without the client knowing whether the bytes were stored inline or as a separate blob

#### Scenario: A client with no branch name discovers what it may read
- **WHEN** a client that knows no branch name issues `GET /branches`
- **THEN** it receives every branch the table holds, each with its name and current head, and can address the branch routes from that list

#### Scenario: Listing branches needs R on the reserved target
- **WHEN** an account holding no `R $branches` grant issues `GET /branches`
- **THEN** the request is denied by the access decision

### Requirement: The query body binds the RankeQL type without restating it
The `POST /query` body SHALL bind the RankeQL `Query` type as the normative spec fixes
it (§RankeQL) — `select`, `where`, `output`, `order`, `limit`, `execution` — using that
type's own field names and values, and SHALL define none of its semantics. The binding
SHALL be faithful: every axis the type defines SHALL be expressible over the wire, with
none folded into another or left unreachable. Where the JSON schema cannot express a
rule the type states, the endpoint SHALL enforce it at the boundary and answer a
malformed query with `400`.

#### Scenario: Every axis of the language reaches the engine
- **WHEN** a query names `output.shape`, `output.detail`, `output.form`, `output.encoding`, a `path` step's `min`/`max` hops, an `order` key's `compare`, or an `execution.report` verbosity
- **THEN** each is carried to the query engine as the value the spec names, none collapsed into another axis

#### Scenario: A scope is mandatory
- **WHEN** a query arrives with no `select.branch`
- **THEN** the response is `400`, since the scope is what the grant is held against

#### Scenario: The unconfined scope requires a head
- **WHEN** a query names `select.branch: $universe` without `select.head`
- **THEN** the response is `400`, because that scope confines nothing and so offers no head to fall back on

### Requirement: The query scope is a branch or a reserved scope
The contract SHALL accept as `select.branch` a branch name (confining the read to that
branch), `$archive` (the whole Ranke-Archive), or `$universe` (unconfined and
privileged). `select.head` SHALL pin the closure read, giving a paged read an immutable
snapshot that cannot shift while the archive advances.

#### Scenario: The archive scope reads across branches
- **WHEN** a query names `select.branch: $archive`
- **THEN** the read is scoped to the whole Ranke-Archive, and access is decided against that reserved scope

#### Scenario: A pinned head fixes a paged read
- **WHEN** a client pages a query with `select.head` set
- **THEN** every page reads the same closure even if the branch head advances between pages

### Requirement: Cypher is never a client route
The REST contract SHALL NOT expose a client-facing Cypher/GQL endpoint. Cypher/GQL is
an internal execution engine the read planner lowers queries to (`core-query`,
`adapter-storage`); a client expresses reads only through the RankeQL body.

#### Scenario: There is no client Cypher endpoint
- **WHEN** the REST surface is inspected
- **THEN** it offers `POST /query` and no route that accepts a raw Cypher/GQL string from a client

### Requirement: Claims and content cross the wire in canonical form
The REST contract SHALL carry a claim fetched by id as its signed CBOR bytes
(`application/cbor`), content blobs as raw bytes (`application/octet-stream`), and
queries, result envelopes, and errors as JSON (`application/json`). A by-id claim read
SHALL return the canonical signed CBOR, independently verifiable against the id and
never reconstructed from a projection.

#### Scenario: A claim read returns canonical CBOR
- **WHEN** a claim is fetched by id
- **THEN** its signed CBOR bytes are returned, verifiable against the id, not a JSON rendering

#### Scenario: A contribution is submitted as signed CBOR
- **WHEN** a client contributes claims via `POST /contribute`
- **THEN** the body is the signed CBOR of the claim(s) and their referenced closure

### Requirement: A query response is a framed sequence whose verifiability follows its shaping
The contract SHALL stream a query's results as a sequence of items, one per result,
framing it by `output.encoding`: `json` as `application/json-seq` (RFC 7464) and `cbor`
as `application/cbor-seq` (RFC 8742). The framing SHALL be this binding's alone —
`output.encoding` names the per-claim serialization, not the sequence. Verifiability
SHALL follow the shaping, not the framing: `detail: claims` with `form: original` and
`encoding: cbor` reproduces the bytes a claim's id is computed over and is the only
combination directly verifiable against that id, and the contract SHALL mark every
other shaping as a convenience projection. When `execution.report` names a verbosity,
the sequence's final item SHALL be a report typed distinctly from result items.

#### Scenario: The media type frames the chosen encoding
- **WHEN** a query sets `output.encoding: cbor`
- **THEN** the response is `application/cbor-seq`, one CBOR item per result

#### Scenario: A JSON projection is not verifiable
- **WHEN** a query sets `output.encoding: json`
- **THEN** the results are a convenience projection, documented as not independently verifiable against the claim ids

#### Scenario: A requested report is the final item
- **WHEN** a query sets `execution.report`
- **THEN** the sequence's last item is the execution report, distinguishable from a result

### Requirement: The contract binds no authorization
The REST contract SHALL be independent of authorization: the endpoint extracts the
request's credential and passes it to core, and `core-access` decides on the subject's
grants (`adapter-auth`, `adapter-endpoint`, `core-access`). The contract SHALL bind no
per-operation rights and SHALL define no read/write/admin ladder — a route's shape
SHALL NOT vary by right. It MAY enumerate how a credential is *presented* (the header
and scheme each auth adapter answers), because an endpoint routes deterministically on
the presented scheme rather than inspecting a token's bytes; that enumeration SHALL
confer no rights, and a presentation with no configured adapter SHALL be `401`.

#### Scenario: The endpoint carries the credential to core
- **WHEN** any request arrives at a REST route
- **THEN** the endpoint passes its credential to core for the access decision, and the route's shape does not vary by right

#### Scenario: Presentation is enumerated, rights are not
- **WHEN** the REST contract is inspected
- **THEN** it names the credential schemes an endpoint dispatches on and binds no per-operation right and no read/write/admin ladder

### Requirement: The status and error model
The REST contract SHALL use: `200`/`201`/`202`/`204` for success as fitting the
operation; `400` for a malformed request; `401` when authentication is required or
failed; `403` when access is denied (the decision made in `core-access`); `404` when a
branch, claim, or content is unknown **or lies outside the named scope's closure**, the
two being indistinguishable; `409` when a contribution conflicts with the current head;
`429` when a resource-capped operation has no slot free; and `501` when an optional
capability the request needs is not configured. Every error body SHALL carry a stable
machine-readable code alongside its human-readable message, and a `403` body SHALL NOT
disclose the subject id or any onboarding hint.

#### Scenario: Out-of-closure is indistinguishable from absent
- **WHEN** a claim exists in the Universe but lies outside branch `{branch}`'s closure and is fetched via `GET /branches/{branch}/claims/{id}`
- **THEN** the response is `404`, identical to the response for a claim that does not exist

#### Scenario: A head conflict is a 409
- **WHEN** a contribution conflicts with the branch's current head
- **THEN** the response is `409`

#### Scenario: A client branches on the code, not the message
- **WHEN** any error response is returned
- **THEN** it carries a stable code from a fixed set, and the message is free to change

### Requirement: Reserved top-level path segments do not collide with branches
The REST contract SHALL keep branch names out of the root namespace by addressing every
branch-scoped route under the `/branches` collection, so that a branch name occupies a
second path segment and never a first. No branch name SHALL be able to shadow a fixed
route, whatever it is called, and a new fixed route SHALL require no defensive prefix to
stay clear of branch reads.

#### Scenario: A branch route never shadows a fixed route
- **WHEN** the router resolves `/health` while a branch of that name exists
- **THEN** `/health` resolves to the health endpoint and the branch is read at `/branches/health/…`

#### Scenario: An operational route cannot be ambiguous with a branch read
- **WHEN** an operational resource sits at the top level with a single `{id}` segment
- **THEN** it resolves unambiguously, branch reads occupying no first segment — so `/system/` groups the operational surface rather than disambiguating it

#### Scenario: A branch cannot shadow a fixed route
- **WHEN** a branch is named `health`, `query` or `claims`
- **THEN** it is addressed under `/branches/{name}/…` and the fixed routes resolve unaffected

#### Scenario: The collection name may also be a branch name
- **WHEN** a branch is named `branches`
- **THEN** `GET /branches` lists the table and `GET /branches/branches/head` reads that branch's head

#### Scenario: A new root resource needs no prefix
- **WHEN** a resource is added at the root
- **THEN** it collides with no branch read, the root namespace carrying no wildcard segment

### Requirement: Operational endpoints are ranke-db extensions beyond the paper
The REST contract SHALL provide operational endpoints beyond the paper's core surface,
marked as such: `GET /health` reporting liveness and the signer/contributor identity
the stack signs with; `GET /system/layers` listing storage layers by **name and type
only**, never connection details or secrets; and the verification run API under
`/system/verifications` — `GET` and `POST /system/verifications`, `GET` and
`DELETE /system/verifications/{reportId}`, and `POST /system/verifications/{reportId}/cancel`
— starting, listing, inspecting, stopping and removing verification runs whose depths
are those of `core-verification`. A run SHALL be asynchronous, returning `202` with the
running report immediately; the stack SHALL cap concurrent runs and answer `429` rather
than stopping a run to make room. Whether these require a privileged subject is decided
by `core-access`, not by this contract.

#### Scenario: Health reports liveness and signer identity
- **WHEN** a client calls `GET /health`
- **THEN** the stack's liveness and its signing contributor identity are returned

#### Scenario: Layer introspection never leaks secrets
- **WHEN** an admin calls `GET /system/layers`
- **THEN** each layer is reported by name and type only, with no connection string or secret

#### Scenario: A verification run is async and pollable
- **WHEN** an admin starts a run via `POST /system/verifications`
- **THEN** a running report is returned immediately with `202` and can be polled via `GET /system/verifications/{reportId}` until it completes or is stopped

#### Scenario: Cancel keeps the report, delete removes it
- **WHEN** an admin cancels a running run and deletes another
- **THEN** the cancelled run's report stays in history with its partial findings, the deleted run's record is gone, and either frees a concurrency slot

### Requirement: Every claim read names one of the three scopes
The REST contract SHALL address each claim read as its scope followed by the claim
within it, one shape serving all three scopes the access model defines:

- `/branches/{branch}/claims/{id}` — confined to that branch's closure, needing **R** on the branch
- `/archive/claims/{id}` — confined to the archive head's closure across every branch, needing **R** on `$archive`
- `/universe/claims/{id}` — reached without any closure, needing **R** on `$universe`

The scope SHALL be an explicit path segment rather than something a reader infers from
what is absent. No path segment SHALL carry a `$`.

The contract SHALL document each scope collection as the reserved scope it reads, naming
it: `/archive/…` reads `$archive`, the closure of the whole Ranke-Archive, and
`/universe/…` reads `$universe`. That is the same scope a RankeQL body names in
`select.branch` and the same target a grant is written against, so a reader sees one
concept across the route, the query language and the access policy rather than three
that happen to correspond.

Naming the correspondence SHALL NOT introduce a second path. Each scope SHALL have
exactly one route; `{branch}` SHALL mean an ordinary branch, so a reserved name supplied
there names a branch that does not exist and is answered as not-found.

#### Scenario: A reader can connect the route to the grant that gates it
- **WHEN** a client reads the contract for `GET /archive/claims/{id}`
- **THEN** it states that the route reads the `$archive` scope, which is what `R $archive` grants and what `select.branch: "$archive"` names

#### Scenario: A reserved name in the branch position is not a second route
- **WHEN** a client requests `/branches/$archive/claims/{id}`
- **THEN** the response is not-found, `{branch}` naming an ordinary branch and each scope having exactly one route

The three SHALL be genuinely distinct in what they reach. A claim the Universe holds but
the current head does not reach SHALL be returned by the Universe route and reported as
not-found by the archive route, since the privileged read exists precisely to reach an
archive by head id alone — restoring from a Universe and a head kept outside the server
(see `core-access`).

#### Scenario: The three scopes read alike in shape
- **WHEN** a client compares the three claim routes
- **THEN** each names a scope and then a claim within it, differing only in which scope

#### Scenario: The Universe reaches beyond the head's closure
- **WHEN** a claim is present in the Universe but outside the archive head's closure
- **THEN** `GET /universe/claims/{id}` returns it and `GET /archive/claims/{id}` reports not-found

#### Scenario: The archive scope spans branches
- **WHEN** a claim lies in the head's closure but in a different branch from the one a client would name
- **THEN** `GET /archive/claims/{id}` returns it without the client naming any branch

#### Scenario: Each scope is separately gated
- **WHEN** an account holds `R $archive` but no `R $universe`
- **THEN** `GET /archive/claims/{id}` is permitted and `GET /universe/claims/{id}` is denied by the access decision

#### Scenario: No path segment carries a reserved name
- **WHEN** the route set is inspected
- **THEN** no path contains a `$`-prefixed segment, those names appearing only in grants and in RankeQL bodies

