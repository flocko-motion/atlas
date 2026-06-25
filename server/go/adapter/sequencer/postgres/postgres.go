// package: postgres / sequencer
// type:    adapter
// job:     persist B_h (the branch-table head Id) as a compare-and-swap row in Postgres — ranke-db as the deployment's sequencing point
// limits:  stores 256 bits, not content (-> adapter/storage in ranke-go); the seam is ranke-go's ranke.BranchTableHead, built via sequencer.New
//
// B_h is the single mutable handle in an otherwise immutable, content-addressed
// system. ranke-go's sequencer.New turns a load/save pair into a
// BranchTableHead, so a Postgres row backs it with no dedicated ranke-go
// adapter. Each archive's head is one row keyed by an opaque archive key.
//
// Two constructors, mirroring the config/access stores:
//   - New(ctx, dsn, key) — owns its connection (closed on Close).
//   - NewWithConn(ctx, conn, key) — borrows a connection (e.g. schemaf's
//     built-in Postgres), so the whole deployment sequences through one DB.
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
// NewWithConn(ctx, schemafdb.DB, key) to sequence through schemaf's built-in
// Postgres. The connection is owned by the caller, so Close does not close it.
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

	// load returns nil when no head is set yet; save persists, or clears on nil.
	load := func(ctx context.Context) (ranke.Id, error) {
		var s string
		err := conn().QueryRowContext(ctx, `SELECT head FROM `+table+` WHERE key = $1`, key).Scan(&s)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("adapter/sequencer/postgres: load: %w", err)
		}
		return ranke.ParseId(s)
	}
	save := func(ctx context.Context, id ranke.Id) error {
		if id == nil {
			_, err := conn().ExecContext(ctx, `DELETE FROM `+table+` WHERE key = $1`, key)
			return err
		}
		_, err := conn().ExecContext(ctx,
			`INSERT INTO `+table+` (key, head) VALUES ($1, $2)
			 ON CONFLICT (key) DO UPDATE SET head = EXCLUDED.head`,
			key, id.String(),
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
