package assembler_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"rankedb/assembler"
)

// Assemble's job is to compose a working, queryable ranke.Archive from each
// backend selection. The graph/claim mechanics are ranke-go's to test; here we
// only assert the archive is built and usable, that file paths are server-derived
// and confined, and that bad specs are rejected.

func TestAssembleMemArchiveIsQueryable(t *testing.T) {
	ctx := context.Background()
	h, err := assembler.Assemble(ctx, "acme", "main", assembler.Spec{
		Storage:   assembler.StorageSpec{Backend: "mem"},
		Sequencer: assembler.SequencerSpec{Backend: "mem"},
	}, assembler.Deps{})
	if err != nil {
		t.Fatalf("Assemble(mem,mem): %v", err)
	}
	defer h.Close()
	if bs := h.Archive.Branches(ctx); len(bs) != 0 {
		t.Fatalf("fresh archive has %d branches, want 0", len(bs))
	}
	if h.Archive.HasBranch(ctx, "main") {
		t.Fatal("fresh archive should not have a 'main' branch")
	}
}

func TestAssembleFsArchiveDerivesConfinedPath(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	h, err := assembler.Assemble(ctx, "acme", "main", assembler.Spec{
		Storage:   assembler.StorageSpec{Backend: "fs"},
		Sequencer: assembler.SequencerSpec{Backend: "file"},
	}, assembler.Deps{DataRoot: root})
	if err != nil {
		t.Fatalf("Assemble(fs,file): %v", err)
	}
	defer h.Close()
	if bs := h.Archive.Branches(ctx); len(bs) != 0 {
		t.Fatalf("fresh fs archive has %d branches, want 0", len(bs))
	}
	// The path is server-derived: <root>/tenants/<tenant>/<ra>/<type>/<adapter>/.
	for _, want := range []string{
		filepath.Join(root, "tenants", "acme", "main", "storage", "fs"),
		filepath.Join(root, "tenants", "acme", "main", "sequencer", "file"),
	} {
		if fi, err := os.Stat(want); err != nil || !fi.IsDir() {
			t.Fatalf("expected derived dir %q to exist: %v", want, err)
		}
	}
}

func TestAssembleSqliteArchive(t *testing.T) {
	ctx := context.Background()
	h, err := assembler.Assemble(ctx, "acme", "main", assembler.Spec{
		Storage:   assembler.StorageSpec{Backend: "sqlite"},
		Sequencer: assembler.SequencerSpec{Backend: "mem"},
	}, assembler.Deps{DataRoot: t.TempDir()})
	if err != nil {
		t.Fatalf("Assemble(sqlite,mem): %v", err)
	}
	defer h.Close()
	if bs := h.Archive.Branches(ctx); len(bs) != 0 {
		t.Fatalf("fresh sqlite archive has %d branches, want 0", len(bs))
	}
}

// Close releases the backends; a fresh handle closes without error.
func TestHandleCloseReleasesBackends(t *testing.T) {
	ctx := context.Background()
	h, err := assembler.Assemble(ctx, "acme", "main", assembler.Spec{
		Storage:   assembler.StorageSpec{Backend: "mem"},
		Sequencer: assembler.SequencerSpec{Backend: "mem"},
	}, assembler.Deps{})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if err := h.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestAssembleRejectsBadSpecs(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		spec assembler.Spec
		deps assembler.Deps
	}{
		{"unknown storage", assembler.Spec{Storage: assembler.StorageSpec{Backend: "bogus"}, Sequencer: assembler.SequencerSpec{Backend: "mem"}}, assembler.Deps{}},
		{"unknown sequencer", assembler.Spec{Storage: assembler.StorageSpec{Backend: "mem"}, Sequencer: assembler.SequencerSpec{Backend: "bogus"}}, assembler.Deps{}},
		// File-based backends without a configured data root are refused (not silently rooted anywhere).
		{"fs without data root", assembler.Spec{Storage: assembler.StorageSpec{Backend: "fs"}, Sequencer: assembler.SequencerSpec{Backend: "mem"}}, assembler.Deps{}},
		{"file without data root", assembler.Spec{Storage: assembler.StorageSpec{Backend: "mem"}, Sequencer: assembler.SequencerSpec{Backend: "file"}}, assembler.Deps{}},
		// "internal" sequencer with no server DB offered is refused.
		{"internal without server db", assembler.Spec{Storage: assembler.StorageSpec{Backend: "mem"}, Sequencer: assembler.SequencerSpec{Backend: "internal", Key: "k"}}, assembler.Deps{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := assembler.Assemble(ctx, "acme", "main", c.spec, c.deps); err == nil {
				t.Fatalf("Assemble(%+v) = nil error, want a rejection", c.spec)
			}
		})
	}
}

// Even with a data root, an unsafe identity segment must never escape it
// (defense in depth — core validates names, but the assembler is the path gate).
func TestAssembleRejectsPathTraversal(t *testing.T) {
	ctx := context.Background()
	_, err := assembler.Assemble(ctx, "../../etc", "main", assembler.Spec{
		Storage:   assembler.StorageSpec{Backend: "fs"},
		Sequencer: assembler.SequencerSpec{Backend: "mem"},
	}, assembler.Deps{DataRoot: t.TempDir()})
	if err == nil {
		t.Fatal("traversal in tenant name should be rejected")
	}
}
