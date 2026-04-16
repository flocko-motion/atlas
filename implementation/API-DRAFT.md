# RankeDB API — Draft

Working draft. Not a spec — a starting point for implementation.

Edge types: `provenance/input`, `provenance/worker`, `relation/head`, `relation/tail`.
Edges always connect nodes to nodes.

---

## URL hierarchy

The graph is one collection. Paths are views into it.

```
/api/nodes                 — all nodes (power-user catch-all)
/api/nodes/sources         — L0 sources
/api/nodes/cognition       — L1 cognitive derivations
/api/nodes/entities        — L2 entities
/api/nodes/relations       — L2 relations

/api/edges                 — all edges
/api/edges/provenance      — provenance edges (provenance/input + provenance/worker)
/api/edges/relations       — semantic edges (relation/head + relation/tail)

/api/content/:sha256       — raw content bytes by hash (read-only, no POST)

/api/workers/runs          — worker run registration
/api/workers/queue         — unprocessed nodes for a worker
```

POST on node paths creates a node (with edges). The specialized paths pre-fill `level` and `content_class` and add path-specific validation.
GET on any path lists/filters the corresponding subset. Same query params work everywhere.
Edges are created via node creation (atomic), not via standalone POST on edge paths.

---

## Creating nodes

### POST /api/nodes

Power-user catch-all. Creates any node with any edges.

**JSON body:**

```json
{
  "level": 2,
  "content_class": "relation",
  "content_type": "family",
  "encoding_class": "text",
  "encoding_format": "plain",
  "content": "sister of",
  "run_id": "run-789",
  "valid_from": "2000-01-01T00:00:00Z",
  "valid_from_blur": "365d",
  "edges": [
    { "type": "provenance/input", "target_node_id": "fact-node-F" },
    { "type": "provenance/worker", "target_node_id": "worker-config-T" },
    { "type": "relation/tail", "target_node_id": "alice-123", "confidence": 1.0 },
    { "type": "relation/head", "target_node_id": "bob-456", "confidence": 1.0 }
  ]
}
```

**Multipart (for binary/large content):**

File in one part, metadata JSON (same structure minus `content`) in another.

**Validation:**
- L0 roots: no edges required, idempotent via `content_sha256` (deterministic ID).
- Everything else: at least one `provenance/input` edge, exactly one `provenance/worker` edge.
- `relation/head` and `relation/tail` edges optional (only meaningful for relation nodes).
- All edges carry the provided `run_id`.

**Returns:** the created node with its ID and all edges.

### POST /api/nodes/sources

Convenience for L0 source ingestion. Pre-fills `level: 0`, `content_class: "source"`.

Two creation modes:

**With payload (normal ingestion):** Multipart form — file upload + metadata fields (`content_type`, `encoding_class`, `encoding_format`, `origin`, `original_name`, `artifact_created_at`, `artifact_created_at_blur`). API stores the blob in S3 and creates the node. Idempotent for root sources: same bytes → same node (deterministic ID from content hash).

**By reference (recovery):** JSON body with `content_sha256` + metadata fields, no file. For when a node was lost (pruning, reset, crash, fork) but the blob is still in S3. API verifies the blob exists via HEAD to S3, fills `content_len` from the response, and creates the node pointing to the existing blob. This is a recovery mechanism — blobs only reach S3 through normal node creation, never via standalone upload.

No edges for root sources. For derived sources (format conversions, normalizations), include provenance edges in the request.

### POST /api/nodes/cognition

Convenience for L1 derivations. Pre-fills `level: 1`.

```json
{
  "content_class": "classification",
  "content_type": "entity",
  "encoding_class": "text",
  "encoding_format": "plain",
  "content": "Person: Alice Müller",
  "run_id": "run-789",
  "edges": [
    { "type": "provenance/input", "target_node_id": "source-node-S" },
    { "type": "provenance/worker", "target_node_id": "worker-config-T" }
  ]
}
```

Validation: at least one `provenance/input`, exactly one `provenance/worker`.

### POST /api/nodes/entities

Convenience for L2 entity creation. Pre-fills `level: 2`, `content_class: "entity"`.

```json
{
  "content_type": "person",
  "encoding_class": "text",
  "encoding_format": "plain",
  "content": "Alice Müller",
  "run_id": "run-789",
  "edges": [
    { "type": "provenance/input", "target_node_id": "classification-node-C" },
    { "type": "provenance/worker", "target_node_id": "worker-config-T" }
  ]
}
```

