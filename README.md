# Atlas

**Everything is Knowledge — Knowledge is Everything.**

A provenance database for LLM context management.

Atlas inverts the usual relationship between knowledge graphs and provenance: the provenance DAG *is* the system, and the semantic knowledge graph is a view projected from it. Nothing is ever deleted. Classifications, visibility decisions, and the act of thinking itself are all first-class nodes with full derivation history.

## Architecture

Three storage levels behind a single API:

- **L0 — Raw Sources.** Immutable, content-addressable object store. The archive.
- **L1 — Provenance DAG.** Append-only graph of Thoughts, each derived from Records or other Thoughts by a worker.
- **L2 — Semantic Index.** Materialized view of L1 optimized for associative retrieval. Every node and edge traces back to L1.

## Components

- **Atlas DB** — self-contained server (Docker Compose), REST API.
- **Atlas Explorer** — bundled visual interface for navigating provenance chains and the semantic graph.

## Papers

Drafts in [`papers/`](papers/):

1. [`01-atlas-db`](papers/01-atlas-db/atlas-db.md) — the database architecture
2. [`02-atlas-workers`](papers/02-atlas-workers/atlas-workers.md) — building a semantic graph from raw data
3. [`03-atlas-retrieval`](papers/03-atlas-retrieval/atlas-retrieval.md) — chat, memory agents, the coordination problem
4. [`04-atlas-coordination`](papers/04-atlas-coordination/atlas-coordination.md) — Stacker: attention allocation in multi-agent chat

## License

MIT (code) · CC-BY-4.0 (papers)
