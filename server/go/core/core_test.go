package core_test

import (
	"context"
	"errors"
	"testing"

	"rankedb/access"
	"rankedb/adapter/config"
	configmem "rankedb/adapter/config/mem"
	"rankedb/adapter/grants"
	grantsmem "rankedb/adapter/grants/mem"
	"rankedb/assembler"
	"rankedb/core"
)

func baseEntries() config.Entries {
	return config.Entries{
		"tenants.acme.title":                             "Acme",
		"tenants.acme.archives.main.title":               "Main",
		"tenants.acme.archives.main.state":               "running",
		"tenants.acme.archives.main.storage.backend":     "mem",
		"tenants.acme.archives.main.sequencer.backend":   "mem",
		"tenants.acme.archives.paused.state":             "stopped",
		"tenants.acme.archives.paused.storage.backend":   "mem",
		"tenants.acme.archives.paused.sequencer.backend": "mem",
	}
}

func newCore(t *testing.T, store config.Store, gs *grantsmem.Store, roots ...string) *core.Core {
	t.Helper()
	c := core.New(access.New(roots, gs), store, assembler.Deps{})
	if err := c.Reconcile(context.Background()); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	return c
}

func TestReconcileBringsUpTargets(t *testing.T) {
	ctx := context.Background()
	c := newCore(t, configmem.NewFrom(baseEntries()), grantsmem.New(), "root")

	if s, ok := c.StateOf("acme", "main"); !ok || s != core.StateRunning {
		t.Fatalf("main state = %q, %v; want running", s, ok)
	}
	if s, ok := c.StateOf("acme", "paused"); !ok || s != core.StateStopped {
		t.Fatalf("paused state = %q, %v; want stopped", s, ok)
	}

	// root reads a running archive; a stopped one is operationally unavailable.
	if _, err := c.Reader(ctx, "root", "acme", "main"); err != nil {
		t.Fatalf("Reader(main): %v", err)
	}
	if _, err := c.Reader(ctx, "root", "acme", "paused"); !errors.Is(err, core.ErrUnavailable) {
		t.Fatalf("Reader(paused) err = %v; want ErrUnavailable", err)
	}
	// Unknown archive → not found.
	if _, err := c.Reader(ctx, "root", "acme", "ghost"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("Reader(ghost) err = %v; want ErrNotFound", err)
	}
}

func TestReaderEnforcesAuthzGate(t *testing.T) {
	ctx := context.Background()
	gs := grantsmem.New()
	c := newCore(t, configmem.NewFrom(baseEntries()), gs) // no roots

	// A stranger with no grant is denied (gate 1).
	_, err := c.Reader(ctx, "stranger", "acme", "main")
	var d *access.Denied
	if !errors.As(err, &d) {
		t.Fatalf("Reader(stranger) err = %v; want *access.Denied", err)
	}

	// Grant read → allowed.
	_ = gs.PutGrant(ctx, grants.Grant{Subject: "r", Scope: grants.Archive("acme", "main"), Role: grants.RoleRARead})
	if _, err := c.Reader(ctx, "r", "acme", "main"); err != nil {
		t.Fatalf("Reader(r) after grant: %v", err)
	}
}

func TestControlStopsAndPersists(t *testing.T) {
	ctx := context.Background()
	store := configmem.NewFrom(baseEntries())
	gs := grantsmem.New()
	c := newCore(t, store, gs, "root")

	if err := c.Control(ctx, "root", "acme", "main", core.StateStopped); err != nil {
		t.Fatalf("Control stop: %v", err)
	}
	if s, _ := c.StateOf("acme", "main"); s != core.StateStopped {
		t.Fatalf("main state after stop = %q; want stopped", s)
	}
	if _, err := c.Reader(ctx, "root", "acme", "main"); !errors.Is(err, core.ErrUnavailable) {
		t.Fatalf("Reader(main) after stop = %v; want ErrUnavailable", err)
	}

	// Target persisted to config.
	e, _ := store.Load(ctx)
	if e["tenants.acme.archives.main.state"] != "stopped" {
		t.Fatalf("persisted state = %q; want stopped", e["tenants.acme.archives.main.state"])
	}
	// A fresh core over the same store reconciles main to stopped.
	c2 := newCore(t, store, gs, "root")
	if s, _ := c2.StateOf("acme", "main"); s != core.StateStopped {
		t.Fatalf("reloaded main state = %q; want stopped", s)
	}
}

