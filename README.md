# RankeDB

**Everything is Knowledge — Knowledge is Everything.**

A provenance-first foundation for knowledge systems.

Please visit [github.com/flocko-motion/ranke-graph](https://github.com/flocko-motion/ranke-graph) to learn about the underlying concepts - this repo focusses on the implementation.

RankeDB is the **server**: a hexagonal wrapper around the
[`ranke-go`](https://github.com/rankegraph/ranke-go) library, which owns the graph
model and verification. One process serves exactly one Ranke-Archive, assembled from
exactly one configuration supplied at launch.

## Repository structure

| Path | What |
|---|---|
| `openapi/` | The REST API spec — the single source of truth — and the artifacts generated from it |
| `cmd/ranke-db/` | The server binary: `run <config>`, `verify <config>` |
| `cmd/generator/` | A contributing client that seeds a running instance over the REST API |
| `internal/core/` | The core: endpoints, access, persistence composition, contribution |
| `config/` | Configuration — the composition root |
| `adapters/` | One directory per adapter port: `storage`, `sequencer`, `signer`, `vault`, `auth`, `endpoints` |
| `examples/` | Launchable example configurations |
| `docs/`, `openspec/` | Documentation and capability specs |
| `frontend/` | Ranke Explorer — the browser client (see [`frontend/README.md`](frontend/README.md)) |

## Building

```bash
make            # regenerate from the OpenAPI spec, then build, vet, test and lint
make build      # compile bin/ranke-db and bin/generator
make smoke      # launch the minimal example, seed it over the API, read it back, shut down
```

## Running

An instance is one binary and one config file — there is no runtime reconfiguration.
The admin cycle is edit → run → observe → stop.

```bash
ranke-db verify examples/minimal/config.json   # offline, secret-free check of the config
ranke-db run    examples/minimal/config.json   # resolve secrets, assemble the stack, serve
```

For a dev server with something in it, `make dev SEED=example` launches the minimal
example with a throwaway signing key and seeds it as soon as it answers. `SEED=release`
writes a release process — four signing identities, two packages travelling from a git
snapshot to a signed-off release, and the CVEs their scans mention — and `SEED=chain`
grows a larger archive one contribution at a time. Seeding is a **client** — a
contributor is an application-held key, so `bin/generator` signs its own claims and
sends them to `POST /contribute`.

See [`examples/minimal/`](examples/minimal/) for the smallest launchable stack.

## API

`openapi/openapi.yaml` is the single source of truth for the REST API. `make generate`
produces the Go server interface, the TS client and the HTML + Markdown references from
it; the references are browsable under [`docs/openapi/`](docs/openapi/).

## Papers

The theory lives in the separate [`ranke-graph`](https://github.com/flocko-motion/ranke-graph)
repository as Typst (`.typ`) sources — **not in this repo**. Read the `.typ` source directly,
or run `make docs` to pull copies into `docs/papers/`:

1. [`01-ranke-graph/ranke-graph.typ`](https://github.com/flocko-motion/ranke-graph/blob/main/01-ranke-graph/ranke-graph.typ)
   — foundational model and design philosophy. **Required reading.**
2. [`02-ranke-db/ranke-db.typ`](https://github.com/flocko-motion/ranke-graph/blob/main/02-ranke-db/ranke-db.typ)
   — the RankeDB architecture paper. **Read it before implementation work.**

## License

[Apache 2.0 License](LICENSE)
