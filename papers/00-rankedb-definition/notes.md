# P0: RankeDB — The Data Structure

*Notes for the formal definition paper. P0 defines the abstract data structure and its invariants. No implementation details, no stack choices, no code.*

---

## Scope

P0 is the theoretical foundation. It defines:
- The graph as an abstract data structure (nodes, edges, three regions)
- Content type taxonomy and encoding scheme
- Formal invariants
- Merkle-DAG property with proofs
- Snapshot hashchain
- Philosophical grounding (provenance ≠ consensus, bounded scope, attributed claims)

P0 does NOT cover: Postgres, S3, FalkorDB, Docker, API design, worker implementations, branching, caching. Those belong in P1 (implementation).

---

## Formal Results

### 1. DAG property under circular semantics

Let G = (V, E_in ∪ E_out) be the graph. Every edge e has a parent (the node that created it) and a target (the node it points to). Edges are created atomically with their parent node.

- **Input edge (direction = in):** "my parent was derived from target." Provenance.
- **Output edge (direction = out):** "my parent asserts something about target." Semantics.

Define the provenance subgraph G_p = (V, E_in).

**Theorem:** G_p is acyclic.

**Proof:** Every node v has a creation time t(v). Input edges can only target nodes that existed before v was created: for every edge (u, v) ∈ E_in where v is the parent, t(target(e)) < t(v). This establishes a strict partial order on V by creation time. A strict partial order admits no cycles. ∎

Output edges are not subject to this constraint — they can target any existing node, including "older" neighbors. Therefore G may contain cycles (through output edges), but G_p cannot.

**Corollary:** The provenance subgraph G_p is always a DAG, regardless of the semantic richness of output edges. Circular semantics (A knows B, B knows A) are modeled by two separate relation nodes, each with output edges, but the provenance subgraph (input edges only) remains acyclic.

### 2. Merkle integrity

Every ID in the system is a cryptographic hash H.

Edge hash:
```
h(e) = H(parent(e) ‖ target(e) ‖ type(e) ‖ direction(e))
```

Node hash:
```
h(v) = H(content_hash(v) ‖ content_type(v) ‖ encoding(v) ‖ h(e₁) ‖ ... ‖ h(eₙ) ‖ created_at(v) ‖ worker_id(v))
```

where e₁...eₙ are all edges (input and output) created with v.

**Theorem:** Manipulation of any node v' in the provenance subgraph of v changes h(v).

**Proof:** By induction on the depth of the DAG.

*Base case:* v' = v. Changing any field of v changes h(v) directly (H is collision-resistant). ✓

*Inductive step:* v' is an ancestor of v in G_p. There exists a path v' → ... → u → v in G_p (following input edges). By inductive hypothesis, manipulation of v' changes h(u). h(u) is the target hash used in the computation of some input edge e of v. Changing h(u) changes h(e). Changing h(e) changes h(v) (since h(e) is part of v's hash computation and H is collision-resistant). ∎

**Corollary:** Each node hash witnesses the integrity of its entire provenance subgraph. Tampering anywhere below is detectable at the root.

### 3. Content-addressing and idempotency

**Theorem:** Identical content with identical provenance produces identical node hashes.

∀ v₁, v₂: fields(v₁) = fields(v₂) → h(v₁) = h(v₂)

Since node ID = node hash, identical nodes are the same node. Writes are idempotent by construction. Deduplication is free.

### 4. Snapshot hashchain

Snapshots are special nodes whose inputs are all current heads (nodes with no children in G_p) plus the previous snapshot.

```
s₀ = H(heads(G_p, t₀))
sₙ = H(heads(G_p, tₙ) ‖ sₙ₋₁)
```

The snapshot sequence (s₀, s₁, ..., sₙ) is a hashchain. Each snapshot witnesses the graph state AND all previous snapshots. Manipulation of any sᵢ invalidates all sⱼ for j > i.

Snapshot hashes can be published to any external timestamping service — e.g. in the New York Times or a public ledger (Haber & Stornetta, 1991) — to provide third-party proof of graph state at a given point in time.

---

## Edge Structure

Every edge belongs to exactly one node: the node that created it (its parent).

```
edge = {
  parent:     hash_of_creating_node,
  target:     hash_of_target_node,
  type:       relation_type,
  direction:  in | out
}
```

- **Input edge (in):** parent is the new node, target is an older node that contributed to the parent's creation. Provenance direction.
- **Output edge (out):** parent is the new node, target is an older node that the parent asserts something about. Semantic direction.

