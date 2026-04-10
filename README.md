# RankeDB

**Everything is Knowledge — Knowledge is Everything.**

A provenance database for LLM context management.

RankeDB inverts the usual relationship between knowledge graphs and provenance: the provenance DAG *is* the system, and the semantic knowledge graph is a view projected from it. Nothing is ever deleted. Multiple levels of detail are preserved in parallel — from the raw source artifact up to the semantic triplet — networked by provenance, and the strategy of retrieval and reasoning is left to the consumer.

RankeDB is a data structure and a set of invariants, not a deployment topology.

## Architecture

Three storage levels behind a single API:

- **L0 — Raw Sources.** Immutable, content-addressable object store. The archive.
- **L1 — Provenance DAG.** Append-only graph of Thoughts, each derived from Records or other Thoughts by a worker. Node authority: full content, full-text search, vector embeddings, provenance edges.
- **L2 — Semantic Index.** Materialized view of L1 optimized for associative retrieval. Edge authority: semantic relations, cross-domain links, temporal validity. Every node and edge traces back to L1.

## Papers

Drafts in [`papers/`](papers/):

1. [`01-rankedb`](papers/01-rankedb/rankedb.md) — the database architecture, design philosophy, comparative positioning
2. [`02-rankedb-workers`](papers/02-rankedb-workers/rankedb-workers.md) — building a semantic graph from raw data
3. [`03-rankedb-retrieval`](papers/03-rankedb-retrieval/rankedb-retrieval.md) — chat, memory agents, the coordination problem
4. [`04-rankedb-coordination`](papers/04-rankedb-coordination/rankedb-coordination.md) — Stacker: attention allocation in multi-agent chat

## License

MIT (code) · CC-BY-4.0 (papers)
