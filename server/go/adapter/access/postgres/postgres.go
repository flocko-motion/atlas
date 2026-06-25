// package: postgres / store
// type:    adapter
// job:     persist access grants + the disabled set in Postgres (an access.Store)
// limits:  storage only — decisions are the engine's (-> access); identity is schemaf's, we key on the opaque subject
//
// One store, two constructors:
//
//   - New(ctx, dsn) — opens and owns its own connection (any external Postgres).
//   - NewWithConn(ctx, conn) — reuses an existing connection from conn, e.g.
//     NewWithConn(ctx, schemafdb.DB) for the framework's built-in Postgres.
//
// Schema is applied via schemaf's scoped single-set migration runner
// (db.RunSet), so external and built-in Postgres share the exact same
// versioned, tracked migration. It stores the two facts the access engine
// needs: which subjects are disabled (ranke_subjects) and which grants they
// hold (ranke_grants).
package postgres

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	schemafdb "github.com/flocko-motion/schemaf/db"

	"rankedb/access"

	_ "github.com/lib/pq"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// migrations is the access schema as a schemaf MigrationSet, applied to
// whichever connection backs the store.
//
// Prefix namespaces our versions in the shared schemaf_migrations table
// (alongside schemaf's own, the app's, and the config adapter's migrations). It
// is a PERMANENT identity — it MUST NEVER be changed for the lifetime of the
// project. Renaming it makes schemaf believe these migrations were never
// applied and re-run them against an already-migrated database, which fails.
var migrations = schemafdb.MigrationSet{
	Prefix: "rankeaccess", // MUST NEVER CHANGE — permanent migration namespace
	Files:  migrationFiles,
}

const (
	subjectsTable = "ranke_subjects"
	grantsTable   = "ranke_grants"
)

// Store is the Postgres-backed access.Store. It also exposes SetDisabled (a
// management op, not part of the read-side access.Store interface) and Close.
type Store struct {
	conn    func() *sql.DB // resolves the live connection on each use
	closeFn func() error   // nil for a borrowed connection
}

var _ access.Store = (*Store)(nil)

// New opens an access Store backed by the Postgres at dsn. The connection is
// owned by the Store and closed on Close.
func New(ctx context.Context, dsn string) (*Store, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("access/postgres.New: open: %w", err)
	}
	s, err := newStore(ctx, func() *sql.DB { return db }, db.Close)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// NewWithConn builds an access Store over an existing connection supplied by
// conn — e.g. NewWithConn(ctx, schemafdb.DB) to reuse the framework's built-in
// Postgres. The connection is owned by the caller, so Close does not close it.
func NewWithConn(ctx context.Context, conn func() *sql.DB) (*Store, error) {
	return newStore(ctx, conn, nil)
}

func newStore(ctx context.Context, conn func() *sql.DB, closeFn func() error) (*Store, error) {
	s := &Store{conn: conn, closeFn: closeFn}
	if err := schemafdb.RunSet(ctx, s.conn(), migrations); err != nil {
		return nil, fmt.Errorf("access/postgres: migrate: %w", err)
	}
	return s, nil
}

// Disabled reports whether subject is disabled. A subject with no row is not
// disabled (lazy provisioning — rows appear only when something is recorded).
func (s *Store) Disabled(ctx context.Context, subject string) (bool, error) {
	var disabled bool
	err := s.conn().QueryRowContext(ctx,
		`SELECT disabled FROM `+subjectsTable+` WHERE id = $1`, subject,
	).Scan(&disabled)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("access/postgres: disabled: %w", err)
	}
	return disabled, nil
}

// GrantsFor returns all grants held by a subject (across tenants).
func (s *Store) GrantsFor(ctx context.Context, subject string) ([]access.Grant, error) {
	rows, err := s.conn().QueryContext(ctx,
		`SELECT scope_tenant, scope_ra, role FROM `+grantsTable+` WHERE subject = $1`, subject,
	)
	if err != nil {
		return nil, fmt.Errorf("access/postgres: grantsFor: %w", err)
	}
	defer rows.Close()
	var out []access.Grant
	for rows.Next() {
		var tenant, ra, role string
		if err := rows.Scan(&tenant, &ra, &role); err != nil {
			return nil, err
		}
		out = append(out, access.Grant{
			Subject: subject,
			Scope:   access.Scope{Tenant: tenant, RA: ra},
			Role:    access.Role(role),
		})
	}
	return out, rows.Err()
}

// PutGrant adds the (subject, scope, role) grant idempotently. Grants are
// additive: the conflict target is the full tuple, so adding a role leaves any
// other roles the subject holds on that scope intact.
func (s *Store) PutGrant(ctx context.Context, g access.Grant) error {
	_, err := s.conn().ExecContext(ctx,
		`INSERT INTO `+grantsTable+` (subject, scope_tenant, scope_ra, role)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (subject, scope_tenant, scope_ra, role) DO NOTHING`,
		g.Subject, g.Scope.Tenant, g.Scope.RA, string(g.Role),
	)
	if err != nil {
		return fmt.Errorf("access/postgres: putGrant: %w", err)
	}
	return nil
}

// DeleteGrant removes the (subject, scope, role) grant if present, leaving any
// other roles the subject holds on that scope intact.
func (s *Store) DeleteGrant(ctx context.Context, subject string, scope access.Scope, role access.Role) error {
	_, err := s.conn().ExecContext(ctx,
		`DELETE FROM `+grantsTable+` WHERE subject = $1 AND scope_tenant = $2 AND scope_ra = $3 AND role = $4`,
		subject, scope.Tenant, scope.RA, string(role),
	)
	if err != nil {
		return fmt.Errorf("access/postgres: deleteGrant: %w", err)
	}
	return nil
}

// SetDisabled records whether a subject is disabled (a management op). It is
// not part of the read-side access.Store interface.
func (s *Store) SetDisabled(ctx context.Context, subject string, disabled bool) error {
	_, err := s.conn().ExecContext(ctx,
		`INSERT INTO `+subjectsTable+` (id, disabled) VALUES ($1, $2)
		 ON CONFLICT (id) DO UPDATE SET disabled = EXCLUDED.disabled`,
		subject, disabled,
	)
	if err != nil {
		return fmt.Errorf("access/postgres: setDisabled: %w", err)
	}
	return nil
}

// Close closes the owned connection (a no-op for a borrowed one).
func (s *Store) Close() error {
	if s.closeFn != nil {
		return s.closeFn()
	}
	return nil
}
