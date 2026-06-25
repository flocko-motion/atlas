// package: assembler / build
// type:    logic
// job:     build one ranke.Archive from a Spec — pick a storage backend (𝒰) and a sequencer backend (B_h) and compose them via ranke.NewArchive
// limits:  builds ONE archive; no lifecycle/registry (-> core); no signing identity — the caller supplies a Contributor at mint time
//
// The assembler is the narrow mechanism that turns one archive's configuration
// into a live ranke.Archive. ranke-go gives two orthogonal seams: a Universe
// (𝒰 — content capacity) from adapter/storage, and a BranchTableHead (B_h —
// the single mutable CAS cell) from adapter/sequencer. The assembler selects a
// backend for each from the Spec and composes them. The server core (->core)
// calls this per configured archive and owns the lifecycle around the result.
package assembler

import (
	"context"
	"fmt"

	ranke "github.com/flocko-motion/ranke-go"
	seqfile "github.com/flocko-motion/ranke-go/adapter/sequencer/file"
	seqmem "github.com/flocko-motion/ranke-go/adapter/sequencer/mem"
	storfs "github.com/flocko-motion/ranke-go/adapter/storage/fs"
	stormem "github.com/flocko-motion/ranke-go/adapter/storage/mem"
)

// StorageSpec selects and configures the Universe (𝒰) backend.
type StorageSpec struct {
	Backend string // "mem" | "fs"
	Dir     string // fs: the directory holding claims + content blobs
}

// SequencerSpec selects and configures the BranchTableHead (B_h) backend.
type SequencerSpec struct {
	Backend string // "mem" | "file"
	Path    string // file: path to the B_h cell
}

// Spec is the build recipe for one archive: which 𝒰 and which B_h.
type Spec struct {
	Storage   StorageSpec
	Sequencer SequencerSpec
}

// Assemble builds one ranke.Archive from spec: a Universe from the storage
// backend, a BranchTableHead from the sequencer backend, composed via
// ranke.NewArchive. The signing identity used to mint (a ranke.Contributor) is
// supplied by the caller at AddGraph time, not here.
func Assemble(ctx context.Context, spec Spec) (ranke.Archive, error) {
	u, err := buildUniverse(spec.Storage)
	if err != nil {
		return nil, fmt.Errorf("assembler: storage: %w", err)
	}
	bth, err := buildSequencer(spec.Sequencer)
	if err != nil {
		return nil, fmt.Errorf("assembler: sequencer: %w", err)
	}
	a, err := ranke.NewArchive(ctx, u, bth)
	if err != nil {
		return nil, fmt.Errorf("assembler: archive: %w", err)
	}
	return a, nil
}

// buildUniverse selects the 𝒰 backend. More backends (sqlite, s3) slot in here.
func buildUniverse(s StorageSpec) (ranke.Universe, error) {
	switch s.Backend {
	case "mem":
		return stormem.New(), nil
	case "fs":
		if s.Dir == "" {
			return nil, fmt.Errorf("fs storage needs a dir")
		}
		return storfs.New(s.Dir)
	default:
		return nil, fmt.Errorf("unknown storage backend %q", s.Backend)
	}
}

// buildSequencer selects the B_h backend. A Postgres-backed CAS cell (via
// sequencer.New) slots in here when the core wires its own DB.
func buildSequencer(s SequencerSpec) (ranke.BranchTableHead, error) {
	switch s.Backend {
	case "mem":
		return seqmem.New(), nil
	case "file":
		if s.Path == "" {
			return nil, fmt.Errorf("file sequencer needs a path")
		}
		return seqfile.New(s.Path)
	default:
		return nil, fmt.Errorf("unknown sequencer backend %q", s.Backend)
	}
}
