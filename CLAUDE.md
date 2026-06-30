# ranke-db — Agent Instructions

## What this repo is

ranke-db is the **server** for the Ranke graph — a slim, **hexagonal** wrapper around
the `ranke-go` library. It launches **exactly one stack** from a config file and serves
it. It is **not** a graph implementation.

**The boundary, always:** graph model + verification = **`ranke-go`**; server, the
adapter ports, config, and access policy = **`ranke-db`**. Never reimplement graph logic
here — compose the library.

## Library: ranke-go (a versioned GitHub module)

Depend on ranke-go as a normal module: `require github.com/flocko-motion/ranke-go vX.Y.Z`,
bumped via semver. **NEVER** wire it as a sibling path, `go.work use`, or `replace` — and
never edit a sibling `ranke-go` checkout. It provides the data model (claims, Universe,
BranchTableHead, Archive), verification, and the **Storage** + **Sequencer** adapters.

## Read the spec FIRST — obligatory before ANY design/API work

The source of truth is the **`ranke-graph`** paper repo (Typst `.typ`). **Read it before
designing — do not reconstruct the model from memory or from this file.** Fetch the raw
sources directly (WebFetch), or `make docs` to pull copies into `docs/papers/` (gitignored):

- **ranke-db — architecture & data model** (read this for the server):
  `https://raw.githubusercontent.com/flocko-motion/ranke-graph/refs/heads/main/02-ranke-db/ranke-db.typ`
- **ranke-graph — foundational model & philosophy**:
  `https://raw.githubusercontent.com/flocko-motion/ranke-graph/refs/heads/main/01-ranke-graph/ranke-graph.typ`

**Never diverge from the papers** — to change a concept, get consensus and update the paper
first. The paper is the user's to write; do not edit it.

**Anchors the spec pins (so you don't misdesign — but still read it):**
- A **claim** is **signed CBOR**, content-addressed by `id = Sign(H(S(claim)))`. The
  **contributor** (an application-held key) signs the claim → its id; the server's separate
  **signing identity** attests the *merge*. So a client knows a claim's id before sending it.
- A **branch table** is itself a claim (`contribution/branches`, edges `contribution/branch`
  → each named graph head); **B_h** = the id of its latest revision. `(Universe, B_h)` = a
  Ranke-Archive. The **Sequencer** advances B_h (keeps its history for rollback). Branches
  are handles, not resources; their heads are moving targets.
- Storage: **Blob store** (`get/put/has`, content-addressed) → typed **Universe** →
  composed via **Stack** (eager/lazy layers) and **Partition** (sharding). "Layers" = a Stack.
- A read is a **filtered query**: the closure of a head, narrowed by a *conjunction of
  filters*, capped by a result limit (conjunctive monotonicity). Out-of-scope refs come back
  as **hash-only stubs** so the subgraph still Merkle-verifies. (Cypher is NOT in the spec.)
- Access is **scope-based**: a base posture (`allow-all`/`deny-all`) + an ordered list of
  wildcard exceptions over field/edge names & types, bound to the authenticated account.
- **Verification** has three depths: completeness (`has` sweep) / record-correctness
  (recanonicalise + recheck id-chain & signatures) / full-content (re-hash blobs).
- Writes carry a **transaction window** `[t_open, t_commit]`; the server merges atomically
  and records the witnessed window in the signed merge (the *hard* timestamp; `created_at`
  is soft/forgeable).
- **Config = the launch artifact**: one JSON doc, `env()`/`vault()` references, age
  encryption; **validate** is offline/secret-free, **resolve** happens only at launch.
- **6 adapter ports**: storage, sequencer, signer, secret-store (vault), authenticator,
  endpoints (transport+authenticator pairs: REST+JWT, MCP+macaroon, …).
- "**One process serves one configuration**" — the process boundary is the design.

## Architecture: the hexagon (6 adapter ports)

One stack, assembled from one config file, no runtime reconfiguration. Admin cycle =
edit config → run → observe → stop → edit → repeat.

- **Driven** (core drives them): **Storage** (𝒰) + **Sequencer** (B_h) — from ranke-go
  (InMemory/FileSystem/S3/Postgres/neo4j and InMemory/FileSystem/Postgres); **Signer**
  (a `crypto.Signer` for contributing claims) + **Vault** (secret KV) — here.
- **Driving** (they drive core): **Auth** (request→subject) + **Endpoints** (REST/HTTP,
  MCP/HTTP) — here.
- Postgres/neo4j/S3/Azure/OpenBao are optional **external plugins**; the default mem/fs
  path needs none.

**Config = the composition root** (`config/`): JSON, with optional whole-file age
encryption (decrypted by core before parse). Any string field may be `env(KEY)` or
`vault(KEY)` — resolved at `run` (env first, then build Vault from the env-only vault
section, then resolve `vault(...)`). `verify` parses + schema-checks only (offline; never
touches env/vault). Accounts + grants are a **static config section**.

**Access** (`access/`): a struct request answered by a checker — `Action ∈ {Read ⊂ Write
⊂ Admin}` (read = query incl. Cypher-read; write = contribute; admin = privileged). No
runtime grant mutation.

## OpenAPI-first

`openapi/openapi.yaml` is the **single source of truth** for the REST API. `make generate`
produces, from it:
- **Go** server interface + models → `api/openapi.gen.go` (oapi-codegen, strict net/http)
- **TS/JS** client → `frontend/src/api/generated/api.gen.ts`
- **HTML** reference → `docs/api/index.html`

Never hand-edit `*.gen.*` or the generated client — change the spec and regenerate.

## Tooling & shell conventions

- **Trust the project's own tooling.** The `make` targets are built and tuned for this
  workflow; their output is concise and meaningful. Run them **directly**.
- **Avoid compound commands.** No `&&` / `;` chaining, and do **not** pipe target output
  through `tail`/`grep`/`head` or redirect it to "manage" it. One command per invocation —
  it reads clearly, fails clearly, and avoids spurious permission prompts. If you feel the
  urge to chain or filter, that's a signal the target is missing or noisy — **fix the
  target**, don't paper over it.
- **`make`** (default) = `generate` then `verify` — the full build (regenerate from the
  spec, then check it).
- **`make verify`** is the green-gate (build + vet + gofmt-check + test), optimized for
  agent use. Run it after changes instead of hand-rolled `go build`/`go test` chains.
- **`make generate`** regenerates everything from the OpenAPI spec.
- **`make check-tools`** reports any missing generation tool (go + node) with install hints.

## Layout

Classical single-module Go repo at the repo root (module `github.com/flocko-motion/rankedb`).

- `openapi/` — the API spec (source of truth) · `api/` — generated Go server (`*.gen.go`)
- `cmd/ranke-db/` — the binary (`run <config>`, `verify <config>`; later `tui`/config edit)
- `core/` · `config/` · `access/` · `port/` · `adapter/<port>/` — the hexagon
- `frontend/` — kept, but **no longer part of ranke-db** (possible future app-layer tooling)
- `docs/`, `openspec/` — docs & specs

Status: mid-refactor on `refactor/hexagonal` — schemaf/tenants/multi-stack purged; the
spec + `make generate`/`verify` pipeline is in place; `core`/adapters/`cmd` are being built.
