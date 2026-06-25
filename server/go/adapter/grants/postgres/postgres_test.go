package postgres_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"rankedb/access"
	"rankedb/adapter/grants"
	pgstore "rankedb/adapter/grants/postgres"

	_ "github.com/lib/pq"
)

// These tests need a real Postgres. Set RANKE_TEST_PG_DSN to a disposable
// database (e.g. postgres://user:pass@localhost:5432/ranke_test?sslmode=disable)
// to run them; otherwise they skip. The migration is idempotent, but the tests
// clean their rows up front so reruns are deterministic.
func openStore(t *testing.T) *pgstore.Store {
	t.Helper()
	dsn := os.Getenv("RANKE_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("RANKE_TEST_PG_DSN not set; skipping Postgres access store test")
	}
	ctx := context.Background()
	s, err := pgstore.New(ctx, dsn)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	// Start from a clean slate (tables are shared across runs). Use a separate
	// short-lived connection so the production store keeps no test-only seams.
	raw, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open clean conn: %v", err)
	}
	defer raw.Close()
	for _, q := range []string{"DELETE FROM ranke_grants", "DELETE FROM ranke_subjects"} {
		if _, err := raw.ExecContext(ctx, q); err != nil {
			t.Fatalf("clean (%s): %v", q, err)
		}
	}
	return s
}

func TestStoreRoundTrip(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	ra := grants.Archive("A", "main")

	// Unknown subject: not disabled, no grants.
	if d, err := s.Disabled(ctx, "ghost"); err != nil || d {
		t.Fatalf("Disabled(ghost) = %v, %v; want false, nil", d, err)
	}
	if gs, err := s.GrantsFor(ctx, "ghost"); err != nil || len(gs) != 0 {
		t.Fatalf("GrantsFor(ghost) = %v, %v; want empty, nil", gs, err)
	}

	// Grants are additive: two roles on one scope coexist; re-putting is idempotent.
	if err := s.PutGrant(ctx, grants.Grant{Subject: "r", Scope: ra, Role: grants.RoleRAWrite}); err != nil {
		t.Fatalf("PutGrant write: %v", err)
	}
	if err := s.PutGrant(ctx, grants.Grant{Subject: "r", Scope: ra, Role: grants.RoleRAOperator}); err != nil {
		t.Fatalf("PutGrant operator: %v", err)
	}
	if err := s.PutGrant(ctx, grants.Grant{Subject: "r", Scope: ra, Role: grants.RoleRAWrite}); err != nil {
		t.Fatalf("PutGrant write (idempotent): %v", err)
	}
	gs, err := s.GrantsFor(ctx, "r")
	if err != nil {
		t.Fatalf("GrantsFor: %v", err)
	}
	if len(gs) != 2 {
		t.Fatalf("GrantsFor(r) = %+v; want 2 grants (write + operator, additive, no duplicate)", gs)
	}

	// Disable round-trips.
	if err := s.SetDisabled(ctx, "r", true); err != nil {
		t.Fatalf("SetDisabled: %v", err)
	}
	if d, err := s.Disabled(ctx, "r"); err != nil || !d {
		t.Fatalf("Disabled(r) = %v, %v; want true, nil", d, err)
	}

	// DeleteGrant removes only the named role, leaving the other intact.
	if err := s.DeleteGrant(ctx, "r", ra, grants.RoleRAOperator); err != nil {
		t.Fatalf("DeleteGrant operator: %v", err)
	}
	gs, err = s.GrantsFor(ctx, "r")
	if err != nil {
		t.Fatalf("GrantsFor after delete: %v", err)
	}
	if len(gs) != 1 || gs[0].Role != grants.RoleRAWrite {
		t.Fatalf("GrantsFor(r) after delete = %+v; want only the write grant", gs)
	}
}

// TestEngineOverPostgres drives the real Authz engine against the Postgres
// store end to end — the same "may A do B on C?" question, answered from a DB.
func TestEngineOverPostgres(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	a := access.New([]string{"root-sub"}, s)

	if err := s.PutGrant(ctx, grants.Grant{Subject: "boss", Scope: grants.Tenant("A"), Role: grants.RoleTenantAdmin}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if d, _ := a.Decide(ctx, access.Request{Subject: "root-sub", Scope: grants.Archive("A", "main"), Action: access.AdminRA}); !d.Allowed || d.Reason != "root" {
		t.Fatalf("root decision = %+v", d)
	}
	if d, _ := a.Decide(ctx, access.Request{Subject: "boss", Scope: grants.Archive("A", "x"), Action: access.WriteRA}); !d.Allowed || d.Reason != "tenant-admin" {
		t.Fatalf("tenant-admin decision = %+v", d)
	}
	if d, _ := a.Decide(ctx, access.Request{Subject: "stranger", Scope: grants.Archive("A", "main"), Action: access.ReadRA}); d.Allowed || d.Reason != "no grant" {
		t.Fatalf("stranger decision = %+v", d)
	}
}

// TestNewWithConn covers the borrowed-connection path (the assembler reuses
// schemaf's built-in Postgres this way): Close must not close the caller's db.
func TestNewWithConn(t *testing.T) {
	dsn := os.Getenv("RANKE_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("RANKE_TEST_PG_DSN not set; skipping Postgres access store test")
	}
	ctx := context.Background()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	s, err := pgstore.NewWithConn(ctx, func() *sql.DB { return db })
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
