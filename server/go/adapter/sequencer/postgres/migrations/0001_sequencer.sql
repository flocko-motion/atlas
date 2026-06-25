-- B_h history: APPEND-ONLY. Each Save inserts a row, so the per-key sequence of
-- rows is the branch-table head's full revision history — the sequencer's
-- purpose (it owns the total order), not a single overwritten cell. The current
-- head is the row with the greatest seq for a key; head IS NULL marks a clear.
-- key is the opaque per-archive identifier chosen by ranke-db.
CREATE TABLE ranke_sequencer (
    seq        BIGSERIAL PRIMARY KEY,
    key        TEXT NOT NULL,
    head       TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Tip lookup per key (ORDER BY seq DESC LIMIT 1) and history scans.
CREATE INDEX ranke_sequencer_key_seq_idx ON ranke_sequencer (key, seq DESC);
