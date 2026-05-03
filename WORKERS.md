# Building RankeDB Workers

Workers are external processes that read and write the RankeDB graph through the API. They live in their own repositories and use the SDK to communicate.

## SDK

```
go get github.com/flocko-motion/rankedb/worker
```

Import as:

```go
import rankedb "github.com/flocko-motion/rankedb/worker"
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
configID, runID, err := client.StartRun(ctx, rankedb.WorkerConfig{
    Name:    "my-entity-extractor",
    Version: "0.1",
    Params:  map[string]string{"model": "gpt-4o"},
})
```

Returns a UUID v7 run ID (time-sortable). The client stores both IDs internally — `Queue` and `Done` use them automatically.

### 3. Query for work

Find source nodes that haven't been processed yet:

```go
nodes, err := client.Queue(ctx, rankedb.QueueParams{
    ContentClass: "source",
    ContentType:  "conversation",
})
```

By default, filters by the exact config ID from `StartRun` — sources already processed by this config are skipped. Override with `ByWorker` (skip if any config of that worker name processed it) or `ByClass` (skip if any derived node of that class exists) for coarser granularity. Set `Reprocess: true` to bypass all filters and return every match — useful during development, replays, or to deliberately layer a second pass over already-processed sources. `Done` still creates its processed marker on a reprocess run (each run produces its own marker, which is itself a fact).

### 4. Process and write

For each source, create output nodes and signal completion:

```go
for _, source := range nodes {
    nodeID, err := client.CreateNode(ctx, rankedb.CreateNodeRequest{
        Level:         1,
        ContentClass:  "classification",
        ContentType:   "entity",
        EncodingClass: "text",
        EncodingFormat: "plain",
        Content:       rankedb.Ptr("Person: Alice Müller"),
        Edges: []rankedb.EdgeSpec{
            {Type: "provenance/input", SourceNodeID: source.Id, RunID: &runID},
            {Type: "provenance/worker", SourceNodeID: configID, RunID: &runID},
        },
    })

    // Signal done — for bulk sources, creates a processed marker.
    // For non-bulk sources, no-op (the L1 output is proof enough).
    client.Done(ctx, &source)
}
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
        {Type: "provenance/input", SourceNodeID: factNodeID},
        {Type: "provenance/worker", SourceNodeID: configID},
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
| `source` | `event` | Calendar entries (ICS), meeting invites, reservations |
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
