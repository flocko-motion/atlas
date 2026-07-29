## Why

`openapi/openapi.yaml` is the single source of truth for RankeDB's REST API, but the
current file predates the hexagonal refactor and diverges from the paper (§Endpoints,
§Filtered Reads) and the merged specs on several axes:

- Its authorization model is the purged `read ⊂ write ⊂ admin` lattice with a
  "403 carries subject id (onboarding)" flow, not the paper's **CRUDA rights over
  branch globs** (`core-access`).
- It has no declarative `POST /query`; instead it exposes a client-facing Cypher
  endpoint (`POST /branches/{name}/gql`). The paper makes Cypher/GQL an **internal
  lowering target only** — the client read surface is the select/where/limit tree
  (`core-query`).
- Its query schema predates the normative spec's **RankeQL `Query` type** (§RankeQL,
  now implemented by `ranke-go`) and cannot express it: there is no `select.head`, so
  the closure a paged read pins is unreachable; no `$archive` scope; `min` hops,
  `output.form`, `detail: graph`, an `order` key's `compare`, and the graded
  `execution.report` are all missing; and `output.detail` folds two orthogonal axes
  (shape and detail) into one enum. A binding that cannot express the language it
  binds is the defect, not a simplification of it.
- Its routes (`/branches/{name}/claims/{id}`, per-branch contribute) do not match the
  routes the paper pins, it exposes no `POST /contribute`, no `GET /{branch}/head`,
  and no privileged `$universe` read, which `core-access` and `adapter-endpoint`
  require.

Authorization is **orthogonal** to the REST contract: the endpoint extracts the
credential and passes it to core, and `core-access` decides. The contract therefore
pins neither an auth scheme nor per-operation rights — it removes the stale lattice
rather than re-encoding a new one.

This change specifies the REST/HTTP contract as the paper defines it, so the
`openapi.yaml` can be rewritten to match and regenerated from.

## What Changes

- Pin the **client surface to the paper's routes**: `POST /query` (the RankeQL query)
  and `POST /contribute` for the full read-and-contribute surface, plus the
  cacheable GET subset `GET /{branch}/head`, `GET /{branch}/claim/{id}`, and
  `GET /$universe/claim/{id}`, each claim route carrying a `/content` form so content
  is addressed by the claim that holds it rather than by a raw hash.
- **Bind the RankeQL type field-for-field** (§RankeQL), using its own names and values
  and restating none of its semantics, so every axis of the language is expressible
  and the wire→RQL mapping is a transcription rather than a fold. The two rules the
  JSON schema cannot state — a scope is mandatory, and `$universe` requires a `head` —
  are enforced at the boundary as `400`.
- Frame the result sequence in the binding: `output.encoding: json|cbor` selects the
  per-claim serialization and the response media type frames it as
  `application/json-seq` / `application/cbor-seq`. Verifiability follows the
  *shaping* — `detail: claims` + `form: original` + `encoding: cbor` — not the framing.
- **Remove the client-facing Cypher endpoint.** Cypher/GQL is an internal execution
  engine the planner lowers to (`core-query`, `adapter-storage`), never a REST route.
- Keep the contract **authorization-agnostic**: the endpoint carries the credential
  to core and `core-access` decides; the contract pins no auth scheme and binds no
  per-operation rights. Remove the stale `read ⊂ write ⊂ admin` lattice.
- Fix the **status/error model**: `404` for out-of-closure/unknown (indistinguishable),
  `409` on head conflict, `400`/`401`/`403`/`501`; drop the subject-id "onboarding"
  leakage on `403`.
- Carry a claim fetched **by id** as **signed CBOR** and content as **raw bytes** on
  the wire; queries, result envelopes, and errors as JSON. A by-id read is never a
  projection — but a query result under `encoding: json` is one, and is documented as
  such rather than being claimed verifiable.
- Keep the **operational endpoints** as explicit ranke-db extensions beyond the
  paper's surface: `GET /health` (liveness + signer identity), storage-layer
  introspection **renamed `/storage/layers` → `/system/layers`** (admin; names and
  types only), and the verification run API under **`/system/verification`** (with a
  `cancel` action and a `429` when the concurrency cap is reached), referencing
  `core-verification` for depths. It sits under `/system/` rather than the paper's
  root `/verification…` because a root-level `/verification/{id}` and `/{branch}/head`
  match the same path with neither more specific.

## Capabilities

### New Capabilities

- `rest-api`: the concrete REST/HTTP binding — routes, methods, wire media types, the
  RankeQL request binding and result framing, the status/error model, and the
  operational endpoints — with `openapi/openapi.yaml` as its single source of truth.

### Modified Capabilities

- `core-query`: the query language moves from restated to cited. What remains is what
  the server owes — access-scoped generators, and result-set equivalence under
  lowering — with §RankeQL the one place the language is fixed.

## Impact

- **`openapi/openapi.yaml`**: rewritten to this contract; `make generate` then
  regenerates `openapi/openapi.gen.go`, the TS client, and the two references. No
  `*.gen.*` is hand-edited.
- **`adapters/endpoints/rest_http`**: the wire→RQL mapping becomes a transcription —
  the folding, the `bool|int|string` content cap and its size parser go away, and the
  boundary rules the schema cannot state are enforced there.
- **`cmd/ranke-db`**: `run` now serves the endpoints the config mounted, concurrently,
  each on its own configured address. It previously served a private `/healthz` stub
  on a `--addr` flag and never mounted them, so every route of this contract answered
  `404` on the wire however well the adapter implemented it; the flag goes, since the
  configuration is the composition root. `scripts/smoke.sh` now probes the real
  surface (`/health` routed, a scopeless `POST /query` refused) instead of the stub it
  used to prove nothing with.
- **`internal/core/access`**: `$archive` joins the reserved `$`-targets, so the scope
  RankeQL defines is grantable; without it an `$archive` query could only ever be
  denied. Right-sets for it await the access model's sign-off, as for `$branches` and
  `$sequencer`.
- **`adapter-endpoint`**: unchanged at the requirement level — it already pins this
  surface; `rest-api` is its concrete HTTP realisation.
- **`core-query`**: stops restating the query language. It predated the normative
  §RankeQL chapter and had drifted from it — `select.format`, a single `depth`, the
  content cap under `limit`, a one-key `order`, `execution.trace` — which is what a
  restatement does eventually. Six requirements are removed as duplication of a chapter
  that now fixes them, the lowering requirement is kept (that guarantee is RankeDB's
  own, not the language's), and two requirements are added: the language is §RankeQL's
  and cited rather than repeated, and a generator reads only within the requesting
  account's scope — the one obligation the removed text carried that was genuinely the
  server's. Its `## Purpose` paragraph still describes the capability as defining the
  language; a delta cannot carry Purpose, so that line needs the same edit when this
  change is applied.
- **Not affected**: `core-contribution`, `core-access`, `core-verification` define
  semantics `rest-api` binds; it restates none of them.
