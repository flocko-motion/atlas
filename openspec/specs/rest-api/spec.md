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
SHALL pin the cacheable GET subset `GET /{branch}/head`, `GET /{branch}/claim/{id}`,
and `GET /$universe/claim/{id}`. The by-id GET forms SHALL be HTTP-cacheable — the
claim-by-id form immutably, the branch head with revalidation. Each by-id claim route
SHALL carry a `/content` form (`GET /{branch}/claim/{id}/content`,
`GET /$universe/claim/{id}/content`) that streams that claim's content, so content is
addressed by the claim holding it rather than by a raw hash and the read stays scoped
to the claim's branch.

#### Scenario: The full read surface is a query POST
- **WHEN** a client reads via `POST /query` with a RankeQL body
- **THEN** the response is the result set the query defines

#### Scenario: A by-id claim GET is cacheable without a JSON body
- **WHEN** a client issues `GET /{branch}/claim/{id}`
- **THEN** the claim is returned without a query body and is cacheable immutably by id

#### Scenario: The branch head GET is revalidated
- **WHEN** a client issues `GET /{branch}/head`
- **THEN** the current head id (a moving target) is returned as a cacheable-with-revalidation resource

#### Scenario: Content is fetched through the claim that holds it
- **WHEN** a client issues `GET /{branch}/claim/{id}/content`
- **THEN** the claim's content is streamed without the client knowing whether the bytes were stored inline or as a separate blob

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
- **WHEN** a claim exists in the Universe but lies outside branch `{name}`'s closure and is fetched via `GET /{branch}/claim/{id}`
- **THEN** the response is `404`, identical to the response for a claim that does not exist

#### Scenario: A head conflict is a 409
- **WHEN** a contribution conflicts with the branch's current head
- **THEN** the response is `409`

#### Scenario: A client branches on the code, not the message
- **WHEN** any error response is returned
- **THEN** it carries a stable code from a fixed set, and the message is free to change

### Requirement: Reserved top-level path segments do not collide with branches
The REST contract SHALL keep its fixed top-level routes (`/query`, `/contribute`,
`/health`, `/system/…`) unambiguous against branch routes: branch reads always carry a
required suffix segment (`/head`, `/claim/{id}`), and `$universe` begins with `$`, which
is illegal in an ordinary branch name. No branch name SHALL be able to shadow a fixed
route, and no fixed route SHALL be ambiguous against `/{branch}/head` under the
router's matching rules.

#### Scenario: A branch route never shadows a fixed route
- **WHEN** the router resolves `/health` versus `/{branch}/head`
- **THEN** `/health` resolves to the health endpoint and branch reads resolve only with their required suffix

#### Scenario: An operational route cannot be ambiguous with a branch read
- **WHEN** an operational resource would sit at the top level with a single `{id}` segment
- **THEN** it is placed under `/system/` instead, since a root-level `/x/{id}` and `/{branch}/head` match the same path with neither more specific

### Requirement: Operational endpoints are ranke-db extensions beyond the paper
The REST contract SHALL provide operational endpoints beyond the paper's core surface,
marked as such: `GET /health` reporting liveness and the signer/contributor identity
the stack signs with; `GET /system/layers` listing storage layers by **name and type
only**, never connection details or secrets; and the verification run API under
`/system/verification` — `GET` and `POST /system/verification`, `GET` and
`DELETE /system/verification/{reportId}`, and `POST /system/verification/{reportId}/cancel`
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
- **WHEN** an admin starts a run via `POST /system/verification`
- **THEN** a running report is returned immediately with `202` and can be polled via `GET /system/verification/{reportId}` until it completes or is stopped

#### Scenario: Cancel keeps the report, delete removes it
- **WHEN** an admin cancels a running run and deletes another
- **THEN** the cancelled run's report stays in history with its partial findings, the deleted run's record is gone, and either frees a concurrency slot
