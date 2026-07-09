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

- Pin the **client surface to the paper's routes**: `POST /query` (the `core-query`
  tree) and `POST /contribute` for the full read-and-contribute surface, plus the
  cacheable GET subset `GET /{branch}/head`, `GET /{branch}/claim/{id}`, and
  `GET /$universe/claim/{id}`.
- **Remove the client-facing Cypher endpoint.** Cypher/GQL is an internal execution
  engine the planner lowers to (`core-query`, `adapter-storage`), never a REST route.
- Keep the contract **authorization-agnostic**: the endpoint carries the credential
  to core and `core-access` decides; the contract pins no auth scheme and binds no
  per-operation rights. Remove the stale `read ⊂ write ⊂ admin` lattice.
- Fix the **status/error model**: `404` for out-of-closure/unknown (indistinguishable),
  `409` on head conflict, `400`/`401`/`403`/`501`; drop the subject-id "onboarding"
  leakage on `403`.
- Carry claims as **signed CBOR** and content as **raw bytes** on the wire; queries,
  results, and errors as JSON. There is no JSON projection of a claim.
- Keep the **operational endpoints** as explicit ranke-db extensions beyond the
  paper's surface: `GET /health` (liveness + signer identity), storage-layer
  introspection **renamed `/storage/layers` → `/system/layers`** (admin; names and
  types only), and the verification run API (`/verification`, `/verification/{id}`)
  left as-is, referencing `core-verification` for depths.

## Capabilities

### New Capabilities

- `rest-api`: the concrete REST/HTTP binding — routes, methods, wire media types,
  the CRUDA right each operation requires, the status/error model, and the
  operational endpoints — with `openapi/openapi.yaml` as its single source of truth.

## Impact

- **`openapi/openapi.yaml`**: rewritten to this contract; `make generate` then
  regenerates `openapi/openapi.gen.go`, the TS client, and the HTML reference. No
  `*.gen.*` is hand-edited.
- **`adapter-endpoint`**: unchanged at the requirement level — it already pins this
  surface; `rest-api` is its concrete HTTP realisation.
- **Not affected**: `core-query`, `core-contribution`, `core-access`,
  `core-verification` define the semantics `rest-api` binds; it restates none of them.
