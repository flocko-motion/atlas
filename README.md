# RankeDB

**Everything is Knowledge — Knowledge is Everything.**

A provenance-first foundation for knowledge systems.

RankeDB is a single graph organized into three levels — Sources (L0), Cognition (L1), and Semantics (L2) — unified behind one API. Provenance edges form a strict DAG; semantic edges may cycle. Every node carries provenance to its inputs. Nothing is ever deleted.

## Repository structure

| Path | What |
|---|---|
| `go/` | Server (schemaf project), SDK, CLI |
| `go/sdk/` | Worker SDK — typed client for building RankeDB workers |
| `go/cmd/ranke-cli/` | CLI tool for admin and ingestion |
| `frontend/` | Graph Explorer (React + Cytoscape) |
| `papers/` | Research papers |

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

MIT (code) · CC-BY-4.0 (papers)