### POST /api/nodes/relations

Convenience for L2 relation creation. Pre-fills `level: 2`, `content_class: "relation"`.
Accepts `heads` and `tails` as top-level fields instead of raw edges (sugar).

```json
{
  "content_type": "family",
  "encoding_class": "text",
  "encoding_format": "plain",
  "content": "sister of",
  "run_id": "run-789",
  "valid_from": "2000-01-01T00:00:00Z",
  "valid_from_blur": "365d",
  "tails": [
    { "entity_id": "alice-123", "confidence": 1.0 }
  ],
  "heads": [
    { "entity_id": "bob-456", "confidence": 1.0 }
  ],
  "inputs": ["fact-node-F"],
  "worker": "worker-config-T"
}
```

Reading convention: **"tail IS relation TOWARDS head"** — Alice(tail) is sister_of towards Bob(head).

Validation: at least one input, exactly one worker. Heads and tails may be empty (structural unknowns).

---

## Reading nodes

### GET /api/nodes/:id

Returns full node metadata. Content is inline if cached (`content_cached` populated); otherwise the client should fetch via `/content`.

### GET /api/nodes/:id/content

Returns raw content bytes. Served from Postgres cache if available; otherwise proxied from S3.

### GET /api/nodes/:id/provenance

Returns the upstream provenance chain from this node back to L0 roots. Traverses `provenance/input` and `provenance/worker` edges recursively. Response is a subgraph (nodes + edges).

### GET /api/nodes/:id/edges

Returns edges connected to a node. Query parameters:

| Parameter   | Type   | Description                                                                           |
| ----------- | ------ | ------------------------------------------------------------------------------------- |
| `direction` | string | `incoming`, `outgoing`, or `both`                                                     |
| `type`      | string | `provenance/input`, `provenance/worker`, `relation/head`, `relation/tail`, or `all`   |
| `limit`     | int    | Pagination limit                                                                      |
| `offset`    | int    | Pagination offset                                                                     |

---

## Listing and filtering

All listing endpoints accept the same query parameters:

| Parameter         | Type     | Description                                    |
| ----------------- | -------- | ---------------------------------------------- |
| `content_sha256`  | string   | Filter by content hash                           |
| `content_class`   | string   | e.g. `source`, `entity`, `relation`            |
| `content_type`    | string   | e.g. `conversation`, `person`, `family`        |
| `encoding_class`  | string   | e.g. `text`, `image`                           |
| `encoding_format` | string   | e.g. `plain`, `eml`, `png`                     |
| `created_after`   | datetime | Lower bound on `created_at`                    |
| `created_before`  | datetime | Upper bound on `created_at`                    |
| `run_id`          | string   | Filter by worker run                           |
| `limit`           | int      | Pagination limit                               |
| `offset`          | int      | Pagination offset                              |

```
GET /api/nodes                — all nodes
GET /api/nodes/sources        — pre-filtered to level=0, content_class=source
GET /api/nodes/cognition      — pre-filtered to level=1
GET /api/nodes/entities       — pre-filtered to level=2, content_class=entity
GET /api/nodes/relations      — pre-filtered to level=2, content_class=relation
```

### GET /api/nodes/relations (additional params)

| Parameter    | Type   | Description                                                              |
| ------------ | ------ | ------------------------------------------------------------------------ |
| `unresolved` | bool   | Relations with 0 or >1 tails (open questions, ambiguity)                 |
| `type`       | string | e.g. `alias`, `family`, `has_role`                                       |

### GET /api/nodes/entities/:id

Returns entity node + all connected relation nodes with their head/tail edges, sorted by temporal validity, filtered by minimum confidence (query param `min_confidence`, default 0).

### GET /api/nodes/entities/:id/timeline

Returns all relations of an entity sorted chronologically by `valid_from`.

---

## Content

```
HEAD /api/content/:sha256        — existence check + Content-Length header
GET  /api/content/:sha256        — raw content bytes by hash
```

### HEAD /api/content/:sha256

Returns 200 with `Content-Length` header if the blob exists, 404 if not. No body. This is the ingest worker's "should I upload?" check — one round-trip, zero bytes transferred.

### GET /api/content/:sha256

Returns raw content bytes. Served from Postgres cache if available; otherwise proxied from S3. The response `Content-Type` header reflects the encoding of the content.

Content is addressed by hash, independent of nodes. Multiple nodes may share the same hash — one blob serves all of them.

---

## Edges

