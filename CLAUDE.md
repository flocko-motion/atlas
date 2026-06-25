# RankeDB — Agent Instructions

## What this repo is

RankeDB is the **server** for the Ranke graph. It is **not** a graph implementation.

- The graph itself — claims, the content-addressed Universe, branches + sequencer,
  signing, verification — lives in the **`ranke-go`** library (separate repo at
  `../ranke-go`, module `github.com/flocko-motion/ranke-go`, wired via the Go
  workspace `go.work`). The old in-Postgres graph (nodes/edges/runs tables,
  L0/L1/L2 levels) has been **purged** — ranke-go owns the model now.
- This repo provides the **server around** that library: a schemaf app exposing a
  thin REST API over a ranke-go `Archive`, plus the control plane (config / vault /
  sequencer adapters, the config-driven assembler, secret-zero sealing, access).

**The boundary, always:** graph + verification = `ranke-go`; server + control plane +
access policy = `ranke-db`. Never reimplement graph logic here — compose the library.

## Read the papers first (obligatory)

The foundational theory lives in the separate
[`ranke-graph`](https://github.com/flocko-motion/ranke-graph) paper repo as Typst
(`.typ`) sources — read them before design work; they are the source of truth.

- **`01-ranke-graph/ranke-graph.typ`** — the foundational model & philosophy. Required.
- **`02-rankedb/rankedb.typ`** — the RankeDB architecture paper (landing soon).

Run **`make docs`** to pull fresh copies into `docs/papers/` (gitignored). **Never
diverge from the papers** — to change a concept, get explicit consensus and update
the paper first.

## Library: ranke-go

`../ranke-go`, wired via `go.work` (`use ../ranke-go`). Provides the data model
(claims, Universe, BranchTableHead, Archive), verification, and the **storage**
(mem/fs/sqlite/s3/minimal/rest) and **sequencer** (mem/file) adapters. ranke-db
composes these via the assembler; it does not implement claims/universe/branches.

## Framework: schemaf

This project uses **schemaf** for all infrastructure (server, the built-in Postgres,
Docker, codegen, frontend embedding, JWT auth). Read its docs before touching server
plumbing:
- README: https://raw.githubusercontent.com/flocko-motion/schemaf/refs/heads/main/README.md
- EXTEND: https://raw.githubusercontent.com/flocko-motion/schemaf/refs/heads/main/EXTEND.md

**Golden rule:** generalize into schemaf; if you hit a schemaf gap, file an issue at
`flocko-motion/schemaf` rather than hacking around it locally.

Key rules:
- **Normative structure:** Go in `go/`, API endpoints in `go/api/`, frontend in `frontend/`.
- **Generated files:** all `*.gen.*` come from `./schemaf.sh codegen` — never hand-edit; commit them.
- **Single binary** gateway (API + frontend), default port 7000.
- **Built-in Postgres** stores the *control plane* (config + access), NOT the graph.
  Our config table is created via schemaf's `RunSet` from `go/adapter/config/postgres`
  (prefix `rankeconfig`, never rename) — not via `go/db/migrations/`.
- **Auth** is schemaf's: JWT over an opaque subject (`api.Subject(ctx)`, `IssueToken`).
- Commands: `./schemaf.sh codegen | dev | run | test`.

## Control-plane (this repo)

- `go/adapter/config` — server-config datatype (`Field` provenance) + `Store` + `Cell`;
  backends `file` (yaml/json/toml/.env, suffix-detected), `env`, `postgres`
  (external via DSN **or** built-in via `NewWithConn(schemafdb.DB)`), `mem`.
- `go/adapter/vault` — secret store: `Vault` (`Secret`/`Signer`) + `SignerFromPEM` +
  `Opener` (declares `NeedsSecret`); backends `env`, `file` (no-security/testing),
  `age` (production: age-encrypted blob held in a config `Cell`).
- `go/seal` — boots LOCKED; secret-zero via env **or** the `/unlock` endpoint
  (rate-limited). Only stood up when the chosen vault's `Opener.NeedsSecret()`.
- **(to build) assembler** — config → ranke-go `Archive`(s); wires `seal.Gate.Open`
  to the configured vault.
- **(to build) thin REST** — schemaf endpoints over the `Archive` + `/unlock`.

## Access policy

schemaf does **authentication**; we do **authorization** only, enforced at the API
(orthogonal to ranke-go verification — anyone can verify regardless). Tiers collapse to
`root` (env-seeded break-glass) / `user` (authenticated) / `anyone` (unauth); everything
else is a grant `(subject, scope, role)` — tenant `{user, admin}`, RA `{read, write,
admin}`. Multi-tenant, tenant-scoped visibility, default-deny, lazy grants, 403 hands a
user their own id. Spec'd under `openspec/`.

## Tooling & shell conventions

- **One command per invocation.** No `&&` / `;` / pipe chaining of unrelated steps —
  separate commands read clearly, fail clearly, and avoid spurious permission prompts.
  (If you find yourself chaining, that's a missing target — add it to a Makefile.)
- **Explore Go with `sindri code map`, not grep** (`--grep`, `--file`, `--depth`).
- **Lint with `sindri lint all`** before considering work done (`sindri lint deadcode|loc`).
- **Verify our packages with `make verify`** (scoped build/vet/fmt/test of `adapter/*` +
  `seal`). Full project: `./schemaf.sh test`.

## Specs

- `openspec/` — OpenSpec change proposals + capability specs (server / access / config behavior).
- Foundational model — the `ranke-graph` paper (above).

## Key files

- `go/main.go` — schemaf entry (minimal; assembler + endpoints wire here).
- `go/adapter/`, `go/seal/` — the control-plane code.
- `../ranke-go` — the library (the graph).
- `openspec/` — specs.
