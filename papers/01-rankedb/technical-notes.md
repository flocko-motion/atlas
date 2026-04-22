# Technical Architecture Notes — Merkle-DAG, Snapshots, Branching

*Captured from design discussion, April 2026. These notes describe implementation-level architecture that extends the concepts in paper 1 (§2, §3, §7.3). To be integrated into the paper and/or a dedicated architecture document.*

---

## 1. Atomic Node+Edge Creation

Nodes are always created atomically together with their edges in a single transaction. Every node creation has:
- **n input edges** (provenance: "derived from these nodes")
- **0..m output edges** (semantics: "relates to these nodes", primarily in L2)
- **1 worker attribution** (who/what produced this)
- **1 content blob** (the actual payload)

Both input and output edges are created at node creation time and are immutable thereafter.

**Two perspectives on the same graph:**
- Ignore output edges → always a DAG (the provenance subgraph). Acyclic by construction because input edges can only point to already-existing nodes.
- Follow output edges too → cycles possible (semantic traversal). "A knows B, B knows A" works because each relation node has output edges to the entities it connects.

This applies uniformly across all levels. L0 nodes have input edges (parent) and no output edges. L1 nodes have input edges (provenance) and rarely output edges. L2 relation nodes have input edges (provenance to the evidence that supports the relation) AND output edges (the semantic relation itself).

The DAG property holds for the input-edge subgraph. Always. Everywhere. Output edges add semantic richness without touching Merkle integrity.

---

## 2. Everything is Content-Addressed (Merkle-DAG)

Every ID in the system is a hash. No sequences, no UUIDs.

**Edge hash:**
```
edge_hash = H(
  source_node_hash  +
  target_node_hash  +
  relation_type     +
  direction         +    // input or output
  ...
)
```

**Node hash = hash of all fields, including edge hashes and content hash:**
```
node_hash = H(
  content_hash      +    // hash of the blob in S3
  content_type      +    // e.g. source/conversation
  encoding          +    // e.g. text/eml
  input_edges[]     +    // array of input edge hashes
  output_edges[]    +    // array of output edge hashes
  created_at        +
  worker_id         +
  ...
)
```

**Why this is a Merkle-DAG:** A node hashes its input edges. Each input edge references its source node by hash. That source node hashes its own input edges. Recursively down to L0 roots (which have no inputs, or a single parent). Change a single node anywhere in the graph → its hash changes → all nodes that reference it as input would have a different hash → inconsistency is immediately detectable.

---

## 3. Snapshots as the Cut Line

Periodically, a **snapshot node** is created. Its inputs are all current heads (nodes with no children) plus the previous snapshot. Its hash witnesses the entire graph state at that point in time.

```
Snapshot_N = H(
  content_hash: H("snapshot at 2026-04-20T23:45:00Z")
  input_edges: [head_1, head_2, ..., head_n, snapshot_N-1]
)
```

The chain of snapshots forms a hashchain: each snapshot includes the previous snapshot's hash as an input. This gives a linear, ordered sequence of graph states.

**Publishing:** The snapshot hash can be published to any external timestamping service — e.g. in the New York Times or a public ledger (Haber & Stornetta, 1991) — to provide third-party proof of graph state at a given point in time.

---

## 4. Three Properties That Usually Conflict

| Property | Mechanism |
|---|---|
| **Immutable** | Append-only DAG, content-addressed nodes |
| **Manipulation-proof** | Merkle-DAG + snapshot hashchain + blockchain notarization |
| **Prune-friendly** | Everything above the last published snapshot is prunable |

**Snapshots are the boundary between immutable and mutable:**
- Everything *below* a published snapshot is irrevocable — the hash is in the blockchain, the Merkle-DAG witnesses every node.
- Everything *above* the last snapshot is workspace — experimental workers run here, buggy runs get pruned here.

**Purge becomes trivial:** "Prune everything since snapshot N" is a single operation. Fork from snapshot N, rebuild with improved workers.

**Publish cadence is policy, not architecture:** How often snapshots are published determines the balance between manipulation-proofing and prune flexibility. Daily? After each audit cycle? After each release? Application-layer decision.

---

## 5. Storage Architecture: Dark S3 + Slim Postgres

### S3 is dark storage
- Nobody lists S3. No `ListBucket`, no enumeration. Access is always by hash: GET/PUT/HEAD.
- Workers never touch S3. They call the API, which handles storage internally.
- Defining invariant: *if it's not in Postgres, it's not there.* Postgres is the sole discovery path.
- Storage abstraction is minimal: any backend supporting content-addressed GET/PUT/HEAD works.

### Postgres holds the entire graph
All nodes (L0 and L1) live in Postgres as rows with metadata, hash, and size. The graph topology (all edges) is in Postgres. S3 holds only byte payloads.

### Postgres is a tunable text cache over S3
- `content_cached` is a `text` column — only text content is cached, never binary.
- Cache eligibility determined by encoding: `WHERE encoding LIKE 'text/%'`.
- Threshold is a runtime config knob. Raise → background fill. Lower → background evict.
- Content-addressed means cache invalidation is impossible — pure LRU.