```
GET /api/edges                    — all edges (filterable)
GET /api/edges/provenance         — provenance edges only (provenance/input + provenance/worker)
GET /api/edges/relations          — semantic edges only (relation/head + relation/tail)
GET /api/edges/:id                — single edge metadata
```

### GET /api/edges

All listing endpoints accept the same query parameters:

| Parameter        | Type   | Description                                                                         |
| ---------------- | ------ | ----------------------------------------------------------------------------------- |
| `type`           | string | `provenance/input`, `provenance/worker`, `relation/head`, `relation/tail`, or `all` |
| `source_node_id` | string | Edges originating from this node                                                    |
| `target_node_id` | string | Edges pointing to this node                                                         |
| `run_id`         | string | Filter by worker run                                                                |
| `min_confidence` | float  | Minimum confidence (for semantic edges)                                             |
| `limit`          | int    | Pagination limit                                                                    |
| `offset`         | int    | Pagination offset                                                                   |

Deeper paths are tighter pre-filters:

```
/api/edges/provenance              — type IN (provenance/input, provenance/worker)
/api/edges/provenance/inputs       — type = provenance/input
/api/edges/provenance/workers      — type = provenance/worker
/api/edges/relations               — type IN (relation/head, relation/tail)
```

All accept the same query params as GET /api/edges. Each deeper path just narrows the type filter.

Note: filtering by relation TYPE (e.g. "all family relations") is a node-level query — the type lives on the relation node, not on the edges: `GET /api/nodes/relations?content_type=family`.

### GET /api/edges/:id

Returns single edge: source_node_id, target_node_id, type, run_id, confidence, created_at.

Edges are created via POST /api/nodes (as part of the edges array or heads/tails fields). There is no standalone POST /api/edges.

---

## Workers

### POST /api/workers/runs

Registers a new worker run. The worker must first have created a `worker/config` node (content_class: `worker`, content_type: `config`, encoding: `text/json`) via POST /api/nodes.

Body:

```json
{
  "worker_config_id": "worker-config-node-id"
}
```

Returns:

```json
{
  "run_id": "01961a2b-..."
}
```

The `run_id` (UUID v7, time-sortable) is used on all edges created during this run.

### GET /api/workers/runs/:id

Returns the run's metadata (worker_config_id, created_at) plus all nodes and edges created during this run. Useful for inspecting a worker's output, reviewing before committing, or identifying what to prune.

### GET /api/workers/queue

Returns nodes that have not yet been processed by a specific worker type. Query parameters:

| Parameter         | Type   | Description                                                           |
| ----------------- | ------ | --------------------------------------------------------------------- |
| `content_class`   | string | Content class the worker consumes (e.g. `source`)                     |
| `content_type`    | string | Content type the worker consumes (e.g. `conversation`)                |
| `encoding_class`  | string | Encoding class (e.g. `text`)                                          |
| `encoding_format` | string | Encoding format (e.g. `plain`)                                        |
| `not_consumed_by` | string | Content class that should NOT already have been derived from this node|
| `limit`           | int    | Pagination limit                                                      |

---

## Worker workflow

1. **Create worker/config node (once):** POST /api/nodes with `content_class: "worker"`, `content_type: "config"`, `encoding_class: "text"`, `encoding_format: "json"`, content = JSON with worker identity, version, parameters.
2. **Start a run:** POST /api/workers/runs with the config node ID. Returns `run_id`.
3. **Query for work:** GET /api/workers/queue with the content class/type the worker consumes.
4. **Process and write:** For each input, create output nodes via POST /api/nodes (or the convenience paths) with `run_id` and provenance edges referencing both the input nodes and the worker/config node.

---

## Design notes

- **Idempotency:** L0 root source creation is idempotent via `content_sha256` (deterministic ID). Everything else creates a new node.
- **Content serving:** The API resolves content from Postgres cache or S3 transparently. Consumers never interact with S3 directly.
- **Provenance enforcement:** Every non-root node requires at least one `provenance/input` edge and exactly one `provenance/worker` edge. The API refuses incomplete creations.
- **Immutability:** No PUT, no PATCH, no DELETE on nodes or edges. Corrections are new nodes with provenance referencing what they correct.
- **Atomic creation:** Every POST creates a node + all its edges in one transaction. No dangling nodes, no orphaned edges.
- **Confidence range:** -1.0 to +1.0. Negative = explicitly rejected ("investigated and ruled out"). Zero = unknown. Positive = affirmed.
