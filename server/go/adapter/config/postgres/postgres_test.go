package postgres_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"rankedb/adapter/config"
	"rankedb/adapter/config/postgres"

	_ "github.com/lib/pq"
)

// Needs a real Postgres. Set RANKE_TEST_PG_DSN to a disposable database to run;
// otherwise it skips. The migration is idempotent; the test overwrites the whole
// config document (Save replaces it), so reruns are deterministic.
func TestStoreRoundTrip(t *testing.T) {
	dsn := os.Getenv("RANKE_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("RANKE_TEST_PG_DSN not set; skipping Postgres config store test")
	}
	ctx := context.Background()
	s, err := postgres.New(ctx, dsn)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Close()

	want := config.Entries{"server.addr": ":8080", "vault.adapter": "age"}
	if err := s.Save(ctx, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load(ctx)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != len(want) || got["server.addr"] != ":8080" || got["vault.adapter"] != "age" {
		t.Fatalf("Load = %v; want %v", got, want)
	}

	// Save replaces the whole document, it does not merge.
	if err := s.Save(ctx, config.Entries{"server.addr": ":9090"}); err != nil {
		t.Fatalf("Save (replace): %v", err)
	}
	got, err = s.Load(ctx)
	if err != nil {
		t.Fatalf("Load after replace: %v", err)
	}
	if len(got) != 1 || got["server.addr"] != ":9090" {
		t.Fatalf("Load after replace = %v; want a single server.addr=:9090", got)
	}
}

// TestNewWithConn covers the borrowed-connection path (the assembler reuses
// schemaf's built-in Postgres this way): Close must not close the caller's db.
func TestNewWithConn(t *testing.T) {
	dsn := os.Getenv("RANKE_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("RANKE_TEST_PG_DSN not set; skipping Postgres config store test")
	}
	ctx := context.Background()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	s, err := postgres.NewWithConn(ctx, func() *sql.DB { return db })
	if err != nil {
		t.Fatalf("NewWithConn: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("borrowed db was closed by store.Close (it must stay open): %v", err)
	}
}
