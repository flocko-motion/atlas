// package: postgres / adapter
// type:    adapter
// job:     store config in Postgres (owned DSN or borrowed conn), schema applied via schemaf's tracked migration runner
// limits:  Postgres only — no file/env source (-> file/env); migration prefix is permanent and MUST NEVER change
//
// Package postgres stores the ranke-db server config in Postgres. One
// adapter, two constructors:
//
//   - New(ctx, dsn) — opens and owns its own connection (any external Postgres).
//   - NewWithConn(ctx, conn) — reuses an existing connection from conn, e.g.
//     NewWithConn(ctx, schemafdb.DB) for the framework's built-in Postgres.
//
// Schema is applied via schemaf's scoped single-set migration runner
// (db.RunSet), so external and built-in Postgres share the exact same
// versioned, tracked migration — RunSet creates only the schemaf_migrations
// tracking table plus this set's tables, nothing else of the framework.
package postgres

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	schemafdb "github.com/flocko-motion/schemaf/db"

	"rankedb/adapter/config"

	_ "github.com/lib/pq"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// migrations is the config schema as a schemaf MigrationSet, applied to
// whichever connection backs the store.
//
// Prefix namespaces our versions in the shared schemaf_migrations table
// (alongside schemaf's own and the app's migrations). It is a PERMANENT
// identity — it MUST NEVER be changed for the lifetime of the project.
// Renaming it makes schemaf believe these migrations were never applied and
// re-run them against an already-migrated database, which fails.
var migrations = schemafdb.MigrationSet{
	Prefix: "rankeconfig", // MUST NEVER CHANGE — permanent migration namespace
	Files:  migrationFiles,
}

const table = "ranke_config"

type store struct {
	conn    func() *sql.DB // resolves the live connection on each use
	closeFn func() error   // nil for a borrowed connection
}

// New opens a config Store backed by the Postgres at dsn. The connection is
// owned by the Store and closed on Close.
func New(ctx context.Context, dsn string) (config.Store, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("adapter/config/postgres.New: open: %w", err)
	}
	s, err := newStore(ctx, func() *sql.DB { return db }, db.Close)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// NewWithConn builds a config Store over an existing connection supplied by
// conn — e.g. NewWithConn(ctx, schemafdb.DB) to reuse the framework's
// built-in Postgres. The connection is owned by the caller, so Close does
// not close it.
func NewWithConn(ctx context.Context, conn func() *sql.DB) (config.Store, error) {
	return newStore(ctx, conn, nil)
}

func newStore(ctx context.Context, conn func() *sql.DB, closeFn func() error) (config.Store, error) {
	s := &store{conn: conn, closeFn: closeFn}
	if err := schemafdb.RunSet(ctx, s.conn(), migrations); err != nil {
		return nil, fmt.Errorf("adapter/config/postgres: migrate: %w", err)
	}
	return s, nil
}

func (s *store) Load(ctx context.Context) (config.Entries, error) {
	rows, err := s.conn().QueryContext(ctx, `SELECT key, value FROM `+table)
	if err != nil {
		return nil, fmt.Errorf("adapter/config/postgres: load: %w", err)
	}
	defer rows.Close()
	out := config.Entries{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

// Save replaces the whole config atomically (the config is one document).
func (s *store) Save(ctx context.Context, e config.Entries) error {
	tx, err := s.conn().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM `+table); err != nil {
		return fmt.Errorf("adapter/config/postgres: clear: %w", err)
	}
	for k, v := range e {
		if _, err := tx.ExecContext(ctx, `INSERT INTO `+table+` (key, value) VALUES ($1, $2)`, k, v); err != nil {
			return fmt.Errorf("adapter/config/postgres: insert %q: %w", k, err)
		}
	}
	return tx.Commit()
}

func (s *store) Close() error {
	if s.closeFn != nil {
		return s.closeFn()
	}
	return nil
}
