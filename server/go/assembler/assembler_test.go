package assembler_test

import (
	"context"
	"path/filepath"
	"testing"

	"rankedb/assembler"
)

// Assemble's job is to compose a working, queryable ranke.Archive from each
// backend selection. The graph/claim mechanics are ranke-go's to test; here we
// only assert the archive is built and usable, and that bad specs are rejected.

func TestAssembleMemArchiveIsQueryable(t *testing.T) {
	ctx := context.Background()
	h, err := assembler.Assemble(ctx, assembler.Spec{
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

func TestAssembleFsArchive(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	h, err := assembler.Assemble(ctx, assembler.Spec{
		Storage:   assembler.StorageSpec{Backend: "fs", Dir: filepath.Join(dir, "store")},
		Sequencer: assembler.SequencerSpec{Backend: "file", Path: filepath.Join(dir, "head")},
	}, assembler.Deps{})
	if err != nil {
		t.Fatalf("Assemble(fs,file): %v", err)
	}
	defer h.Close()
	if bs := h.Archive.Branches(ctx); len(bs) != 0 {
		t.Fatalf("fresh fs archive has %d branches, want 0", len(bs))
	}
}

func TestAssembleSqliteArchive(t *testing.T) {
	ctx := context.Background()
	dsn := filepath.Join(t.TempDir(), "u.db")
	h, err := assembler.Assemble(ctx, assembler.Spec{
		Storage:   assembler.StorageSpec{Backend: "sqlite", DSN: dsn},
		Sequencer: assembler.SequencerSpec{Backend: "mem"},
	}, assembler.Deps{})
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
	h, err := assembler.Assemble(ctx, assembler.Spec{
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
	}{
		{"unknown storage", assembler.Spec{Storage: assembler.StorageSpec{Backend: "bogus"}, Sequencer: assembler.SequencerSpec{Backend: "mem"}}},
		{"unknown sequencer", assembler.Spec{Storage: assembler.StorageSpec{Backend: "mem"}, Sequencer: assembler.SequencerSpec{Backend: "bogus"}}},
		{"fs without dir", assembler.Spec{Storage: assembler.StorageSpec{Backend: "fs"}, Sequencer: assembler.SequencerSpec{Backend: "mem"}}},
		{"file without path", assembler.Spec{Storage: assembler.StorageSpec{Backend: "mem"}, Sequencer: assembler.SequencerSpec{Backend: "file"}}},
		// "internal" sequencer with no server DB offered is refused.
		{"internal without server db", assembler.Spec{Storage: assembler.StorageSpec{Backend: "mem"}, Sequencer: assembler.SequencerSpec{Backend: "internal", Key: "k"}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := assembler.Assemble(ctx, c.spec, assembler.Deps{}); err == nil {
				t.Fatalf("Assemble(%+v) = nil error, want a rejection", c.spec)
			}
		})
	}
}
