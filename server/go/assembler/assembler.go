// package: assembler / build
// type:    logic
// job:     build one ranke.Archive from a Spec — pick a storage backend (𝒰) and a sequencer backend and compose them via ranke.NewArchive
// limits:  builds ONE archive; no lifecycle/registry (-> core); no signing identity — the caller supplies a Contributor at mint time
//
// The assembler is the narrow mechanism that turns one archive's configuration
// into a live ranke.Archive. ranke-go gives two orthogonal seams: a Universe
// (𝒰 — content capacity) from adapter/storage, and a Sequencer — the record of
// B_h's history (the ordered sequence of branch-table head revisions; its
// latest entry is the current head) from adapter/sequencer. (ranke-go types the
// sequencer seam as ranke.BranchTableHead: Load reads the tip, Save appends a
// revision. Lightweight backends keep only the tip — mem/file — while postgres
// keeps the full sequence.) The assembler selects a backend for each from the
// Spec and composes them. The server core (->core) calls this per configured
// archive and owns the lifecycle around it.
package assembler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	ranke "github.com/flocko-motion/ranke-go"
	seqfile "github.com/flocko-motion/ranke-go/adapter/sequencer/file"
	seqmem "github.com/flocko-motion/ranke-go/adapter/sequencer/mem"
	storfs "github.com/flocko-motion/ranke-go/adapter/storage/fs"
	stormem "github.com/flocko-motion/ranke-go/adapter/storage/mem"
	storsqlite "github.com/flocko-motion/ranke-go/adapter/storage/sqlite"

	pgseq "rankedb/adapter/sequencer/postgres"
)

// StorageSpec selects and configures the Universe (𝒰) backend.
type StorageSpec struct {
	Backend string // "mem" | "fs" | "sqlite"
	Dir     string // fs: the directory holding claims + content blobs
	DSN     string // sqlite: data source name
}

// SequencerSpec selects and configures the sequencer backend — the record of
// B_h's history, whose latest entry is the current branch-table head. There is
// no default: B_h is the archive's key (the Universe is unloadable without it),
// so where the sequencer keeps it must be chosen explicitly.
type SequencerSpec struct {
	Backend string // "mem" | "file" | "postgres" | "internal"
	Path    string // file: path to the cell
	DSN     string // postgres: connection string (an external DB) for the cell
	Key     string // postgres/internal: row key identifying this archive's head
}

// Spec is the build recipe for one archive: which 𝒰 and which sequencer.
type Spec struct {
	Storage   StorageSpec
	Sequencer SequencerSpec
}

// Deps carries live server resources that opt-in backends may draw on. The
// default backends (mem/fs/sqlite/file/external-postgres) ignore it; only the
// "internal" backend choices use InternalDB — the server's own Postgres, which
// it offers (never imposes) as a place to colocate archive data.
type Deps struct {
	InternalDB func() *sql.DB // nil unless the server offers its internal DB
}

// Handle is an assembled archive plus the means to release it. Close shuts the
// underlying Universe and sequencer — core calls it to stop an archive (the
// ranke.Archive does not own those backends, so it cannot close them).
type Handle struct {
	Archive ranke.Archive
	u       ranke.Universe
	seq     ranke.BranchTableHead // the sequencer (persists B_h)
}

// Close releases the archive's backends (best-effort: both are attempted).
func (h *Handle) Close() error {
	var errs []error
	if h.seq != nil {
		if err := h.seq.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if h.u != nil {
		if err := h.u.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Assemble builds one archive from spec: a Universe from the storage backend, a
// sequencer from the sequencer backend, composed via ranke.NewArchive. deps
// supplies server resources for opt-in "internal" backends. The signing
// identity used to mint (a ranke.Contributor) is supplied by the caller at
// AddGraph time, not here. On any failure the partially-built backends are
// closed before returning.
func Assemble(ctx context.Context, spec Spec, deps Deps) (*Handle, error) {
	u, err := buildUniverse(spec.Storage)
	if err != nil {
		return nil, fmt.Errorf("assembler: storage: %w", err)
	}
	seq, err := buildSequencer(ctx, spec.Sequencer, deps)
	if err != nil {
		_ = u.Close()
		return nil, fmt.Errorf("assembler: sequencer: %w", err)
	}
	a, err := ranke.NewArchive(ctx, u, seq)
	if err != nil {
		_ = seq.Close()
		_ = u.Close()
		return nil, fmt.Errorf("assembler: archive: %w", err)
	}
	return &Handle{Archive: a, u: u, seq: seq}, nil
}

// buildUniverse selects the 𝒰 backend. More backends (s3, …) slot in here.
func buildUniverse(s StorageSpec) (ranke.Universe, error) {
	switch s.Backend {
	case "mem":
		return stormem.New(), nil
	case "fs":
		if s.Dir == "" {
			return nil, fmt.Errorf("fs storage needs a dir")
		}
		return storfs.New(s.Dir)
	case "sqlite":
		if s.DSN == "" {
			return nil, fmt.Errorf("sqlite storage needs a dsn")
		}
		return storsqlite.New(s.DSN)
	default:
		return nil, fmt.Errorf("unknown storage backend %q", s.Backend)
	}
}

// buildSequencer selects the sequencer backend (the cell persisting B_h).
// "postgres" is an external DB named by the archive's config; "internal" is the
// opt-in offer to keep B_h in the server's own Postgres (requires deps.InternalDB).
func buildSequencer(ctx context.Context, s SequencerSpec, deps Deps) (ranke.BranchTableHead, error) {
	switch s.Backend {
	case "mem":
		return seqmem.New(), nil
	case "file":
		if s.Path == "" {
			return nil, fmt.Errorf("file sequencer needs a path")
		}
		return seqfile.New(s.Path)
	case "postgres":
		if s.DSN == "" || s.Key == "" {
			return nil, fmt.Errorf("postgres sequencer needs a dsn and a key")
		}
		return pgseq.New(ctx, s.DSN, s.Key)
	case "internal":
		if deps.InternalDB == nil {
			return nil, fmt.Errorf("internal sequencer requires the server DB, which this server does not offer")
		}
		if s.Key == "" {
			return nil, fmt.Errorf("internal sequencer needs a key")
		}
		return pgseq.NewWithConn(ctx, deps.InternalDB, s.Key)
	default:
		return nil, fmt.Errorf("unknown sequencer backend %q", s.Backend)
	}
}
