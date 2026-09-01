# RankeDB

**Everything is Knowledge — Knowledge is Everything.**

A provenance-first foundation for knowledge systems.

Please visit [github.com/rankegraph/ranke-graph](https://github.com/rankegraph/ranke-graph) to learn about the underlying concepts - this repo focusses on the implementation.

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
| `docs/`, `openspec/` | The operator's manual (built as `dist/docs.pdf`) and the capability specs |
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

## Handbook

`docs/` is the operator's manual: running an instance, and every field of the launch
artifact — the adapter sections and their backends, the endpoints and their
authenticators, and the account roster that decides what a request may do. `make docs`
fetches the shared ranke-graph typography and builds it to `dist/docs.pdf`, which each
release carries alongside the binaries. It needs [typst](https://github.com/typst/typst) from the
series the repo pins — `make check-tools` reports whether you have it.

The release also carries `ranke-db-docs.tar.gz` (`make docs-bundle`) — the chapter sources plus
`openapi/openapi.gen.yaml`, the self-contained REST contract. One download gives a website its
pages and a client generator its contract, with no clone. It holds what this repository wrote
and nothing fetched: the templates, the papers and `rql.schema.json` are ranke-graph's, and
ranke-graph publishes them.

## Papers

The theory lives in the separate [`ranke-graph`](https://github.com/rankegraph/ranke-graph)
repository as Typst (`.typ`) sources — **not in this repo**. Read the `.typ` source directly,
or run `make docs-papers` to pull copies into `docs/papers/`:

1. [`01-ranke-graph/ranke-graph.typ`](https://github.com/rankegraph/ranke-graph/blob/main/01-ranke-graph/ranke-graph.typ)
   — foundational model and design philosophy. **Required reading.**
2. [`02-ranke-db/ranke-db.typ`](https://github.com/rankegraph/ranke-graph/blob/main/02-ranke-db/ranke-db.typ)
   — the RankeDB architecture paper. **Read it before implementation work.**

## License

[Apache 2.0 License](LICENSE)
