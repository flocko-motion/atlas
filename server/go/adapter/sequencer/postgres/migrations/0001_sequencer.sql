-- B_h storage: one compare-and-swap row per archive holding the current
-- branch-table head Id (a ranke.Id rendered via Id.String()). key is an opaque
-- per-archive identifier chosen by ranke-db; head is never empty (clearing the
-- head deletes the row).
CREATE TABLE ranke_sequencer (
    key  TEXT PRIMARY KEY,
    head TEXT NOT NULL
);