The direction flag is what separates the provenance subgraph (acyclic, Merkle-secured) from the semantic layer (potentially cyclic, expressive). Both coexist in the same graph. Both are immutable once created. Both are hashed into the parent node's ID.

---

## Node Creation is Atomic

A node and all its edges are created in a single atomic transaction:
- n input edges (provenance: sources and derivations)
- m output edges (semantics: relations asserted)
- 1 content blob (the payload, stored separately, referenced by content_hash)
- 1 worker attribution

Nothing can be added to a node after creation. No edge can be added later. The node's hash covers everything it will ever have. This is what makes the Merkle property hold: h(v) is final at creation time.

---

## Hash Function Agnosticism

P0 uses H(x) to denote the cryptographic hash function, not a specific algorithm. The reference implementation (P1) specifies the concrete choice (e.g. SHA-256). The architecture allows:
- Hash type prefix on IDs: `sha256:a3f2b7c...`
- Coexistence of different hash functions during migration
- Future migration to post-quantum hash functions

Reference: Haber & Stornetta (1991), "How to Time-Stamp a Digital Document" — the foundational paper on cryptographic timestamping. They demonstrated the concept by publishing hash digests in the New York Times.

---

## Auth-Scoped Visibility and Merkle Compatibility

Auth-scoped visibility (a node derived from a confidential source is automatically confidential) is compatible with the Merkle-DAG.

A user receives a verifiable subgraph: full nodes with content for everything in scope. For branches outside their scope, they see only the hash — enough to verify the integrity of their own subgraph, but no content access.

```
[hash_only] ← confidential node, user sees only hash
     ↓
[full node] ← derived, user has access
     ↓
[full node] ← user has access
```

The user can verify: "my subgraph is intact, it builds on a node with hash X whose content I don't know." Integrity is provable without transparency. Only the server sees everything.

Merkle structure is what *enables* verifiable partial views. Auth scoping and Merkle integrity are complementary.

---

## Content Type Taxonomy

(Defined in detail in the current paper 1 §2.1 and §2.2. To be moved to P0.)

Three-part identifier: `content_type` + `encoding`.

- `content_type` = `category/type` (RankeDB-defined categories, application-extendable types)
- `encoding` = MIME-style `class/format` (application-defined, cache policy falls out of class prefix)

### Source types (Level 0)
- `source/conversation` — communicative act with sender/receiver
- `source/media` — perceptual capture, content opaque until processed
- `source/record` — machine/objective observation of world-state
- `source/data` — structured information (defined by exclusion)
- `source/bulk` — container of other sources

### Derivation types (Level 1 — the cognitive layer)
Every L1 node is a thought — the output of a worker interpreting the graph.

**Resolved forms:** `conversation/*`, `image/*`, `video/*`
**Cognitive derivations:** `classification/*`, `observation/*`, `summary/*`, `fact/*`

### Encoding
MIME-style: `text/eml`, `image/png`, `audio/wav`, `application/pdf`.
- `text/*` → eligible for inline caching (P1 implementation detail)
- Everything else → binary, stored in content-addressed blob store only

### Design principle
*Few types, many encodings.* The diversity of the world lives in encodings, not in the type system. Each encoding is a micro-project: a parser.

---

## Philosophical Grounding

(Quotes and design notes collected in papers/01-rankedb/quotes.md. Key points for P0:)

### Provenance and consensus are orthogonal problems
- **Provenance** = attribution (who said what, when, on what basis). Solvable by construction.
- **Consensus** = social agreement on what to trust. Human process, not database architecture.
- RankeDB handles provenance. Consensus is downstream, built by consumers on top.

### Bounded scope: personal to small-enterprise
- At bounded scale, trust is pre-established, ontology is finite, adversarial resistance is simple.
- RankeDB does not aim for Wikipedia-scale global consensus.

### Thesis
**RankeDB stores attributed claims; common truth is what consumers build on top when they want it.**

---

## Paper Structure (planned)

| Paper | Title | Scope |
|---|---|---|
| **P0** | The Data Structure | Abstract definition, invariants, formal proofs, philosophy |
| **P1** | The Implementation | Reference stack, dark S3, caching, branching, snapshots, compliance |
| **P2** | Workers | Pipeline, dispatch, reactive/analytical, claim decomposition |
| **P3** | Retrieval | Memory agents, verification gate, conviction, user confirmation |
| **P4** | Chat Frontend | Stacker, multi-agent coordination, user interface |