func TestControlNeedsControlGrant(t *testing.T) {
	ctx := context.Background()
	c := newCore(t, configmem.NewFrom(baseEntries()), grantsmem.New()) // no roots
	err := c.Control(ctx, "stranger", "acme", "main", core.StateStopped)
	var d *access.Denied
	if !errors.As(err, &d) {
		t.Fatalf("Control(stranger) err = %v; want *access.Denied", err)
	}
}

func TestFailedArchiveDoesNotBlockBoot(t *testing.T) {
	entries := baseEntries()
	entries["tenants.acme.archives.broken.state"] = "running"
	entries["tenants.acme.archives.broken.storage.backend"] = "bogus" // assembly will fail
	entries["tenants.acme.archives.broken.sequencer.backend"] = "mem"

	c := newCore(t, configmem.NewFrom(entries), grantsmem.New(), "root")

	if s, _ := c.StateOf("acme", "broken"); s != core.StateFailed {
		t.Fatalf("broken state = %q; want failed", s)
	}
	// The healthy archive still came up.
	if s, _ := c.StateOf("acme", "main"); s != core.StateRunning {
		t.Fatalf("main state = %q; want running (failed sibling must not block boot)", s)
	}
}

func TestInvalidNameFailsLoad(t *testing.T) {
	store := configmem.NewFrom(config.Entries{
		"tenants.Acme.archives.main.storage.backend":   "mem", // uppercase tenant name
		"tenants.Acme.archives.main.sequencer.backend": "mem",
	})
	c := core.New(access.New([]string{"root"}, grantsmem.New()), store, assembler.Deps{})
	if err := c.Reconcile(context.Background()); err == nil {
		t.Fatal("Reconcile with an invalid tenant name should error")
	}
}

func TestCreateAndDeleteArchive(t *testing.T) {
	ctx := context.Background()
	c := newCore(t, configmem.NewFrom(config.Entries{}), grantsmem.New(), "root")

	st, err := c.CreateArchive(ctx, "root", "acme", "fresh", "Fresh", assembler.Spec{
		Storage:   assembler.StorageSpec{Backend: "mem"},
		Sequencer: assembler.SequencerSpec{Backend: "mem"},
	})
	if err != nil {
		t.Fatalf("CreateArchive: %v", err)
	}
	if st.Current != core.StateRunning {
		t.Fatalf("created archive state = %q, want running", st.Current)
	}
	if s, ok := c.StateOf("acme", "fresh"); !ok || s != core.StateRunning {
		t.Fatalf("StateOf(fresh) = %q, %v; want running", s, ok)
	}

	// A missing sequencer backend (B_h is the key) is rejected.
	if _, err := c.CreateArchive(ctx, "root", "acme", "bad", "", assembler.Spec{
		Storage: assembler.StorageSpec{Backend: "mem"},
	}); !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("missing sequencer.backend should be ErrInvalid; got %v", err)
	}

	// A non-admin cannot create.
	_, err = c.CreateArchive(ctx, "stranger", "acme", "x", "", assembler.Spec{
		Storage:   assembler.StorageSpec{Backend: "mem"},
		Sequencer: assembler.SequencerSpec{Backend: "mem"},
	})
	var d *access.Denied
	if !errors.As(err, &d) {
		t.Fatalf("non-admin create should be denied; got %v", err)
	}

	// Delete removes it.
	if err := c.DeleteArchive(ctx, "root", "acme", "fresh"); err != nil {
		t.Fatalf("DeleteArchive: %v", err)
	}
	if _, ok := c.StateOf("acme", "fresh"); ok {
		t.Fatal("archive should be gone after delete")
	}
}
