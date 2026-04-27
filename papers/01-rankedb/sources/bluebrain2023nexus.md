---
citekey: bluebrain2023nexus
title: "Blue Brain Nexus: An open, secure, scalable system for knowledge graph management and data-driven science"
authors: "Sy, M.F., Roman, B., Kerrien, S., et al."
year: 2022
venue: "Semantic Web Journal"
type: article
---

# Blue Brain Nexus: An open, secure, scalable system for knowledge graph management and data-driven science

**Authors:** Mohameth François Sy, Bogdan Roman, Samuel Kerrien, et al. (EPFL Blue Brain Project)
**Year:** 2022
**Venue:** Semantic Web Journal (IOS Press)
**DOI / URL:** <fill in — ResearchGate 330751750>

## Why this matters for RankeDB

**Closest existing system to RankeDB. The strongest comparison point in §6.**

BBN shares the most architectural DNA with RankeDB of any surveyed system. The critical difference — and the core of RankeDB's novelty claim — is that BBN keeps the knowledge graph primary and adds provenance to it, while RankeDB inverts this: the provenance DAG is the primary data structure and the semantic graph is a materialized view.

## Notes (from full paper reading)

### Architecture overlap with RankeDB

- **Event sourcing / append-only**: "Nexus Delta uses an event sourcing persistence model where all changes to the global state of the system is recorded as a sequence of events." "data are never physically removed from the system, but rather marked as obsolete." Near-identical to RankeDB's immutability invariant.
- **Provenance as first-class concern**: W3C PROV-O for tracking. "Tracking provenance is essential for attribution, quality assessment, and reproducibility."
- **Multi-store architecture**: Cassandra (event store) + Blazegraph (RDF graph) + Elasticsearch (full-text search). Three engines behind one API — like RankeDB's S3 + Postgres + FalkorDB.
- **Files immutable**: "files are immutable once uploaded to the system and future updates represent new revisions."
- **Domain-agnostic**: BBN is a framework, not an application — like RankeDB.
- **Schema-validated**: W3C SHACL for data validation at ingest.
- **Iterative data-driven science cycle**: data discovery → acquisition → preparation → knowledge discovery → dissemination. Parallel to RankeDB's source → derivation → semantic graph pipeline.

### Critical differences (where RankeDB goes further)

- **The inversion is unmade.** BBN's knowledge graph is primary; provenance (PROV-O) is metadata attached to it. RankeDB inverts: the provenance DAG is the primary data structure, the semantic graph is a view. This is the core novelty claim.
- **Full Semantic Web stack.** BBN uses RDF, SPARQL, SHACL, JSON-LD — the complete W3C stack. RankeDB deliberately rejects this in favor of emergent, perspective-local ontology. No global concept definitions.
- **Event sourcing is linear, not a DAG.** BBN's event log is a per-resource sequence of events. RankeDB's provenance is a graph of derivations with combinatorial inputs (N parents per node). The DAG captures richer relationships than a linear event log.
- **No content type taxonomy.** BBN uses SHACL schemas for validation but has no "few types, many encodings" concept. No category/type/encoding triplet.
- **No Merkle-DAG.** BBN has no content-addressing, no hash-based node IDs, no cryptographic integrity chain.
- **No bounded-scope philosophy.** BBN targets large-scale scientific infrastructure (EPFL, neuroscience, 200+ researchers). RankeDB targets personal to SME scale and explicitly bounds its scope.
- **No provenance/consensus separation.** BBN doesn't address this distinction.

### Key quote for §6

From the paper: "The unique challenges of building and simulating the whole rodent brain [...] required a solution to managing large-scale highly heterogeneous data, and tracking their provenance to ensure quality, reproducibility and attribution throughout these iterative cycles."

### Positioning for RankeDB papers

- **P0 (§6 Related Work):** BBN is the strongest comparison. Frame as: "BBN has provenance, immutability, and multi-store architecture. What it lacks is the inversion — in BBN, the knowledge graph is still primary and provenance enriches it. RankeDB proposes the reverse."
- **P1 (Implementation):** BBN's multi-store architecture (Cassandra + Blazegraph + Elasticsearch) is a direct comparison to RankeDB's (S3 + Postgres/AGE + FalkorDB). Different engines, same pattern: specialized stores behind a unified API.
- **P2 (Workers):** BBN's "data preparation" phase (curation, integration, feature extraction, densification, validation) maps to RankeDB's worker pipeline. BBN does this with human-driven workflows; RankeDB automates with LLM workers.