### Per-node fields
- `content_hash` — H(content), points at S3 blob, also the node ID
- `content_size` — bytes, for cache policy queries
- `content_type` — e.g. `source/conversation`
- `encoding` — MIME-style, e.g. `text/eml`, `image/png`
- `content_cached` — nullable `text`, inlined content when policy allows
- `input_edges[]` — array of edge hashes (provenance)
- `output_edges[]` — array of edge hashes (semantic relations)
- `created_at`, `worker_id`, `origin`, `original_name`, `parent`, etc.

---

## 6. Branching Over Shared Storage

### One S3 blob pool per branch, with read-through

```
S3 main (append-only, published, notarized)
  ↑ read-through fallback
S3 dev (own bucket, only new blobs)
  ↑ read-through fallback
S3 experiment (own bucket, only new blobs)
```

Each branch has its own Postgres instance (or schema) AND its own S3 bucket. Read-through: branch S3 miss → falls through to main S3. Transparent to the API.

### Main S3 grows only at merge

Side branches accumulate blobs in their own S3. Main S3 is never polluted by experimental work. At merge (snapshot publish), missing hashes are copied from side S3 to main S3. Content-addressed means: no duplicate possible, no conflict possible. Pure append.

### Purge = delete side storage

Drop the Postgres branch. Delete the side S3 bucket. Main is untouched. No orphan scan needed.

### Dev workflow (Git-like)

1. Publish snapshot N → blockchain notarization
2. Fork from snapshot N → new Postgres + new S3 bucket (with read-through to main S3)
3. Run experimental workers → new nodes in branch, new blobs in branch S3
4. Evaluate results
5. Good → merge to main: copy Postgres rows, copy missing S3 blobs to main S3
6. Bad → drop branch: delete Postgres, delete S3 bucket. Zero cost.

### Merge is trivial
Nodes are content-addressed. Transferring a node from dev to main means: copy the Postgres row (the hash is the same because the content is the same). Copy the S3 blob if main S3 doesn't have it yet. No conflict resolution needed as long as the inputs exist in the target branch — and they do, because both branches forked from the same snapshot.

### The layering

| Layer | Main | Side Branch |
|---|---|---|
| Blockchain | Snapshot hashes | — |
| Postgres | Production graph | Fork from snapshot N |
| S3 | All published blobs | Only new blobs, read-through to main |

---

## 7. Security Compliance Use Case

This architecture is a natural fit for security compliance provenance: recording all security scans, tool configurations, AI analyses with full prompt/response history, and user decisions as a forensically dense, searchable graph.

Key mappings:
- `source/bulk` → SAST/DAST scan report
- `source/record` → individual scan finding, tool config, ruleset version
- `source/conversation` → AI security review with full prompt/response trail
- `source/data` → CVE snapshot, dependency list, SBOM
- `classification/entity` → identified risk, affected component, responsible person
- `observation/contradiction` → Tool A says "safe", Tool B says "critical" — both stay
- `fact/*` → "Vulnerability X marked resolved on date Y by person Z"

Automated peer review through claim decomposition:
- AI analysis → worker decomposes into individual claims → each claim a `fact/claim` node → dedicated challenge agent per claim → produces `observation/challenge` node (confirmed, refuted, or unverifiable with reasoning)

The Merkle-DAG + snapshot + external timestamping makes the entire audit trail manipulation-proof. In a liability case, the graph proves: we scanned with *this* tool in *this* version with *these* rules, *this* person made *this* decision based on *this* AI analysis with *this* prompt, and the graph had *this* state at *this* time (externally witnessed).

---

## 8. Hash Function Agnosticism

Paper 1 uses `H(x)` to denote the cryptographic hash function, not a specific algorithm. The reference implementation uses SHA-256, but the architecture does not depend on it. The hash algorithm should be configurable, and node IDs could carry a type prefix (e.g. `sha256:a3f2...`) to allow coexistence of different hash functions during migration.

Reference: Haber & Stornetta (1991), "How to Time-Stamp a Digital Document" — the foundational paper on cryptographic timestamping, cited by Satoshi (2008). They demonstrated the concept by publishing hash digests in the New York Times.

---

## 9. Auth-Scoped Visibility and Merkle Compatibility

Auth-scoped visibility (§3.4: a node derived from a confidential source is automatically confidential) is *compatible* with the Merkle-DAG, not in conflict with it.

A user receives a verifiable subgraph: full nodes with content for everything in scope. For branches outside their scope, they see only the hash — enough to verify the integrity of their own subgraph, but no content access. This is exactly how Merkle proofs work in other systems: you don't need the entire tree, only the hashes of the branches you can't see, to verify the branches you can see.

```
[hash_only] ← confidential node, user sees only hash
     ↓
[full node] ← derived, user has access
     ↓
[full node] ← user has access
```

The user can verify: "my subgraph is intact, it builds on a node with hash X whose content I don't know." Integrity is provable without transparency. Only the server sees everything.

This means auth scoping and Merkle integrity are complementary: the Merkle structure is what *enables* verifiable partial views, rather than conflicting with them.
