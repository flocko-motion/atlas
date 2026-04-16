# RankeDB API — Draft

Working draft. Not a spec — a starting point for implementation.

---

## Nodes

```
POST   /api/nodes              — Create node (multipart for L0 with file, JSON for L1/L2;
                                  idempotent via content_sha256 for L0 roots)
GET    /api/nodes/:id          — Node with content
GET    /api/nodes/:id/content  — Raw content (redirect to S3 or inline from cache)
GET    /api/nodes/:id/provenance — Upstream DAG to L0 roots
GET    /api/nodes              — List/filter (level, content_type, encoding, created_at range,
                                  worker/run_id)
```

### POST /api/nodes

Creates a node. Returns existing node if idempotent match (L0 root: same `content_sha256`).

**L0 (source):** multipart form — file upload + metadata fields (`content_type`, `encoding`, `origin`, `original_name`, `artifact_created_at`, `artifact_created_at_blur`).

**L1 (cognition):** JSON body — `content` (text), `content_type`, `encoding`, plus provenance edges (list of input node IDs + tool node ID + run_id).

**L2 (semantics):** JSON body — `content` (e.g. relation label), `content_type`, `encoding`, `valid_from`, `valid_from_blur`, `valid_until`, `valid_until_blur`, `confidence`, plus provenance edges.

### GET /api/nodes/:id

Returns full node metadata + content (inline if cached, otherwise fetched from S3).

### GET /api/nodes/:id/content

Returns raw content bytes. If content is cached in Postgres, serves directly; otherwise redirects to S3 (or proxies the blob).

### GET /api/nodes/:id/provenance

Returns the upstream provenance chain from this node back to L0 roots. Traverses provenance edges recursively. Response is a subgraph (nodes + edges).

### GET /api/nodes

Filtered listing. Query parameters:

| Parameter      | Type     | Description                                    |
| -------------- | -------- | ---------------------------------------------- |
| `level`        | int      | 0, 1, or 2                                     |
| `content_type` | string   | e.g. `source/conversation`, `classification/*` |
| `encoding`     | string   | e.g. `text/plain`, `text/eml`                  |
| `created_after`| datetime | Lower bound on `created_at`                    |
| `created_before`| datetime | Upper bound on `created_at`                   |
| `run_id`       | string   | Filter by worker run                           |
| `limit`        | int      | Pagination limit                               |
| `offset`       | int      | Pagination offset                              |

---

## Edges

```
POST   /api/edges              — Create edge (provenance or semantic, with run_id)
GET    /api/edges/:id          — Edge with run_id, tool reference
GET    /api/edges/:id/provenance — Which run, which tool, which config
GET    /api/nodes/:id/edges    — Edges of a node (filterable: direction, type)
```

### POST /api/edges

Creates an edge. Body:

```json
{
  "source_id": "node-or-edge-id",
  "target_id": "node-or-edge-id",
  "type": "provenance | head | tail",
  "run_id": "run-abc-123",
  "confidence": 0.85
}
```

Note: `source_id` / `target_id` can reference nodes OR edges (provenance can target edges per §3.2).

### GET /api/edges/:id

Returns edge metadata: source, target, type, run_id, confidence, timestamps.

### GET /api/edges/:id/provenance

Returns the provenance context of this edge: which run produced it, which tool node was in effect, which config. Follows the run_id to the tool node.

### GET /api/nodes/:id/edges

Returns edges connected to a node. Query parameters:

| Parameter   | Type   | Description                              |
| ----------- | ------ | ---------------------------------------- |
| `direction` | string | `incoming`, `outgoing`, or `both`        |
| `type`      | string | `provenance`, `head`, `tail`, or `all`   |
| `limit`     | int    | Pagination limit                         |
| `offset`    | int    | Pagination offset                        |

---

## Semantic Graph (L2)

```
GET    /api/entities/:id           — Entity with all relations, temporal sorted, confidence filtered
GET    /api/entities               — Full-text search over entity names/aliases
GET    /api/entities/:id/timeline  — Chronological relations of an entity
GET    /api/relations              — Filter relations (e.g. unresolved, by type)
```

### GET /api/entities/:id

Returns entity node + all connected relation nodes with their head/tail edges, sorted by temporal validity, filtered by minimum confidence (query param `min_confidence`, default 0).

### GET /api/entities

Search entities. Query parameters:

| Parameter | Type   | Description                                |
| --------- | ------ | ------------------------------------------ |
| `q`       | string | Full-text search over entity names/aliases |
| `type`    | string | e.g. `entity/person`, `entity/organization`|
| `limit`   | int    | Pagination limit                           |
| `offset`  | int    | Pagination offset                          |

### GET /api/entities/:id/timeline

Returns all relations of an entity sorted chronologically by `valid_from`. Useful for building a narrative timeline of an entity's history.

### GET /api/relations

Filter relations. Query parameters:

| Parameter    | Type   | Description                                                 |
| ------------ | ------ | ----------------------------------------------------------- |
| `unresolved` | bool   | If true, return relations with 0 or >1 heads (open questions, knowledge gaps) |
| `type`       | string | e.g. `relation/alias`, `relation/has_role`                  |
| `limit`      | int    | Pagination limit                                            |
| `offset`     | int    | Pagination offset                                           |

---

## Workers

```
GET    /api/queue     — Unprocessed nodes for a given content_type/encoding filter
POST   /api/runs      — Register a worker run (tool node reference, returns run_id)
```

### GET /api/queue

Returns nodes that have not yet been processed by a specific worker type. Query parameters:

| Parameter      | Type   | Description                                       |
| -------------- | ------ | ------------------------------------------------- |
| `content_type` | string | Content type the worker consumes (e.g. `source/conversation`) |
| `encoding`     | string | Encoding the worker consumes (e.g. `text/plain`)  |
| `not_consumed_by` | string | Tool content_type that should NOT already have derived from this node |
| `limit`        | int    | Pagination limit                                  |

### POST /api/runs

Registers a new worker run. Body:

```json
{
  "tool_node_id": "node-id-of-tool-config"
}
```

Returns:

```json
{
  "run_id": "run-abc-123"
}
```

The `run_id` is then used on all edges created during this run.

---

## Design notes

- **Idempotency:** L0 root node creation is idempotent via `content_sha256`. Re-uploading the same bytes returns the existing node. Derived nodes (L0 derived, L1, L2) are NOT idempotent — same content with different provenance creates a new node.
- **Content serving:** The API resolves content from Postgres cache or S3 transparently. Consumers never interact with S3 directly.
- **Provenance enforcement:** POST /api/nodes for L1/L2 requires at least one input edge. The API refuses to create a derivation without provenance.
- **Immutability enforcement:** No PUT, no PATCH, no DELETE on nodes or edges. Corrections are new nodes that reference what they correct.
- **Edge targets:** Edges can target both nodes and edges (per §3.2 of the paper). The `source_id` and `target_id` fields accept either.
