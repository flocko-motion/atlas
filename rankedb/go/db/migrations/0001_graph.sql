-- RankeDB core graph schema
-- Three-level graph: Sources (L0), Cognition (L1), Semantics (L2)
-- One graph, one nodes table, one edges table.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE nodes (
    -- Common fields (§3.1)
    id                TEXT PRIMARY KEY,
    level             SMALLINT NOT NULL CHECK (level IN (0, 1, 2)),
    content_class  TEXT NOT NULL,           -- infrastructure-fixed (e.g. source, entity, relation, classification)
    content_type      TEXT NOT NULL,           -- application-defined (e.g. conversation, person, family)
    encoding_class    TEXT NOT NULL CHECK (encoding_class IN ('text', 'image', 'audio', 'video', 'application')),
    encoding_format   TEXT NOT NULL,           -- application-extensible (e.g. eml, whatsapp, png, pdf, plain)
    content_sha256    TEXT NOT NULL,
    content_len       BIGINT NOT NULL,
    content_cached    TEXT,                    -- inline cache for eligible nodes
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Level 0 fields (§3.1.1)
    artifact_created_at      TIMESTAMPTZ,
    artifact_created_at_blur TEXT DEFAULT '0',
    origin                   TEXT,             -- ingest pathway
    original_name            TEXT,             -- original filename

    -- Level 2 fields (§3.1.3)
    valid_from       TIMESTAMPTZ,
    valid_from_blur  TEXT DEFAULT '0',
    valid_until      TIMESTAMPTZ,
    valid_until_blur TEXT DEFAULT '0',
    confidence       REAL CHECK (confidence >= -1.0 AND confidence <= 1.0)
);

-- Worker runs — registered before creating edges
CREATE TABLE runs (
    id              TEXT PRIMARY KEY,
    worker_config_id    TEXT NOT NULL REFERENCES nodes(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- All edges in one table — provenance and semantic.
-- Relation nodes + their head/tail edges are created atomically.
-- Provenance on the relation node covers the entire creation act.
-- Edges always connect nodes to nodes — no edge-to-edge references.
CREATE TABLE edges (
    id              TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    source_node_id  TEXT NOT NULL REFERENCES nodes(id),
    target_node_id  TEXT NOT NULL REFERENCES nodes(id),
    type            TEXT NOT NULL CHECK (type IN ('provenance/input', 'provenance/worker', 'relation/head', 'relation/tail')),
    confidence      REAL CHECK (confidence >= -1.0 AND confidence <= 1.0),  -- -1 = rejected, 0 = unknown, +1 = certain
    run_id          TEXT REFERENCES runs(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes: nodes
CREATE INDEX idx_nodes_level ON nodes(level);
CREATE INDEX idx_nodes_content_class ON nodes(content_class);
CREATE INDEX idx_nodes_content_type ON nodes(content_type);
CREATE INDEX idx_nodes_encoding_class ON nodes(encoding_class);
CREATE INDEX idx_nodes_encoding_format ON nodes(encoding_format);
CREATE INDEX idx_nodes_content_sha256 ON nodes(content_sha256);
CREATE INDEX idx_nodes_created_at ON nodes(created_at);
CREATE INDEX idx_nodes_artifact_created_at ON nodes(artifact_created_at) WHERE artifact_created_at IS NOT NULL;

-- Indexes: edges
CREATE INDEX idx_edges_source ON edges(source_node_id);
CREATE INDEX idx_edges_target ON edges(target_node_id);
CREATE INDEX idx_edges_type ON edges(type);
CREATE INDEX idx_edges_run ON edges(run_id);

-- Full-text search on cached content
CREATE INDEX idx_nodes_fts ON nodes USING gin(to_tsvector('english', content_cached)) WHERE content_cached IS NOT NULL;
