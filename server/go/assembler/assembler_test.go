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
	a, err := assembler.Assemble(ctx, assembler.Spec{
		Storage:   assembler.StorageSpec{Backend: "mem"},
		Sequencer: assembler.SequencerSpec{Backend: "mem"},
	})
	if err != nil {
		t.Fatalf("Assemble(mem,mem): %v", err)
	}
	if bs := a.Branches(ctx); len(bs) != 0 {
		t.Fatalf("fresh archive has %d branches, want 0", len(bs))
	}
	if a.HasBranch(ctx, "main") {
		t.Fatal("fresh archive should not have a 'main' branch")
	}
}

func TestAssembleFsArchive(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	a, err := assembler.Assemble(ctx, assembler.Spec{
		Storage:   assembler.StorageSpec{Backend: "fs", Dir: filepath.Join(dir, "store")},
		Sequencer: assembler.SequencerSpec{Backend: "file", Path: filepath.Join(dir, "head")},
	})
	if err != nil {
		t.Fatalf("Assemble(fs,file): %v", err)
	}
	if bs := a.Branches(ctx); len(bs) != 0 {
		t.Fatalf("fresh fs archive has %d branches, want 0", len(bs))
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
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := assembler.Assemble(ctx, c.spec); err == nil {
				t.Fatalf("Assemble(%+v) = nil error, want a rejection", c.spec)
			}
		})
	}
}
