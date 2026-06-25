package postgres_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	ranke "github.com/flocko-motion/ranke-go"
	pgseq "rankedb/adapter/sequencer/postgres"

	_ "github.com/lib/pq"
)

// Needs a real Postgres. Set RANKE_TEST_PG_DSN to a disposable database to run;
// otherwise it skips. The test clears its key first so reruns are deterministic.
func TestHeadRoundTrip(t *testing.T) {
	dsn := os.Getenv("RANKE_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("RANKE_TEST_PG_DSN not set; skipping Postgres sequencer test")
	}
	ctx := context.Background()
	h, err := pgseq.New(ctx, dsn, "tenantA/main")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer h.Close()

	// Start from a clean slate for this key.
	if err := h.Save(ctx, nil); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if id, err := h.Load(ctx); err != nil || id != nil {
		t.Fatalf("Load (empty) = %v, %v; want nil, nil", id, err)
	}

	// Save a head and read it back.
	want, err := ranke.HashContent([]byte("head-1"))
	if err != nil {
		t.Fatalf("HashContent: %v", err)
	}
	if err := h.Save(ctx, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := h.Load(ctx)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got == nil || !got.Equal(want) {
		t.Fatalf("Load = %v; want %v", got, want)
	}

	// Saving nil clears it.
	if err := h.Save(ctx, nil); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if id, err := h.Load(ctx); err != nil || id != nil {
		t.Fatalf("Load (after clear) = %v, %v; want nil, nil", id, err)
	}
}

// TestNewWithConn covers the borrowed-connection path (the sequencing-point use:
// share schemaf's built-in Postgres): Close must not close the caller's db.
func TestNewWithConn(t *testing.T) {
	dsn := os.Getenv("RANKE_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("RANKE_TEST_PG_DSN not set; skipping Postgres sequencer test")
	}
	ctx := context.Background()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	h, err := pgseq.NewWithConn(ctx, func() *sql.DB { return db }, "tenantB/main")
	if err != nil {
		t.Fatalf("NewWithConn: %v", err)
	}
	if err := h.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("borrowed db was closed by Head.Close (it must stay open): %v", err)
	}
}
