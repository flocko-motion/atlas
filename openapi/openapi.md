---
title: ranke-db API v0.2.0
language_tabs:
  - shell: Shell
  - http: HTTP
  - javascript: JavaScript
  - ruby: Ruby
  - python: Python
  - php: PHP
  - java: Java
  - go: Go
toc_footers: []
includes: []
search: true
highlight_theme: darkula
headingLevel: 2

---

<!-- Generator: Widdershins v4.0.1 -->

<h1 id="ranke-db-api">ranke-db API v0.2.0</h1>

> Scroll down for code samples, example requests and responses. Select a language for code samples from the tabs above or the mobile navigation menu.

REST/HTTP binding for a single **ranke-db** stack: read the verifiable graph
with a declarative query, and contribute signed claims to it.

A server hosts exactly **one stack**, launched from a config file — there is
no tenant or archive routing in the paths.

## Reads are queries

The full read surface is `POST /query`, carrying a **RankeQL** query — the
declarative `Query` type the normative spec fixes (§RankeQL) — as a JSON object:
`select` generates the result set, `where` filters it, `order` sorts it, `limit`
truncates it, and `output` shapes and encodes each surviving claim. This contract
*binds* that type to HTTP and defines none of it; the field names, values and
meanings are the spec's. Cypher/GQL is **never** a client route — it is an
internal execution engine the planner lowers a query to.

A **cacheable GET subset** covers the by-id reads without a query body:
`GET /{branch}/head`, `GET /{branch}/claim/{id}`,
`GET /{branch}/claim/{id}/content`, and the privileged `GET /$universe/claim/{id}` /
`GET /$universe/claim/{id}/content`. Content is addressed by the **claim** that
holds it (not a raw hash), so whether the bytes are inline or a separate blob is
hidden and the read is scoped to the claim's branch. Content also rides **inline**
in query results via `output.content`; the content route fetches the bytes for a
single claim — including the blob an `output.content.overflow: reference` stub
leaves behind.

## Scopes and closures

`select.branch` is the mandatory **scope**: a branch name confines the query to
that branch, `$archive` to the whole Ranke-Archive, and `$universe` applies no
confinement — the privileged by-head-id read, which therefore **requires**
`select.head`. Under every other scope `head` is optional and narrows the read to
that claim's closure, so it can only narrow, never widen past the grant. A closure
is immutable, so pinning `head` gives a paged read a snapshot that cannot shift
while the archive advances.

## Encodings and verifiability

A query returns a **streaming sequence** of results, one item each, in the query's
order. `output.encoding` fixes how each item is serialised and the response media
type frames the sequence:

  - `json` → `application/json-seq` (RFC 7464) — JSON records, content
    base64-encoded when inlined. A **convenience projection**: easy to read and
    debug, but **not** independently verifiable.
  - `cbor` → `application/cbor-seq` (RFC 8742) — binary.

Verifiability is a property of the *shaping*, not of the framing alone:
`detail: claims` + `form: original` + `encoding: cbor` reproduces the canonical
serialization a claim's id is computed over, and is the only combination directly
re-hashable and signature-checkable against that id. Every other shaping is a
rendering for convenience.

A by-id claim GET returns the claim as its signed CBOR (`application/cbor`); a
content GET streams the blob as raw bytes (`application/octet-stream`). In query
results content is instead carried inline (base64 under `json`, byte strings under
`cbor`), capped by `output.content`.

## Credentials and authorization

The contract **binds no per-operation rights** and defines **no** read/write/admin
ladder — authorization is entirely core-access (system accounts with CRUDA rights
over branch globs), decided on the subject the credential resolves to. Every route
accepts the same set of credentials; the shape of a route never varies by right.

What the contract *does* enumerate is how a credential is **presented**, because
the endpoint routes deterministically on the presented scheme to the auth adapter
of that type — no parsing a token to guess its kind:

  - `Authorization: Bearer <jwt>` → the **JWT** adapter
  - `X-API-Key: <key>` → the **API-key** adapter
  - `Authorization: Macaroon <token>` → the **macaroon** adapter
  - **no** credential header → the **noauth** adapter

Which of these adapters is actually mounted is the stack's config; a presentation
with no configured adapter is `401`. (Only if two mechanisms were forced onto one
scheme would the endpoint fall back to trying each — the schemes above are
distinct, so routing stays deterministic.) A `403` body never discloses the
subject id.

## Closure and the 404 rule

Every branch read is bounded by that branch's closure. A claim or content that
exists in the Universe but lies **outside** the named branch's closure returns
`404` — indistinguishable from one that does not exist. Head-id reads that
bypass the branch table are privileged and reached only through the reserved
`$universe` name (`$` is illegal in ordinary branch names).

Base URLs:

* <a href="/">/</a>

# Authentication

- HTTP Authentication, scheme: bearer `Authorization: Bearer <jwt>` → the JWT auth adapter.

* API Key (apikey)
    - Parameter Name: **X-API-Key**, in: header. `X-API-Key: <key>` → the API-key auth adapter.

- HTTP Authentication, scheme: macaroon `Authorization: Macaroon <token>` → the macaroon auth adapter.

<h1 id="ranke-db-api-read">read</h1>

Read the graph — the query surface and the cacheable by-id GETs.

## Read the graph with a declarative query

<a id="opIdquery"></a>

`POST /query`

Runs a RankeQL query (§RankeQL) and streams the result set, one item per
result, in the serialization chosen by `output.encoding`. Results carry the
natural `(created_at, id)` order unless `order` sorts them, with its keys
applied in priority order and that natural order breaking any remaining ties;
to page, carry the last result's order key into the next request's `where`
(pin `select.head` so the closure cannot shift between pages).

When `execution.report` names a verbosity, the **final item** in the sequence
is a `QueryReport` (see the schema) — typed distinctly from result items and
always last — carrying the execution log, timing, and whether a limit truncated
the read. `execution.layer` pins which storage layer runs it.

The response media type mirrors `output.encoding`: `application/json-seq` or
`application/cbor-seq`. A branch named in `select` that does not exist is
`404`; claims outside the scope's closure simply do not appear. `$universe`
is the privileged unconfined read and requires `select.head`.

> Body parameter

```json
{
  "select": {
    "branch": "string",
    "head": "string",
    "claim": "string",
    "path": [
      {
        "edges": [
          "string"
        ],
        "dir": "provenance",
        "min": 0,
        "max": 0,
        "nodes": [
          "string"
        ]
      }
    ]
  },
  "where": {
    "and": [
      {
        "and": []
      }
    ]
  },
  "output": {
    "shape": "single",
    "detail": "id",
    "form": "original",
    "content": {
      "max": 0,
      "overflow": "cutoff"
    },
    "encoding": "json"
  },
  "order": [
    {
      "field": "string",
      "compare": "numeric",
      "dir": "asc"
    }
  ],
  "limit": {
    "results": 0,
    "time": "string"
  },
  "execution": {
    "layer": "string",
    "report": "info"
  }
}
```

