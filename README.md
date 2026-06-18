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

The theory lives in the separate [`ranke-graph`](https://github.com/flocko-motion/ranke-graph)
repository as Typst (`.typ`) sources — **not in this repo**. Read the `.typ` source directly:

1. [`01-ranke-graph/ranke-graph.typ`](https://github.com/flocko-motion/ranke-graph/blob/main/01-ranke-graph/ranke-graph.typ)
   — foundational model and design philosophy. **Required reading.**
2. [`02-rankedb/rankedb.typ`](https://github.com/flocko-motion/ranke-graph/blob/main/02-rankedb/rankedb.typ)
   — the RankeDB architecture paper. *In progress — landing very soon; read it before implementation work once available.*

## License

[Apache 2.0 License](LICENSE)
