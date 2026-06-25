package api

import (
	"context"
	"errors"
	"testing"

	schemafapi "github.com/flocko-motion/schemaf/api"

	"rankedb/access"
	"rankedb/adapter/config"
	configmem "rankedb/adapter/config/mem"
	grantsmem "rankedb/adapter/grants/mem"
	"rankedb/assembler"
	"rankedb/core"
)

// TestGetArchiveRoute pins the endpoint's HTTP contract (and keeps Method/Path/
// Auth reachable until codegen wires the Provider that calls them).
func TestGetArchiveRoute(t *testing.T) {
	e := GetArchiveEndpoint{}
	if e.Method() != "GET" {
		t.Fatalf("method = %q, want GET", e.Method())
	}
	if e.Path() != "/api/archives/{tenant}/{ra}" {
		t.Fatalf("path = %q", e.Path())
	}
	if !e.Auth() {
		t.Fatal("archive endpoint must require auth")
	}
}

func TestMapErr(t *testing.T) {
	if mapErr(nil) != nil {
		t.Fatal("mapErr(nil) should be nil")
	}
	if !errors.Is(mapErr(core.ErrNotFound), schemafapi.ErrNotFound) {
		t.Fatal("core.ErrNotFound should map to schemaf ErrNotFound (404)")
	}
	denied := &access.Denied{Subject: "alice", Reason: "no grant"}
	got := mapErr(denied)
	if !errors.Is(got, schemafapi.ErrForbidden) {
		t.Fatalf("Denied should map to ErrForbidden (403); got %v", got)
	}
	if !contains(got.Error(), "alice") {
		t.Fatalf("403 body should carry the subject id for onboarding; got %q", got.Error())
	}
}

// GetArchive hides an archive from a caller with no visibility into the tenant
// (here: no authenticated subject → not visible → 404). The positive path needs
// an authenticated JWT subject and is covered end-to-end once serving is wired.
func TestGetArchiveHidesWhenNotVisible(t *testing.T) {
	entries := config.Entries{
		"tenants.acme.archives.main.state":             "running",
		"tenants.acme.archives.main.storage.backend":   "mem",
		"tenants.acme.archives.main.sequencer.backend": "mem",
	}
	c := core.New(access.New(nil, grantsmem.New()), configmem.NewFrom(entries), assembler.Deps{})
	if err := c.Reconcile(context.Background()); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	Use(c)

	_, err := GetArchiveEndpoint{}.Handle(context.Background(), GetArchiveReq{Tenant: "acme", RA: "main"})
	if !errors.Is(err, schemafapi.ErrNotFound) {
		t.Fatalf("unauthenticated/invisible caller should get 404 (hide), got %v", err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
