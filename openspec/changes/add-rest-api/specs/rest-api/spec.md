## ADDED Requirements

### Requirement: OpenAPI is the single source of truth for the REST contract

The REST/HTTP contract SHALL be defined in `openapi/openapi.yaml`, from which the Go
server interface and models, the TS/JS client, and the HTML reference are generated
(`make generate`). Generated artifacts (`*.gen.*`) SHALL NOT be hand-edited; the
contract changes only by editing the spec and regenerating.

#### Scenario: The generated server matches the spec
- **WHEN** `make generate` runs against `openapi/openapi.yaml`
- **THEN** `openapi/openapi.gen.go`, the TS client, and the HTML reference are produced from it, and no generated file is edited by hand

### Requirement: REST binds the paper's read-and-contribute surface

The REST endpoint SHALL expose the full surface as `POST /query` (a `core-query`
select/where/limit query tree) and `POST /contribute` (an atomic contribution of one
or more signed claims), and SHALL pin the cacheable GET subset `GET /{branch}/head`,
`GET /{branch}/claim/{id}`, and `GET /$universe/claim/{id}`. The by-id GET forms SHALL
be HTTP-cacheable — the claim-by-id form immutably, the branch head with revalidation.

#### Scenario: The full read surface is a query POST
- **WHEN** a client reads via `POST /query` with a select/where/limit body
- **THEN** the response is the result set the query defines, per `core-query`

#### Scenario: A by-id claim GET is cacheable without a JSON body
- **WHEN** a client issues `GET /{branch}/claim/{id}`
- **THEN** the claim is returned without a query body and is cacheable immutably by id

#### Scenario: The branch head GET is revalidated
- **WHEN** a client issues `GET /{branch}/head`
- **THEN** the current head id (a moving target) is returned as a cacheable-with-revalidation resource

### Requirement: Cypher is never a client route

The REST contract SHALL NOT expose a client-facing Cypher/GQL endpoint. Cypher/GQL is
an internal execution engine the read planner lowers queries to (`core-query`,
`adapter-storage`); a client expresses reads only through the `POST /query` tree.

#### Scenario: There is no client Cypher endpoint
- **WHEN** the REST surface is inspected
- **THEN** it offers `POST /query` and no route that accepts a raw Cypher/GQL string from a client

### Requirement: Claims and content cross the wire in canonical form

The REST contract SHALL carry claims as their signed CBOR bytes (`application/cbor`)
in both directions, content blobs as raw bytes (`application/octet-stream`), and
queries, result envelopes, and errors as JSON (`application/json`). There SHALL be no
JSON projection of a claim: a read returns the canonical signed CBOR, independently
verifiable and never reconstructed.

#### Scenario: A claim read returns canonical CBOR
- **WHEN** a claim is fetched by id
- **THEN** its signed CBOR bytes are returned, verifiable against the id, not a JSON rendering

#### Scenario: A contribution is submitted as signed CBOR
- **WHEN** a client contributes claims via `POST /contribute`
- **THEN** the body is the signed CBOR of the claim(s) and their referenced closure

### Requirement: The contract is authorization-agnostic

The REST contract SHALL be independent of authorization: the endpoint extracts the
request's credential and passes it to core, and `core-access` decides on the subject's
grants (`adapter-auth`, `adapter-endpoint`, `core-access`). The contract SHALL pin no
authentication scheme and SHALL bind no per-operation rights — authorization is
orthogonal to the wire contract and lives entirely in `core-access`.

#### Scenario: The endpoint carries the credential to core
- **WHEN** any request arrives at a REST route
- **THEN** the endpoint passes its credential to core for the access decision, and the route's shape does not vary by right

#### Scenario: The contract names no auth scheme
- **WHEN** the REST contract is inspected
- **THEN** it fixes no authentication mechanism and no read/write/admin ladder; those are `adapter-auth` and `core-access` concerns

### Requirement: The status and error model

The REST contract SHALL use: `200`/`201`/`202`/`204` for success as fitting the
operation; `400` for a malformed request; `401` when authentication is required or
failed; `403` when access is denied (the decision made in `core-access`); `404` when a
branch, claim, or content is unknown **or lies outside the named branch's closure**,
the two being indistinguishable; `409` when a contribution conflicts with the current
head; and `501` when an optional capability the request needs is not configured. A
`403` body SHALL NOT disclose the subject id or any onboarding hint.

#### Scenario: Out-of-closure is indistinguishable from absent
- **WHEN** a claim exists in the Universe but lies outside branch `{name}`'s closure and is fetched via `GET /{branch}/claim/{id}`
- **THEN** the response is `404`, identical to the response for a claim that does not exist

#### Scenario: A head conflict is a 409
- **WHEN** a contribution conflicts with the branch's current head
- **THEN** the response is `409`

### Requirement: Reserved top-level path segments do not collide with branches

The REST contract SHALL keep its fixed top-level routes (`/query`, `/contribute`,
`/health`, `/system/…`, `/verification…`) unambiguous against branch routes: branch
reads always carry a required suffix segment (`/head`, `/claim/{id}`), and `$universe`
begins with `$`, which is illegal in an ordinary branch name. No branch name SHALL be
able to shadow a fixed route.

#### Scenario: A branch route never shadows a fixed route
- **WHEN** the router resolves `/health` versus `/{branch}/head`
- **THEN** `/health` resolves to the health endpoint and branch reads resolve only with their required suffix

### Requirement: Operational endpoints are ranke-db extensions beyond the paper

The REST contract SHALL provide operational endpoints beyond the paper's core
surface, marked as such: `GET /health` reporting liveness and the signer/contributor
identity the stack signs with; `GET /system/layers` listing storage layers by **name
and type only**, never connection details or secrets; and the verification run API —
`GET`/`POST /verification` and `GET`/`DELETE /verification/{id}` — starting, listing,
inspecting, and stopping verification runs whose depths are those of
`core-verification`. Whether these require a privileged subject is decided by
`core-access`, not by this contract.

#### Scenario: Health reports liveness and signer identity
- **WHEN** a client calls `GET /health`
- **THEN** the stack's liveness and its signing contributor identity are returned

#### Scenario: Layer introspection never leaks secrets
- **WHEN** an admin calls `GET /system/layers`
- **THEN** each layer is reported by name and type only, with no connection string or secret

#### Scenario: A verification run is async and pollable
- **WHEN** an admin starts a run via `POST /verification`
- **THEN** a running report is returned immediately and can be polled via `GET /verification/{id}` until it completes or is stopped
