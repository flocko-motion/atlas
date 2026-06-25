// package: grants / store
// type:    interface
// job:     define the grant-store seam — storage for (subject, scope, role) grants and disabled-subject flags
// limits:  storage only — a sibling of vault (secrets) and config (config), holding authz POLICY data (not secret); decisions are the engine's (-> access)
//
// This is "storage for grants": the records the authz engine reads. It is not a
// vault (grants are policy, not secrets) and not the engine (no decisions here)
// — just the persistence seam, with backends in grants/{mem,postgres}.
package grants

import "context"

// Role is a grant's role within its scope.
type Role string

const (
	// Tenant-scope roles.
	RoleTenantUser  Role = "user"
	RoleTenantAdmin Role = "admin"
	// RA-scope data roles (a ladder: read ⊂ write ⊂ admin).
	RoleRARead  Role = "read"
	RoleRAWrite Role = "write"
	RoleRAAdmin Role = "admin"
	// RoleRAOperator is ORTHOGONAL to the data ladder: it confers lifecycle
	// control and no data access. Grantable at RA scope (one archive) or tenant
	// scope (every archive in the tenant — a watchdog).
	RoleRAOperator Role = "operator"
)

// Scope identifies what a grant or a request targets. RA == "" means the scope
// is the tenant itself; otherwise it is the named archive within Tenant.
type Scope struct {
	Tenant string
	RA     string
}

// Tenant returns a tenant-level scope.
func Tenant(tenant string) Scope { return Scope{Tenant: tenant} }

// Archive returns an RA-level scope (an archive within a tenant).
func Archive(tenant, ra string) Scope { return Scope{Tenant: tenant, RA: ra} }

// IsTenant reports whether s targets a tenant (not a specific archive).
func (s Scope) IsTenant() bool { return s.RA == "" }

// Grant is a single (subject, scope, role) authorization record.
type Grant struct {
	Subject string
	Scope   Scope
	Role    Role
}

// Store persists authorization data. Backends live under grants/: mem
// (tests/dev), postgres (production, via schemaf's DB). Grants are additive
// (MySQL-style — a collection): a subject may hold several roles on one scope.
type Store interface {
	// Disabled reports whether a subject is disabled (denied despite a valid token).
	Disabled(ctx context.Context, subject string) (bool, error)
	// GrantsFor returns all grants held by a subject (across tenants).
	GrantsFor(ctx context.Context, subject string) ([]Grant, error)
	// PutGrant adds the (subject, scope, role) grant, idempotently — it adds the
	// role without removing others on that scope. Lazy: the subject need not
	// exist beforehand.
	PutGrant(ctx context.Context, g Grant) error
	// DeleteGrant removes the (subject, scope, role) grant if present, leaving
	// any other roles the subject holds on that scope intact.
	DeleteGrant(ctx context.Context, subject string, scope Scope, role Role) error
}
