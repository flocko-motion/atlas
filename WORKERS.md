# Building RankeDB Workers

Workers are external processes that read and write the RankeDB graph through the API. They live in their own repositories and use the SDK to communicate.

## SDK

```
go get github.com/flocko-motion/rankedb/sdk
```

Import as:

```go
import rankedb "github.com/flocko-motion/rankedb/sdk"
```

## Worker lifecycle

### 1. Create a worker config (once)

Every worker registers its identity and configuration as an L0 source node:

```go
client := rankedb.NewClient("http://localhost:8000")

configID, err := client.CreateWorkerConfig(ctx, rankedb.WorkerConfig{
    Name:    "my-entity-extractor",
    Version: "0.1",
    Params:  map[string]any{"model": "gpt-4o", "temperature": 0.2},
})
```

This creates a `source/worker-config` node with `text/json` encoding. Do this once, or when the config changes (new node, immutability).

### 2. Start a run

Before creating any edges, register a run:

```go
runID, err := client.StartRun(ctx, configID)
```

Returns a UUID v7 (time-sortable). All edges created during this run carry this `run_id`, enabling batch operations (inspect, prune).

### 3. Query for work

Find nodes that haven't been processed yet:

```go
nodes, err := client.Queue(ctx, rankedb.QueueParams{
    ContentClass: "source",
    ContentType:  "conversation",
    EncodingClass: "text",
    NotConsumedBy: "classification",
})
```

Returns source nodes that have no `provenance/input` edge from a `classification/*` node.

### 4. Process and write

For each input, create output nodes with provenance:

```go
nodeID, err := client.CreateNode(ctx, rankedb.CreateNodeRequest{
    Level:         1,
    ContentClass:  "classification",
    ContentType:   "entity",
    EncodingClass: "text",
    EncodingFormat: "plain",
    Content:       "Person: Alice Müller",
    RunID:         runID,
    Edges: []rankedb.EdgeSpec{
        {Type: "provenance/input", TargetNodeID: sourceNodeID},
        {Type: "provenance/worker", TargetNodeID: configID},
    },
})
```

### 5. Create relations (L2)

Relations are nodes with head/tail edges, created atomically:

```go
relID, err := client.CreateNode(ctx, rankedb.CreateNodeRequest{
    Level:         2,
    ContentClass:  "relation",
    ContentType:   "family",
    EncodingClass: "text",
    EncodingFormat: "plain",
    Content:       "sister of",
    RunID:         runID,
    ValidFrom:     "2000-01-01T00:00:00Z",
    ValidFromBlur: "365d",
    Edges: []rankedb.EdgeSpec{
        {Type: "provenance/input", TargetNodeID: factNodeID},
        {Type: "provenance/worker", TargetNodeID: configID},
        {Type: "relation/tail", TargetNodeID: aliceID, Confidence: 1.0},
        {Type: "relation/head", TargetNodeID: bobID, Confidence: 1.0},
    },
})
```

Reading convention: **"tail IS relation TOWARDS head"** — Alice(tail) is sister_of towards Bob(head).

## Provenance rules

The API enforces:

- **Every non-root node** must have at least one `provenance/input` edge and exactly one `provenance/worker` edge.
- **L0 root sources** have no edges (they are roots).
- **Relations** are created atomically — the node and all its head/tail/provenance edges in one request. Edges are never added to an existing relation.
- **Immutability** — no updates, no deletes. Corrections are new nodes referencing what they correct.

## Edge types

| Type | Meaning |
|---|---|
| `provenance/input` | Evidence — what this node was derived from |
| `provenance/worker` | Who made it — the worker config node |
| `relation/head` | Object of the relation (entity node) |
| `relation/tail` | Subject of the relation (entity node) |

## Content types

### L0 Sources (ingest)

| content_class | content_type | What |
|---|---|---|
| `source` | `conversation` | Email, chat, letter |
| `source` | `media` | Photo, video, audio |
| `source` | `record` | Sensor data, transactions |
| `source` | `data` | Spreadsheets, exports |
| `source` | `bulk` | Archive to be unpacked |
| `source` | `worker-config` | Worker identity + config |

### L1 Cognition (workers produce)

| content_class | content_type | What |
|---|---|---|
| `classification` | `entity`, `content`, `topic` | Statement about a node |
| `observation` | application-defined | Relationship between nodes |
| `summary` | application-defined | Condensed representation |
| `fact` | application-defined | Extracted factual claim |
| `conversation` | application-defined | Resolved conversation |

### L2 Semantics (workers project)

| content_class | content_type | What |
|---|---|---|
| `entity` | `person`, `organization`, `place`, `thing`, `work`, `idea`, `event`, `role` | Identifiable thing |
| `relation` | `alias`, `part_of`, `has_role`, `family`, + application-defined | Semantic relation |

## Confidence

Range: **-1.0 to +1.0**

- `+1.0` = certain
- `0.0` = unknown
- `-1.0` = explicitly rejected ("investigated and ruled out")

Negative confidence records negative knowledge — "we looked and it's not this." Different from no edge (never investigated).

## Temporal fuzziness

Date fields (`valid_from`, `valid_until`, `artifact_created_at`) have a `_blur` sibling — a duration string (e.g. `"30d"`, `"365d"`, `"0"`) expressing temporal fuzziness. Default `"0"` (precise).
