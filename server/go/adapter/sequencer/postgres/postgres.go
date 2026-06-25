// package: postgres / sequencer
// type:    adapter
// job:     persist the HISTORY of an archive's B_h (the branch-table head Id) as an append-only sequence in a user-configured external Postgres; the latest entry is the current head
// limits:  stores head ids, not content (-> adapter/storage in ranke-go); the seam is ranke-go's ranke.BranchTableHead (Load reads the tip / Save appends), built via sequencer.New. This is ARCHIVE (use-case) data — it lives wherever the archive's config points, NEVER the server's internal operational DB.
//
// B_h is the single mutable handle in an otherwise immutable, content-addressed
// system, and the SEQUENCER is the record of its history — the ordered sequence
// of head revisions, which is what establishes the total order. So the table is
// append-only (one row per Save, keyed by archive + monotonic seq); Load returns
// the tip. ranke-go's sequencer.New turns the load/save pair into a
// BranchTableHead with no dedicated ranke-go adapter.
//
// Two constructors:
//   - New(ctx, dsn, key) — owns its connection (closed on Close). The common
//     case: the archive's config names an external Postgres by DSN.
//   - NewWithConn(ctx, conn, key) — borrows a connection, e.g. to share one
//     external Postgres across several archives' heads.
package postgres

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	ranke "github.com/flocko-motion/ranke-go"
	"github.com/flocko-motion/ranke-go/adapter/sequencer"
	schemafdb "github.com/flocko-motion/schemaf/db"

	_ "github.com/lib/pq"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// migrations is the sequencer schema as a schemaf MigrationSet. The prefix
// namespaces our versions in the shared schemaf_migrations table; it is a
// PERMANENT identity — it MUST NEVER be changed (renaming re-runs the
// migrations against an already-migrated database, which fails).
var migrations = schemafdb.MigrationSet{
	Prefix: "rankeseq", // MUST NEVER CHANGE — permanent migration namespace
	Files:  migrationFiles,
}

const table = "ranke_sequencer"

// Head is a Postgres-backed ranke.BranchTableHead. Load/Save delegate to the
// row; Close closes the owned connection (a no-op for a borrowed one).
type Head struct {
	inner   ranke.BranchTableHead // from sequencer.New(load, save)
	closeFn func() error          // nil for a borrowed connection
}

var _ ranke.BranchTableHead = (*Head)(nil)

// New opens a Postgres-backed head for the archive identified by key, owning
// its connection.
func New(ctx context.Context, dsn, key string) (*Head, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("adapter/sequencer/postgres.New: open: %w", err)
	}
	h, err := newHead(ctx, func() *sql.DB { return db }, key, db.Close)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return h, nil
}

// NewWithConn builds a head over an existing connection supplied by conn — e.g.
// to share one external Postgres across several archives' heads. The connection
// is owned by the caller, so Close does not close it.
func NewWithConn(ctx context.Context, conn func() *sql.DB, key string) (*Head, error) {
	return newHead(ctx, conn, key, nil)
}

func newHead(ctx context.Context, conn func() *sql.DB, key string, closeFn func() error) (*Head, error) {
	if key == "" {
		return nil, fmt.Errorf("adapter/sequencer/postgres: empty key")
	}
	if err := schemafdb.RunSet(ctx, conn(), migrations); err != nil {
		return nil, fmt.Errorf("adapter/sequencer/postgres: migrate: %w", err)
	}

	// The table is APPEND-ONLY: each Save records a new entry, so the row
	// sequence IS the head's history (what makes this the sequencer). load
	// reads the tip (latest seq); a NULL head tip means the head was cleared.
	load := func(ctx context.Context) (ranke.Id, error) {
		var head sql.NullString
		err := conn().QueryRowContext(ctx,
			`SELECT head FROM `+table+` WHERE key = $1 ORDER BY seq DESC LIMIT 1`, key,
		).Scan(&head)
		if err == sql.ErrNoRows {
			return nil, nil // no history yet
		}
		if err != nil {
			return nil, fmt.Errorf("adapter/sequencer/postgres: load: %w", err)
		}
		if !head.Valid {
			return nil, nil // tip is a clear marker
		}
		return ranke.ParseId(head.String)
	}
	save := func(ctx context.Context, id ranke.Id) error {
		var head sql.NullString
		if id != nil {
			head = sql.NullString{String: id.String(), Valid: true}
		}
		_, err := conn().ExecContext(ctx,
			`INSERT INTO `+table+` (key, head) VALUES ($1, $2)`, key, head,
		)
		return err
	}

	return &Head{inner: sequencer.New(load, save), closeFn: closeFn}, nil
}

// Load returns the current head Id (nil if no branches yet).
func (h *Head) Load(ctx context.Context) (ranke.Id, error) { return h.inner.Load(ctx) }

// Save persists id as the head (nil clears it).
func (h *Head) Save(ctx context.Context, id ranke.Id) error { return h.inner.Save(ctx, id) }

// Close closes the owned connection (a no-op for a borrowed one).
func (h *Head) Close() error {
	if h.closeFn != nil {
		return h.closeFn()
	}
	return nil
}
