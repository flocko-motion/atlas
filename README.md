# RankeDB

**Everything is Knowledge — Knowledge is Everything.**

A provenance-first foundation for knowledge systems.

Please visit [github.com/flocko-motion/ranke-graph](https://github.com/flocko-motion/ranke-graph) to learn about the underlying concepts - this repo focusses on the implementation.

## Repository structure

| Path | What |
|---|---|
| `go/` | Server (schemaf project), SDK, CLI |
| `go/sdk/` | Worker SDK — typed client for building RankeDB workers |
| `go/cmd/ranke-cli/` | CLI tool for admin and ingestion |
| `frontend/` | Graph Explorer (React + Cytoscape) |

## Building workers

See [WORKERS.md](WORKERS.md) — the complete guide for building RankeDB workers using the SDK (`go/sdk/`).

## Running

```bash
./schemaf.sh dev backend,frontend    # dev mode (Postgres + MinIO + server + Explorer)
./schemaf.sh run                     # prod mode (Docker)
```

## Papers

Drafts in [`papers/`](papers/):

1. [`01-rankedb`](papers/01-rankedb/rankedb.md) — architecture, design philosophy

## License

[Apache 2.0 License](LICENSE)