<h3 id="read-the-graph-with-a-declarative-query-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|body|body|[Query](#schemaquery)|true|none|

> Example responses

> 200 Response

> 400 Response

```json
{
  "code": "string",
  "error": "string"
}
```

<h3 id="read-the-graph-with-a-declarative-query-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The result sequence. One item per result, plus an optional trailing
`QueryReport` when `execution.report` is set. The content type is the one
matching `output.encoding`.|string|
|400|[Bad Request](https://tools.ietf.org/html/rfc7231#section-6.5.1)|The request was malformed.|[Error](#schemaerror)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required or failed.|[Error](#schemaerror)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Access was denied by core-access. The body carries no subject id or
onboarding hint.|[Error](#schemaerror)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|The branch, claim, or content is unknown — or lies outside the named
branch's closure. The two are indistinguishable.|[Error](#schemaerror)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None, jwt, apikey, macaroon
</aside>

## Current head id of a branch

<a id="opIdgetBranchHead"></a>

`GET /{branch}/head`

Returns the branch's current head id — a moving target (it advances on every
contribution). Cacheable **with revalidation** (weak `ETag`, `Cache-Control:
no-cache`): a conditional request is cheap when the head has not moved. To
inspect the head claim itself, fetch it via `GET /{branch}/claim/{id}`.

<h3 id="current-head-id-of-a-branch-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|branch|path|string|true|The branch name (`$` is reserved and illegal in ordinary names).|

> Example responses

> 200 Response

```json
{
  "head": "string"
}
```

<h3 id="current-head-id-of-a-branch-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The current head id.|[BranchHead](#schemabranchhead)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required or failed.|[Error](#schemaerror)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Access was denied by core-access. The body carries no subject id or
onboarding hint.|[Error](#schemaerror)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|The branch, claim, or content is unknown — or lies outside the named
branch's closure. The two are indistinguishable.|[Error](#schemaerror)|

### Response Headers

|Status|Header|Type|Format|Description|
|---|---|---|---|---|
|200|ETag|string||Weak validator for the current head.|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None, jwt, apikey, macaroon
</aside>

## Fetch a claim within a branch's closure

<a id="opIdgetBranchClaim"></a>

`GET /{branch}/claim/{id}`

Returns claim `{id}` as its signed CBOR bytes, **only if it lies in branch
`{name}`'s closure**. A claim that is superseded, contradicted, or otherwise
outside the closure returns `404`, indistinguishable from one that does not
exist. Immutably **cacheable** by id (strong `ETag`, `Cache-Control: public,
immutable`): the id content-addresses the bytes, so they never change.

<h3 id="fetch-a-claim-within-a-branch's-closure-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|branch|path|string|true|The branch name (`$` is reserved and illegal in ordinary names).|
|id|path|string|true|The content-addressed claim id.|

> Example responses

> 200 Response

> 401 Response

```json
{
  "code": "string",
  "error": "string"
}
```

<h3 id="fetch-a-claim-within-a-branch's-closure-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The claim's signed CBOR bytes.|string|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required or failed.|[Error](#schemaerror)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Access was denied by core-access. The body carries no subject id or
onboarding hint.|[Error](#schemaerror)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|The branch, claim, or content is unknown — or lies outside the named
branch's closure. The two are indistinguishable.|[Error](#schemaerror)|

### Response Headers

|Status|Header|Type|Format|Description|
|---|---|---|---|---|
|200|ETag|string||Strong validator — the claim id.|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None, jwt, apikey, macaroon
</aside>

## Fetch the content of a claim within a branch's closure

<a id="opIdgetBranchClaimContent"></a>

`GET /{branch}/claim/{id}/content`

Streams the content of claim `{id}` — **only if it lies in branch `{name}`'s
closure** (same closure guarantee as the claim itself; out-of-closure or
unknown → `404`). Content is addressed by the claim that holds it, not by a
raw hash: the server resolves whether the bytes live inline in the claim or
in a separate blob, so the client can't tell and doesn't need to — and the
read is scoped to the claim's branch. Immutably **cacheable** by the claim id.

<h3 id="fetch-the-content-of-a-claim-within-a-branch's-closure-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|branch|path|string|true|The branch name (`$` is reserved and illegal in ordinary names).|
|id|path|string|true|The content-addressed claim id.|

> Example responses

> 200 Response

> 401 Response

```json
{
  "code": "string",
  "error": "string"
}
```

<h3 id="fetch-the-content-of-a-claim-within-a-branch's-closure-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The content bytes, streamed.|string|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required or failed.|[Error](#schemaerror)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Access was denied by core-access. The body carries no subject id or
onboarding hint.|[Error](#schemaerror)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|The branch, claim, or content is unknown — or lies outside the named
branch's closure. The two are indistinguishable.|[Error](#schemaerror)|

### Response Headers

|Status|Header|Type|Format|Description|
|---|---|---|---|---|
|200|ETag|string||Strong validator — the claim id.|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None, jwt, apikey, macaroon
</aside>

## Fetch a claim by id from the Universe (privileged)

<a id="opIdgetUniverseClaim"></a>

`GET /$universe/claim/{id}`

Returns claim `{id}` as its signed CBOR bytes directly from the Universe,
bypassing any branch table. This is a **privileged** head-id read, conferred
only through the reserved `$universe` name (see core-access). Immutably
cacheable by id.

<h3 id="fetch-a-claim-by-id-from-the-universe-(privileged)-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|id|path|string|true|The content-addressed claim id.|

> Example responses

> 200 Response

> 401 Response

```json
{
  "code": "string",
  "error": "string"
}
```

<h3 id="fetch-a-claim-by-id-from-the-universe-(privileged)-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The claim's signed CBOR bytes.|string|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required or failed.|[Error](#schemaerror)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Access was denied by core-access. The body carries no subject id or
onboarding hint.|[Error](#schemaerror)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|The branch, claim, or content is unknown — or lies outside the named
branch's closure. The two are indistinguishable.|[Error](#schemaerror)|

### Response Headers

|Status|Header|Type|Format|Description|
|---|---|---|---|---|
|200|ETag|string||Strong validator — the claim id.|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None, jwt, apikey, macaroon
</aside>

## Fetch the content of a claim by id from the Universe (privileged)

<a id="opIdgetUniverseClaimContent"></a>

`GET /$universe/claim/{id}/content`

Streams the content of claim `{id}` directly from the Universe, bypassing
any branch table — a **privileged** read conferred only through `$universe`.
Content is addressed by the claim; whether the bytes are inline or a separate
blob is hidden. Immutably cacheable by the claim id.

<h3 id="fetch-the-content-of-a-claim-by-id-from-the-universe-(privileged)-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|id|path|string|true|The content-addressed claim id.|

> Example responses

> 200 Response

> 401 Response

```json
{
  "code": "string",
  "error": "string"
}
```

<h3 id="fetch-the-content-of-a-claim-by-id-from-the-universe-(privileged)-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The content bytes, streamed.|string|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required or failed.|[Error](#schemaerror)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Access was denied by core-access. The body carries no subject id or
onboarding hint.|[Error](#schemaerror)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|The branch, claim, or content is unknown — or lies outside the named
branch's closure. The two are indistinguishable.|[Error](#schemaerror)|

### Response Headers

|Status|Header|Type|Format|Description|
|---|---|---|---|---|
|200|ETag|string||Strong validator — the claim id.|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None, jwt, apikey, macaroon
</aside>

<h1 id="ranke-db-api-write">write</h1>

Contribute signed claims.

## Contribute signed claims (atomic)

<a id="opIdcontribute"></a>

`POST /contribute`

Contributes one or more signed claims to a branch as a single **atomic**
merge — all are absorbed under one new branch-table head, or none are. The
request body is the claims as **signed CBOR** in ranke's bundle format (a
subgraph; claims reference each other and their closure by content-addressed
id). Content-addressed and therefore **idempotent**: re-contributing yields
the same ids with no duplicates.

The target branch is **required** and given as the `branch` query parameter —
a contribution is always a set of claims merged onto one named branch, and core
checks the **C** right on it before the sequencer merges. It is not in the body:
the body carries only the claims being added, and the sequencer itself creates
the `contribution/branches` branch-table claim that records them (paper 02
§Sequencer), so the target cannot be derived from the submitted bundle. On
success the new branch-table head id and the contributed claim ids are returned.

> Body parameter

<h3 id="contribute-signed-claims-(atomic)-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|branch|query|string|true|The target branch the claims are merged onto (the `C` right applies here).|
|body|body|string(binary)|true|none|

> Example responses

> 201 Response

```json
{
  "head": "string",
  "ids": [
    "string"
  ]
}
```

<h3 id="contribute-signed-claims-(atomic)-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|201|[Created](https://tools.ietf.org/html/rfc7231#section-6.3.2)|The merge committed (or the claims were already present).|[ContributionResult](#schemacontributionresult)|
|400|[Bad Request](https://tools.ietf.org/html/rfc7231#section-6.5.1)|The request was malformed.|[Error](#schemaerror)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required or failed.|[Error](#schemaerror)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Access was denied by core-access. The body carries no subject id or
onboarding hint.|[Error](#schemaerror)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|The branch, claim, or content is unknown — or lies outside the named
branch's closure. The two are indistinguishable.|[Error](#schemaerror)|
|409|[Conflict](https://tools.ietf.org/html/rfc7231#section-6.5.8)|The contribution conflicts with the branch's current head.|[Error](#schemaerror)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None, jwt, apikey, macaroon
</aside>

<h1 id="ranke-db-api-system">system</h1>

Operate the stack — liveness, storage introspection, verification runs.

## Liveness and signer identity

<a id="opIdhealth"></a>

`GET /health`

Reports that the stack is up and the signing/contributor identity it
attests merges under. Whether this requires a privileged subject is a
core-access decision, not part of this contract.

> Example responses

> 200 Response

```json
{
  "status": "ok",
  "signer": "did:key:z6Mk..."
}
```

<h3 id="liveness-and-signer-identity-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The stack is up.|[Health](#schemahealth)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None, jwt, apikey, macaroon
</aside>

## List storage layers

<a id="opIdlistStorageLayers"></a>

`GET /system/layers`

Lists the stack's storage layers (read-through tiers) by **name and type
only** — never connection details or secrets. Naming layers is what lets a
verification run target one directly (a read-through view can mask the loss
of an object on a deeper layer).

> Example responses

> 200 Response

```json
{
  "layers": [
    {
      "name": "string",
      "type": "string"
    }
  ]
}
```

<h3 id="list-storage-layers-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The storage layers, top (cache) to bottom (authoritative).|[StorageLayerList](#schemastoragelayerlist)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required or failed.|[Error](#schemaerror)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Access was denied by core-access. The body carries no subject id or
onboarding hint.|[Error](#schemaerror)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None, jwt, apikey, macaroon
</aside>

## List verification runs

<a id="opIdlistVerifications"></a>

`GET /system/verification`

Verification runs across the whole stack, newest first. Each report is a
**point-in-time record**: a layer repaired externally shows clean in a later
run, so reports accumulate rather than overwrite — until explicitly removed
with `DELETE /system/verification/{reportId}`.

The API lives under `/system/` (rather than the paper's root `/verification…`)
for two reasons: a root-level `/verification/{id}` would collide with
`/{branch}/head` under net/http (both match `/verification/head`, neither more
specific), and a run is a stack-wide operational resource — it roots at a
closure named in the request body, not in the path.

> Example responses

> 200 Response

```json
{
  "reports": [
    {
      "id": "string",
      "config": {
        "closure": "string",
        "layer": "string",
        "depth": "completeness",
        "contentThreshold": 0
      },
      "head": "string",
      "status": "running",
      "startedAt": "2019-08-24T14:15:22Z",
      "completedAt": "2019-08-24T14:15:22Z",
      "claimsChecked": 0,
      "bytesRead": 0,
      "ok": true,
      "failures": [
        {
          "id": "string",
          "mode": "corrupt-bytes",
          "layer": "string",
          "detail": "string"
        }
      ]
    }
  ]
}
```

<h3 id="list-verification-runs-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The verification reports.|[VerificationReportList](#schemaverificationreportlist)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required or failed.|[Error](#schemaerror)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Access was denied by core-access. The body carries no subject id or
onboarding hint.|[Error](#schemaerror)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None, jwt, apikey, macaroon
</aside>

## Start a verification run

<a id="opIdstartVerification"></a>

`POST /system/verification`

Starts a run from a `VerificationConfig`: walk the closure rooted at
`closure` (a branch name — resolved to its current head and **pinned** for
the life of the run — or a head id directly), reading the named `layer`
directly, re-checking each claim to the configured depth. It returns
*findings*, not contents, so it never leaks across closures.

A run may take a **very long time** (hours to days over a large closure at
full-content depth), so it is always **asynchronous**: the call returns `202`
immediately with the running report and a `Location` header pointing at the
report resource. Poll it with `GET /system/verification/{reportId}`, pacing
by the `Retry-After` hint; stop it with `DELETE`. Starting the same run twice
yields two independent point-in-time reports (runs are not deduplicated).

Verification is resource-heavy, so the stack caps the number of runs that may
execute **concurrently** (configured; default 1). When that cap is already
reached the call returns `429` — the server never stops a run to make room. To
proceed, either wait (per `Retry-After`) or `GET /system/verification` to see
what is running and free a slot deliberately: `cancel` a run to stop it while
keeping its report, or `DELETE` it to remove it entirely.

> Body parameter

```json
{
  "closure": "string",
  "layer": "string",
  "depth": "completeness",
  "contentThreshold": 0
}
```

<h3 id="start-a-verification-run-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|body|body|[VerificationConfig](#schemaverificationconfig)|true|none|

> Example responses

> 202 Response

```json
{
  "id": "string",
  "config": {
    "closure": "string",
    "layer": "string",
    "depth": "completeness",
    "contentThreshold": 0
  },
  "head": "string",
  "status": "running",
  "startedAt": "2019-08-24T14:15:22Z",
  "completedAt": "2019-08-24T14:15:22Z",
  "claimsChecked": 0,
  "bytesRead": 0,
  "ok": true,
  "failures": [
    {
      "id": "string",
      "mode": "corrupt-bytes",
      "layer": "string",
      "detail": "string"
    }
  ]
}
```

<h3 id="start-a-verification-run-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|202|[Accepted](https://tools.ietf.org/html/rfc7231#section-6.3.3)|The run started; the running report is returned.|[VerificationReport](#schemaverificationreport)|
|400|[Bad Request](https://tools.ietf.org/html/rfc7231#section-6.5.1)|The request was malformed.|[Error](#schemaerror)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required or failed.|[Error](#schemaerror)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Access was denied by core-access. The body carries no subject id or
onboarding hint.|[Error](#schemaerror)|
|429|[Too Many Requests](https://tools.ietf.org/html/rfc6585#section-4)|The concurrent-run cap is reached. No run was started and none was
cancelled; retry after a running one finishes or is stopped.|[Error](#schemaerror)|

### Response Headers

|Status|Header|Type|Format|Description|
|---|---|---|---|---|
|202|Location|string||The report resource to poll — `/system/verification/{reportId}`.|
|202|Retry-After|integer||Suggested seconds to wait before the first poll.|
|429|Retry-After|integer||Suggested seconds to wait before retrying.|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None, jwt, apikey, macaroon
</aside>

## Show a verification run

<a id="opIdgetVerification"></a>

`GET /system/verification/{reportId}`

The report for a run. Poll until `status` leaves `running`; while it is still
running the response carries a `Retry-After` hint and the progress counters
(`claimsChecked`, `bytesRead`) advance between polls.

<h3 id="show-a-verification-run-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|reportId|path|string|true|The verification report id.|

> Example responses

> 200 Response

```json
{
  "id": "string",
  "config": {
    "closure": "string",
    "layer": "string",
    "depth": "completeness",
    "contentThreshold": 0
  },
  "head": "string",
  "status": "running",
  "startedAt": "2019-08-24T14:15:22Z",
  "completedAt": "2019-08-24T14:15:22Z",
  "claimsChecked": 0,
  "bytesRead": 0,
  "ok": true,
  "failures": [
    {
      "id": "string",
      "mode": "corrupt-bytes",
      "layer": "string",
      "detail": "string"
    }
  ]
}
```

<h3 id="show-a-verification-run-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The verification report.|[VerificationReport](#schemaverificationreport)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required or failed.|[Error](#schemaerror)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Access was denied by core-access. The body carries no subject id or
onboarding hint.|[Error](#schemaerror)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|The branch, claim, or content is unknown — or lies outside the named
branch's closure. The two are indistinguishable.|[Error](#schemaerror)|

### Response Headers

|Status|Header|Type|Format|Description|
|---|---|---|---|---|
|200|Retry-After|integer||While `status` is `running`, suggested seconds before the next poll.|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None, jwt, apikey, macaroon
</aside>

## Delete a verification run

<a id="opIddeleteVerification"></a>

`DELETE /system/verification/{reportId}`

Really deletes the run and its report. If the run is still `running` it is
stopped first, then the record is removed — so this both aborts a run and
cleans up finished history in one step. Unlike the cancel action, nothing
survives: a subsequent `GET` is `404`. Either verb frees a concurrency slot;
cancel keeps the report, delete removes it.

<h3 id="delete-a-verification-run-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|reportId|path|string|true|The verification report id.|

> Example responses

> 401 Response

```json
{
  "code": "string",
  "error": "string"
}
```

<h3 id="delete-a-verification-run-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|204|[No Content](https://tools.ietf.org/html/rfc7231#section-6.3.5)|The run was stopped (if running) and its report deleted.|None|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required or failed.|[Error](#schemaerror)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Access was denied by core-access. The body carries no subject id or
onboarding hint.|[Error](#schemaerror)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|The branch, claim, or content is unknown — or lies outside the named
branch's closure. The two are indistinguishable.|[Error](#schemaerror)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None, jwt, apikey, macaroon
</aside>

## Cancel a running verification run

<a id="opIdcancelVerification"></a>

`POST /system/verification/{reportId}/cancel`

Stops a `running` run but **keeps** its report: the record stays in history
with `status` `stopped` and whatever partial findings it had gathered, and the
concurrency slot is freed. Use this to abort a run you want to keep a record
of; use `DELETE` to remove it entirely. Idempotent — cancelling a run that has
already finished or stopped returns its current report unchanged.

<h3 id="cancel-a-running-verification-run-parameters">Parameters</h3>

|Name|In|Type|Required|Description|
|---|---|---|---|---|
|reportId|path|string|true|The verification report id.|

> Example responses

> 200 Response

```json
{
  "id": "string",
  "config": {
    "closure": "string",
    "layer": "string",
    "depth": "completeness",
    "contentThreshold": 0
  },
  "head": "string",
  "status": "running",
  "startedAt": "2019-08-24T14:15:22Z",
  "completedAt": "2019-08-24T14:15:22Z",
  "claimsChecked": 0,
  "bytesRead": 0,
  "ok": true,
  "failures": [
    {
      "id": "string",
      "mode": "corrupt-bytes",
      "layer": "string",
      "detail": "string"
    }
  ]
}
```

<h3 id="cancel-a-running-verification-run-responses">Responses</h3>

|Status|Meaning|Description|Schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|The run's report after the cancel (status `stopped` if it was running).|[VerificationReport](#schemaverificationreport)|
|401|[Unauthorized](https://tools.ietf.org/html/rfc7235#section-3.1)|Authentication is required or failed.|[Error](#schemaerror)|
|403|[Forbidden](https://tools.ietf.org/html/rfc7231#section-6.5.3)|Access was denied by core-access. The body carries no subject id or
onboarding hint.|[Error](#schemaerror)|
|404|[Not Found](https://tools.ietf.org/html/rfc7231#section-6.5.4)|The branch, claim, or content is unknown — or lies outside the named
branch's closure. The two are indistinguishable.|[Error](#schemaerror)|

<aside class="warning">
To perform this operation, you must be authenticated by means of one of the following methods:
None, jwt, apikey, macaroon
</aside>

# Schemas

<h2 id="tocS_Query">Query</h2>
<!-- backwards compatibility -->
<a id="schemaquery"></a>
<a id="schema_Query"></a>
<a id="tocSquery"></a>
<a id="tocsquery"></a>

```json
{
  "select": {
    "branch": "string",
    "head": "string",
    "claim": "string",
    "path": [
      {
        "edges": [
          "string"
        ],
        "dir": "provenance",
        "min": 0,
        "max": 0,
        "nodes": [
          "string"
        ]
      }
    ]
  },
  "where": {
    "and": [
      {
        "and": []
      }
    ]
  },
  "output": {
    "shape": "single",
    "detail": "id",
    "form": "original",
    "content": {
      "max": 0,
      "overflow": "cutoff"
    },
    "encoding": "json"
  },
  "order": [
    {
      "field": "string",
      "compare": "numeric",
      "dir": "asc"
    }
  ],
  "limit": {
    "results": 0,
    "time": "string"
  },
  "execution": {
    "layer": "string",
    "report": "info"
  }
}

```

A RankeQL read (§RankeQL), evaluated in a fixed logical order: `select`
generates the result set, `where` filters it, `order` sorts it, `limit`
truncates it, and `output` shapes and encodes each surviving claim. An engine
may reorder or lower those steps as long as the delivered result set is
identical.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|select|[Select](#schemaselect)|true|none|A generator in four orthogonal parts: `branch` is the **scope**, `head` the<br>**closure** read, `claim` where the walk **starts**, and `path` the<br>**traversal**. Scope and start are independent because a walk runs both ways:<br>a `uses` step reaches the claims that *cite* the current one, which lie above<br>it, so it is the closure — not the start — that decides a reverse step's<br>answer.|
|where|[Where](#schemawhere)|false|none|A boolean tree. Each node is exactly one of the `and`/`or`/`not` combinators<br>over sub-trees, or a **leaf** naming a `field` and its `test`. Within a<br>`where`, `or` is boolean; across generators it unions whole result sets. A leaf<br>may name any field a claim carries, including the derived `height` (the level a<br>claim sits at above its sources, compared numerically).|
|output|[Output](#schemaoutput)|false|none|Shapes each result along orthogonal axes and fixes the per-claim<br>serialization. `detail: claims` + `form: original` + `encoding: cbor`<br>reproduces the canonical bytes a claim's id is computed over — the only<br>combination directly verifiable against that id.|
|order|[[OrderKey](#schemaorderkey)]|false|none|Sort keys in priority order; absent or empty leaves the natural<br>`(created_at, id)` order.|
|limit|[Limit](#schemalimit)|false|none|Bounds the read; each field's `0` means unbounded.|
|execution|[Execution](#schemaexecution)|false|none|Where the query runs and how it reports on itself (paper 02 §Filtered Reads).<br>Queries are declarative; the planner lowers each to the most capable engine the<br>storage stack offers, so these controls affect execution and diagnostics only,<br>never the result set.|

<h2 id="tocS_Select">Select</h2>
<!-- backwards compatibility -->
<a id="schemaselect"></a>
<a id="schema_Select"></a>
<a id="tocSselect"></a>
<a id="tocsselect"></a>

```json
{
  "branch": "string",
  "head": "string",
  "claim": "string",
  "path": [
    {
      "edges": [
        "string"
      ],
      "dir": "provenance",
      "min": 0,
      "max": 0,
      "nodes": [
        "string"
      ]
    }
  ]
}

```

A generator in four orthogonal parts: `branch` is the **scope**, `head` the
**closure** read, `claim` where the walk **starts**, and `path` the
**traversal**. Scope and start are independent because a walk runs both ways:
a `uses` step reaches the claims that *cite* the current one, which lie above
it, so it is the closure — not the start — that decides a reverse step's
answer.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|branch|string|true|none|The mandatory scope: a branch name (confines the query to that branch),<br>`$archive` (the whole Ranke-Archive), or `$universe` (no confinement —<br>privileged). An empty value is not allowed.|
|head|string|false|none|The closure read: the query sees the closure of this claim and nothing<br>else. **Required** under `$universe`, which confines nothing and so offers<br>no head to fall back on; optional under every other scope, where the<br>scope's own head serves. Given explicitly it must resolve to a claim<br>within the scope's closure, so it narrows the read and can never widen it<br>past the grant.|
|claim|string|false|none|Where the walk starts, which must lie inside the closure; absent, the walk<br>starts at the closure's head. It moves where reading begins, never what is<br>visible.|
|path|[[PathStep](#schemapathstep)]|false|none|The traversal: steps applied in order. A step-less `path` returns the full<br>outward closure of the start claim.|

<h2 id="tocS_PathStep">PathStep</h2>
<!-- backwards compatibility -->
<a id="schemapathstep"></a>
<a id="schema_PathStep"></a>
<a id="tocSpathstep"></a>
<a id="tocspathstep"></a>

```json
{
  "edges": [
    "string"
  ],
  "dir": "provenance",
  "min": 0,
  "max": 0,
  "nodes": [
    "string"
  ]
}

```

One bounded walk: follow `edges` in direction `dir` for between `min` and `max`
hops, optionally constraining what it yields to `nodes` types. `edges` gates
every hop; `nodes` gates only the claims the step yields, never those it passes
through. Each step starts from the **set** of endpoints the previous step
produced — a frontier pipeline — and the no-repeat rule applies within a step
and resets at each step boundary.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|edges|[string]|false|none|Edge-class globs over `class/sub`; a leading `-` on an entry excludes that class.|
|dir|string|false|none|provenance (outgoing, toward references — the default), uses (incoming,<br>the claims that cite this one), or connections (either).|
|min|integer|false|none|Fewest hops; absent means 1, so a step moves at least one hop. `0` also<br>yields the step's own starting set, carrying the frontier through<br>alongside what lies beyond it.|
|max|integer|false|none|Most hops; `0` means unbounded for this step (a step of at most zero hops<br>would move nothing). A `min` above a bounded `max` is rejected.|
|nodes|[string]|false|none|Node-class globs the step may yield; a leading `-` excludes a class.|

#### Enumerated Values

|Property|Value|
|---|---|
|dir|provenance|
|dir|uses|
|dir|connections|

<h2 id="tocS_Where">Where</h2>
<!-- backwards compatibility -->
<a id="schemawhere"></a>
<a id="schema_Where"></a>
<a id="tocSwhere"></a>
<a id="tocswhere"></a>

```json
{
  "and": [
    {
      "and": []
    }
  ]
}

```

A boolean tree. Each node is exactly one of the `and`/`or`/`not` combinators
over sub-trees, or a **leaf** naming a `field` and its `test`. Within a
`where`, `or` is boolean; across generators it unions whole result sets. A leaf
may name any field a claim carries, including the derived `height` (the level a
claim sits at above its sources, compared numerically).

### Properties

oneOf

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|object|false|none|none|
|» and|[[Where](#schemawhere)]|true|none|[A boolean tree. Each node is exactly one of the `and`/`or`/`not` combinators<br>over sub-trees, or a **leaf** naming a `field` and its `test`. Within a<br>`where`, `or` is boolean; across generators it unions whole result sets. A leaf<br>may name any field a claim carries, including the derived `height` (the level a<br>claim sits at above its sources, compared numerically).<br>]|

xor

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|object|false|none|none|
|» or|[[Where](#schemawhere)]|true|none|[A boolean tree. Each node is exactly one of the `and`/`or`/`not` combinators<br>over sub-trees, or a **leaf** naming a `field` and its `test`. Within a<br>`where`, `or` is boolean; across generators it unions whole result sets. A leaf<br>may name any field a claim carries, including the derived `height` (the level a<br>claim sits at above its sources, compared numerically).<br>]|

xor

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|object|false|none|none|
|» not|[Where](#schemawhere)|true|none|A boolean tree. Each node is exactly one of the `and`/`or`/`not` combinators<br>over sub-trees, or a **leaf** naming a `field` and its `test`. Within a<br>`where`, `or` is boolean; across generators it unions whole result sets. A leaf<br>may name any field a claim carries, including the derived `height` (the level a<br>claim sits at above its sources, compared numerically).|

xor

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|*anonymous*|object|false|none|A leaf: one field, one comparison (e.g. {"field": "type", "test": {"glob": "source/*"}}).|
|» field|string|true|none|The field tested — any field a claim carries, or the derived `height`.|
|» test|[Comparison](#schemacomparison)|true|none|A test on one field. Exactly one operator is expected. `in` takes a set;<br>`glob` a shell-style wildcard; the rest a scalar.|

<h2 id="tocS_Comparison">Comparison</h2>
<!-- backwards compatibility -->
<a id="schemacomparison"></a>
<a id="schema_Comparison"></a>
<a id="tocScomparison"></a>
<a id="tocscomparison"></a>

```json
{
  "eq": null,
  "ne": null,
  "lt": null,
  "le": null,
  "gt": null,
  "ge": null,
  "in": [
    null
  ],
  "glob": "string"
}

```

A test on one field. Exactly one operator is expected. `in` takes a set;
`glob` a shell-style wildcard; the rest a scalar.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|eq|any|false|none|none|
|ne|any|false|none|none|
|lt|any|false|none|none|
|le|any|false|none|none|
|gt|any|false|none|none|
|ge|any|false|none|none|
|in|[any]|false|none|none|
|glob|string|false|none|none|

<h2 id="tocS_Output">Output</h2>
<!-- backwards compatibility -->
<a id="schemaoutput"></a>
<a id="schema_Output"></a>
<a id="tocSoutput"></a>
<a id="tocsoutput"></a>

```json
{
  "shape": "single",
  "detail": "id",
  "form": "original",
  "content": {
    "max": 0,
    "overflow": "cutoff"
  },
  "encoding": "json"
}

```

Shapes each result along orthogonal axes and fixes the per-claim
serialization. `detail: claims` + `form: original` + `encoding: cbor`
reproduces the canonical bytes a claim's id is computed over — the only
combination directly verifiable against that id.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|shape|string|false|none|single (each item is one reached endpoint — the default) or path (each item<br>is the route to it, start-first).|
|detail|string|false|none|How each element is carried: id (identities only), graph (nodes joined by<br>the edges between them), or claims (each node with **all** its outgoing<br>edges — the default, and richer than graph).|
|form|string|false|none|Which field values each claim carries: original (as written — a<br>diff-overlaid claim's delta, and the id-defining form) or materialized<br>(with any `contribution/diff` chain resolved). A property of the values,<br>so orthogonal to `detail` and `encoding`.|
|content|[OutputContent](#schemaoutputcontent)|false|none|Inline content per claim. Absent, no content is inlined and a claim carries<br>only its `content_hash`.|
|encoding|string|false|none|How each claim is serialised — json (text; content base64-encoded) or cbor<br>(binary). The same information either way, and orthogonal to the other<br>output fields; the response media type frames the sequence to match<br>(`application/json-seq` / `application/cbor-seq`).|

#### Enumerated Values

|Property|Value|
|---|---|
|shape|single|
|shape|path|
|detail|id|
|detail|graph|
|detail|claims|
|form|original|
|form|materialized|
|encoding|json|
|encoding|cbor|

<h2 id="tocS_OutputContent">OutputContent</h2>
<!-- backwards compatibility -->
<a id="schemaoutputcontent"></a>
<a id="schema_OutputContent"></a>
<a id="tocSoutputcontent"></a>
<a id="tocsoutputcontent"></a>

```json
{
  "max": 0,
  "overflow": "cutoff"
}

```

Inline content per claim. Absent, no content is inlined and a claim carries
only its `content_hash`.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|max|integer|true|none|Cap in bytes on the content inlined per claim.|
|overflow|string|false|none|What happens to content past the cap: cutoff (truncate), omit (drop it),<br>or reference (a `content_hash` stub in its place — the bytes are then<br>fetched from the claim's content route).|

#### Enumerated Values

|Property|Value|
|---|---|
|overflow|cutoff|
|overflow|omit|
|overflow|reference|

<h2 id="tocS_Execution">Execution</h2>
<!-- backwards compatibility -->
<a id="schemaexecution"></a>
<a id="schema_Execution"></a>
<a id="tocSexecution"></a>
<a id="tocsexecution"></a>

```json
{
  "layer": "string",
  "report": "info"
}

```

Where the query runs and how it reports on itself (paper 02 §Filtered Reads).
Queries are declarative; the planner lowers each to the most capable engine the
storage stack offers, so these controls affect execution and diagnostics only,
never the result set.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|layer|string|false|none|Pin execution to a named storage layer — any layer the stack composes —<br>instead of letting the planner choose which layer answers. The engine then<br>follows from that layer's capability (its native walk, or a Cypher/GQL<br>engine if it has one).|
|report|string|false|none|Execution-report verbosity: info (high-level stages), debug (routing and<br>lowering), or trace (per-claim detail). Absent, no report is produced; set,<br>the sequence's **final item** is a QueryReport.|

#### Enumerated Values

|Property|Value|
|---|---|
|report|info|
|report|debug|
|report|trace|

<h2 id="tocS_OrderKey">OrderKey</h2>
<!-- backwards compatibility -->
<a id="schemaorderkey"></a>
<a id="schema_OrderKey"></a>
<a id="tocSorderkey"></a>
<a id="tocsorderkey"></a>

```json
{
  "field": "string",
  "compare": "numeric",
  "dir": "asc"
}

```

One sort key. Keys apply in priority order, claims lacking a key's field sort
last, and the natural `(created_at, id)` order breaks any remaining ties — so
the sort always resolves to a total order and paging stays stable.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|field|string|true|none|The field sorted on — any field a claim carries, or the derived `height`.|
|compare|string|false|none|How the values compare; absent = lexical.|
|dir|string|false|none|Sort direction; absent = asc.|

#### Enumerated Values

|Property|Value|
|---|---|
|compare|numeric|
|compare|lexical|
|dir|asc|
|dir|desc|

<h2 id="tocS_Limit">Limit</h2>
<!-- backwards compatibility -->
<a id="schemalimit"></a>
<a id="schema_Limit"></a>
<a id="tocSlimit"></a>
<a id="tocslimit"></a>

```json
{
  "results": 0,
  "time": "string"
}

```

Bounds the read; each field's `0` means unbounded.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|results|integer|false|none|Maximum number of claims returned; `0` = unbounded.|
|time|string|false|none|Execution budget as a duration (e.g. `"5s"`); the query is cancelled when<br>exceeded. Absent or `"0"` = no budget.|

<h2 id="tocS_QueryReport">QueryReport</h2>
<!-- backwards compatibility -->
<a id="schemaqueryreport"></a>
<a id="schema_QueryReport"></a>
<a id="tocSqueryreport"></a>
<a id="tocsqueryreport"></a>

```json
{
  "startedAt": "2019-08-24T14:15:22Z",
  "elapsedMs": 0,
  "results": 0,
  "truncated": true,
  "events": [
    {
      "atMs": 0,
      "engine": "string",
      "op": "string",
      "level": "error",
      "durationMs": 0,
      "detail": "string",
      "attrs": {}
    }
  ]
}

```

Diagnostic report emitted as the **final item** of a `POST /query` sequence
when `execution.report` names a verbosity. Typed distinctly from result items,
so a reader never mistakes it for data, and always last.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|startedAt|string(date-time)|false|none|Wall clock at query start.|
|elapsedMs|integer|false|none|Total execution time in milliseconds.|
|results|integer|false|none|Number of result items emitted before this report.|
|truncated|boolean|false|none|True if `limit.results` or `limit.time` cut the read short.|
|events|[[QueryEvent](#schemaqueryevent)]|false|none|The ordered, multi-engine execution log, at the requested verbosity.|

<h2 id="tocS_QueryEvent">QueryEvent</h2>
<!-- backwards compatibility -->
<a id="schemaqueryevent"></a>
<a id="schema_QueryEvent"></a>
<a id="tocSqueryevent"></a>
<a id="tocsqueryevent"></a>

```json
{
  "atMs": 0,
  "engine": "string",
  "op": "string",
  "level": "error",
  "durationMs": 0,
  "detail": "string",
  "attrs": {}
}

```

One entry in a query's execution log — a stage, a routing decision, or a
lowering.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|atMs|integer|false|none|Offset from `startedAt` in milliseconds.|
|engine|string|false|none|Who emitted it (e.g. native, cypher, stack, partition).|
|op|string|false|none|What it did (e.g. load-root, step, filter, sort, route, lower-cypher).|
|level|string|false|none|The entry's own level; a report carries everything at or above the<br>verbosity `execution.report` asked for.|
|durationMs|integer|false|none|Elapsed time for a timed step; 0 for a point event.|
|detail|string|false|none|Human-readable message, or the lowered query text (e.g. the Cypher).|
|attrs|object|false|none|Structured extras — layer or shard name, depth, edge and result counts, …|

#### Enumerated Values

|Property|Value|
|---|---|
|level|error|
|level|warn|
|level|info|
|level|debug|
|level|trace|

<h2 id="tocS_ContributionResult">ContributionResult</h2>
<!-- backwards compatibility -->
<a id="schemacontributionresult"></a>
<a id="schema_ContributionResult"></a>
<a id="tocScontributionresult"></a>
<a id="tocscontributionresult"></a>

```json
{
  "head": "string",
  "ids": [
    "string"
  ]
}

```

The outcome of a contribution — the new branch-table head and the appended claim ids.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|head|string|true|none|The new branch-table head id after the merge.|
|ids|[string]|true|none|Content-addressed ids of the contributed claims, in order.|

<h2 id="tocS_BranchHead">BranchHead</h2>
<!-- backwards compatibility -->
<a id="schemabranchhead"></a>
<a id="schema_BranchHead"></a>
<a id="tocSbranchhead"></a>
<a id="tocsbranchhead"></a>

```json
{
  "head": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|head|string|true|none|The content-addressed head claim id.|

<h2 id="tocS_Health">Health</h2>
<!-- backwards compatibility -->
<a id="schemahealth"></a>
<a id="schema_Health"></a>
<a id="tocShealth"></a>
<a id="tocshealth"></a>

```json
{
  "status": "ok",
  "signer": "did:key:z6Mk..."
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|status|string|true|none|none|
|signer|string|false|none|The contributor identity this stack signs merges with.|

<h2 id="tocS_StorageLayer">StorageLayer</h2>
<!-- backwards compatibility -->
<a id="schemastoragelayer"></a>
<a id="schema_StorageLayer"></a>
<a id="tocSstoragelayer"></a>
<a id="tocsstoragelayer"></a>

```json
{
  "name": "string",
  "type": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|name|string|true|none|none|
|type|string|true|none|Adapter type (e.g. memory, filesystem, s3, postgres, neo4j).|

<h2 id="tocS_StorageLayerList">StorageLayerList</h2>
<!-- backwards compatibility -->
<a id="schemastoragelayerlist"></a>
<a id="schema_StorageLayerList"></a>
<a id="tocSstoragelayerlist"></a>
<a id="tocsstoragelayerlist"></a>

```json
{
  "layers": [
    {
      "name": "string",
      "type": "string"
    }
  ]
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|layers|[[StorageLayer](#schemastoragelayer)]|true|none|Read-through tiers, top (cache) to bottom (authoritative).|

<h2 id="tocS_VerificationConfig">VerificationConfig</h2>
<!-- backwards compatibility -->
<a id="schemaverificationconfig"></a>
<a id="schema_VerificationConfig"></a>
<a id="tocSverificationconfig"></a>
<a id="tocsverificationconfig"></a>

```json
{
  "closure": "string",
  "layer": "string",
  "depth": "completeness",
  "contentThreshold": 0
}

```

Parameters for a verification run — the same shape whether declared in the
stack config (scheduled) or posted ad-hoc. Depths are those of
core-verification.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|closure|string|true|none|Root of the closure to verify — a branch name or a claim id.|
|layer|string|false|none|Storage layer (by name) to read directly. Omit to read the composed<br>read-through view — which may mask the loss of an object on a deeper<br>layer, so name a layer to verify it without blind spots.|
|depth|string|false|none|completeness (a `has` sweep), record-correctness (recanonicalise and<br>recheck the id-chain and signatures), or full-content (also re-hash blobs).|
|contentThreshold|integer|false|none|For full-content, the max content size in bytes to re-read and re-hash<br>per claim; larger content is skipped this run. Omit to verify all content.|

#### Enumerated Values

|Property|Value|
|---|---|
|depth|completeness|
|depth|record-correctness|
|depth|full-content|

<h2 id="tocS_VerificationReport">VerificationReport</h2>
<!-- backwards compatibility -->
<a id="schemaverificationreport"></a>
<a id="schema_VerificationReport"></a>
<a id="tocSverificationreport"></a>
<a id="tocsverificationreport"></a>

```json
{
  "id": "string",
  "config": {
    "closure": "string",
    "layer": "string",
    "depth": "completeness",
    "contentThreshold": 0
  },
  "head": "string",
  "status": "running",
  "startedAt": "2019-08-24T14:15:22Z",
  "completedAt": "2019-08-24T14:15:22Z",
  "claimsChecked": 0,
  "bytesRead": 0,
  "ok": true,
  "failures": [
    {
      "id": "string",
      "mode": "corrupt-bytes",
      "layer": "string",
      "detail": "string"
    }
  ]
}

```

A point-in-time record of a verification run; embeds the config that produced it.

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|string|true|none|none|
|config|[VerificationConfig](#schemaverificationconfig)|true|none|Parameters for a verification run — the same shape whether declared in the<br>stack config (scheduled) or posted ad-hoc. Depths are those of<br>core-verification.|
|head|string|true|none|The head id the run verified — the closure it pinned at start. When<br>`config.closure` names a branch, this is the branch's head at start time;<br>the report stays fixed to it even as the branch head moves on.|
|status|string|true|none|running (in progress), complete (finished on its own), stopped (cancelled<br>by an operator via the cancel action — partial findings kept), or error<br>(the run itself failed).|
|startedAt|string(date-time)|true|none|none|
|completedAt|string(date-time)|false|none|none|
|claimsChecked|integer|false|none|Claims checked so far; advances while `status` is `running`.|
|bytesRead|integer|false|none|Content bytes re-read so far; advances while `status` is `running`.|
|ok|boolean|true|none|True when no failures were found in this run.|
|failures|[[VerificationFailure](#schemaverificationfailure)]|false|none|The failures found, accumulating while `status` is `running`. Empty (with<br>`ok: true`) for an intact closure; a healthy archive verifies with none.|

#### Enumerated Values

|Property|Value|
|---|---|
|status|running|
|status|complete|
|status|stopped|
|status|error|

<h2 id="tocS_VerificationReportList">VerificationReportList</h2>
<!-- backwards compatibility -->
<a id="schemaverificationreportlist"></a>
<a id="schema_VerificationReportList"></a>
<a id="tocSverificationreportlist"></a>
<a id="tocsverificationreportlist"></a>

```json
{
  "reports": [
    {
      "id": "string",
      "config": {
        "closure": "string",
        "layer": "string",
        "depth": "completeness",
        "contentThreshold": 0
      },
      "head": "string",
      "status": "running",
      "startedAt": "2019-08-24T14:15:22Z",
      "completedAt": "2019-08-24T14:15:22Z",
      "claimsChecked": 0,
      "bytesRead": 0,
      "ok": true,
      "failures": [
        {
          "id": "string",
          "mode": "corrupt-bytes",
          "layer": "string",
          "detail": "string"
        }
      ]
    }
  ]
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|reports|[[VerificationReport](#schemaverificationreport)]|true|none|[A point-in-time record of a verification run; embeds the config that produced it.]|

<h2 id="tocS_VerificationFailure">VerificationFailure</h2>
<!-- backwards compatibility -->
<a id="schemaverificationfailure"></a>
<a id="schema_VerificationFailure"></a>
<a id="tocSverificationfailure"></a>
<a id="tocsverificationfailure"></a>

```json
{
  "id": "string",
  "mode": "corrupt-bytes",
  "layer": "string",
  "detail": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|id|string|true|none|The claim or object id that failed.|
|mode|string|true|none|corrupt-bytes — stored bytes don't match their hash (storage rot/loss;<br>self-heals via read-through if a deeper layer is intact).<br>invalid-content — the claim itself doesn't validate (e.g. bad signature);<br>unrepairable.|
|layer|string|true|none|The layer where the failure was observed.|
|detail|string|false|none|none|

#### Enumerated Values

|Property|Value|
|---|---|
|mode|corrupt-bytes|
|mode|invalid-content|

<h2 id="tocS_Error">Error</h2>
<!-- backwards compatibility -->
<a id="schemaerror"></a>
<a id="schema_Error"></a>
<a id="tocSerror"></a>
<a id="tocserror"></a>

```json
{
  "code": "string",
  "error": "string"
}

```

### Properties

|Name|Type|Required|Restrictions|Description|
|---|---|---|---|---|
|code|string|true|none|Machine-readable failure category, stable across releases — one of<br>unauthenticated, forbidden, not_found, conflict, busy, invalid,<br>unimplemented, internal. Clients branch on this, not on the message.|
|error|string|true|none|Human-readable message. Carries no subject id, even on 403.|

