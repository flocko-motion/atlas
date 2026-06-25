// package: access / engine
// type:    logic
// job:     decide whether a subject may perform an action on a scope ("may A do B on C?")
// limits:  authz only — authn is schemaf's, secrets are the vault's; never touches the verifiable data model (anyone may verify)
//
// schemaf does authentication (a JWT over an opaque subject); this package
// decides what an authenticated subject may do, enforced at the API — it is
// minting-side server policy only.
//
// The engine is queryable: an access question is a Request ("may Subject do
// Action on Scope?"), an answer is a Decision (Allowed + a Reason). Decide
// returns the Decision; Require is the gate wrapper that turns a deny into an
// error. Requests and Decisions are explicit, so callers express questions and
// inspect answers without touching the engine internals (grants/roles).
//
// Model (see openspec/specs/access-policy): the only intrinsic distinction is
// root vs. not. `root` is env-seeded and overrides everything. Every other
// capability is a grant `(subject, scope, role)`:
//   - tenant scope: role `user` (member; cannot modify users) or `admin`
//     (manage the tenant — its users/grants, RAs, components).
//   - RA scope: role `read` (universe read/query/verify), `write` (+ advance
//     the sequencer / mint), or `admin` (+ reconfigure the RA, manage its grants).
//
// A tenant-admin grant authorises any action within that tenant (including its
// RAs). Default-deny otherwise.
package access

import (
	"context"
	"fmt"
	"strings"
)

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
	// control (ControlRA) and no data access. Grantable at RA scope (one
	// archive) or tenant scope (every archive in the tenant — a watchdog).
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

// Action is an operation a request wants to perform on a scope.
type Action string

const (
	ReadRA      Action = "ra.read"      // read/query/verify the universe
	WriteRA     Action = "ra.write"     // advance the sequencer (mint)
	ControlRA   Action = "ra.control"   // lifecycle: start/stop/set-readonly/restart
	AdminRA     Action = "ra.admin"     // reconfigure RA, manage its grants
	AdminTenant Action = "tenant.admin" // manage tenant users/grants/RAs/config
)

// Grant is a single (subject, scope, role) authorization record.
type Grant struct {
	Subject string
	Scope   Scope
	Role    Role
}

// Request is an access question: "may Subject do Action on Scope?"
type Request struct {
	Subject string
	Action  Action
	Scope   Scope
}

// Decision is the answer to a Request: whether it is allowed, and why (the
// reason is useful for audit logs and for explaining a denial).
type Decision struct {
	Allowed bool
	Reason  string // "root" | "tenant-admin" | "tenant-operator" | "grant" | "no grant" | "disabled" | "unauthenticated"
}

// Store is the persistence seam for authorization data. Backends live under
// adapter/access: mem (tests/dev), postgres (production, via schemaf's DB).
type Store interface {
	// Disabled reports whether a subject is disabled (denied despite a valid token).
	Disabled(ctx context.Context, subject string) (bool, error)
	// GrantsFor returns all grants held by a subject (across tenants).
	GrantsFor(ctx context.Context, subject string) ([]Grant, error)
	// PutGrant adds the (subject, scope, role) grant, idempotently. Grants are
	// additive (MySQL-style — a collection): a subject may hold several roles on
	// one scope, so this adds the role without removing others. Lazy: the
	// subject need not exist beforehand.
	PutGrant(ctx context.Context, g Grant) error
	// DeleteGrant removes the (subject, scope, role) grant if present, leaving
	// any other roles the subject holds on that scope intact.
	DeleteGrant(ctx context.Context, subject string, scope Scope, role Role) error
}

// Denied is returned by Require when a request is refused. It carries the
// subject so the API layer can hand it back in a 403 (the onboarding path).
type Denied struct {
	Subject string
	Reason  string
}

// Error renders the denial, including the subject (for logs and the 403 body).
func (d *Denied) Error() string {
	return fmt.Sprintf("access denied for subject %q: %s", d.Subject, d.Reason)
}

// Authz resolves access decisions against the env root set and the store.
type Authz struct {
	roots map[string]struct{}
	store Store
}

// New builds an Authz. rootSubjects come from the env (RANKE_ROOT_SUBJECT) and
// override all checks — break-glass, evaluated before the store.
func New(rootSubjects []string, store Store) *Authz {
	roots := make(map[string]struct{}, len(rootSubjects))
	for _, s := range rootSubjects {
		if s = strings.TrimSpace(s); s != "" {
			roots[s] = struct{}{}
		}
	}
	return &Authz{roots: roots, store: store}
}

// IsRoot reports whether subject is an env-seeded root.
func (a *Authz) IsRoot(subject string) bool {
	_, ok := a.roots[subject]
	return ok
}

// Decide answers a Request. The error is non-nil only on a store failure; a
// refusal is a Decision{Allowed: false}, not an error. Order: root → disabled
// → tenant-admin of the scope's tenant → matching RA grant → deny.
func (a *Authz) Decide(ctx context.Context, req Request) (Decision, error) {
	if req.Subject == "" {
		return Decision{Allowed: false, Reason: "unauthenticated"}, nil
	}
	if a.IsRoot(req.Subject) {
		return Decision{Allowed: true, Reason: "root"}, nil // break-glass, before the store
	}
	disabled, err := a.store.Disabled(ctx, req.Subject)
	if err != nil {
		return Decision{}, err
	}
	if disabled {
		return Decision{Allowed: false, Reason: "disabled"}, nil
	}
	grants, err := a.store.GrantsFor(ctx, req.Subject)
	if err != nil {
		return Decision{}, err
	}
	for _, g := range grants {
		// A tenant-scope grant: admin authorises any action in the tenant; an
		// operator authorises lifecycle control of any archive in the tenant.
		if g.Scope.IsTenant() && g.Scope.Tenant == req.Scope.Tenant {
			if g.Role == RoleTenantAdmin {
				return Decision{Allowed: true, Reason: "tenant-admin"}, nil
			}
			if g.Role == RoleRAOperator && req.Action == ControlRA {
				return Decision{Allowed: true, Reason: "tenant-operator"}, nil
			}
		}
		// An RA-scope grant authorises matching RA actions on that archive.
		if !req.Scope.IsTenant() && g.Scope == req.Scope && raRoleAllows(g.Role, req.Action) {
			return Decision{Allowed: true, Reason: "grant"}, nil
		}
	}
	return Decision{Allowed: false, Reason: "no grant"}, nil
}

// Require is the enforcement gate: nil if the request is allowed, else a
// *Denied carrying the subject (or a store error).
func (a *Authz) Require(ctx context.Context, req Request) error {
	d, err := a.Decide(ctx, req)
	if err != nil {
		return err
	}
	if !d.Allowed {
		return &Denied{Subject: req.Subject, Reason: d.Reason}
	}
	return nil
}

// raRoleAllows reports whether an RA-scope role permits an RA action.
func raRoleAllows(role Role, action Action) bool {
	switch action {
	case ReadRA:
		return role == RoleRARead || role == RoleRAWrite || role == RoleRAAdmin
	case WriteRA:
		return role == RoleRAWrite || role == RoleRAAdmin
	case ControlRA:
		// Orthogonal: operator (lifecycle only) or admin (RA superuser); NOT read/write.
		return role == RoleRAOperator || role == RoleRAAdmin
	case AdminRA:
		return role == RoleRAAdmin
	default:
		return false // tenant actions are not satisfiable by RA roles
	}
}

// Grant creates/updates a grant, enforcing that actor may manage g's tenant
// (tenant-admin or root). This is the single guarded entry point for modifying
// authorization — a tenant `user` cannot reach it.
func (a *Authz) Grant(ctx context.Context, actor string, g Grant) error {
	if err := a.Require(ctx, Request{Subject: actor, Action: AdminTenant, Scope: Tenant(g.Scope.Tenant)}); err != nil {
		return err
	}
	return a.store.PutGrant(ctx, g)
}

// Revoke removes one (subject, scope, role) grant, enforcing the same
// tenant-management authority as Grant. Other roles the subject holds on that
// scope are left intact (grants are additive).
func (a *Authz) Revoke(ctx context.Context, actor, subject string, scope Scope, role Role) error {
	if err := a.Require(ctx, Request{Subject: actor, Action: AdminTenant, Scope: Tenant(scope.Tenant)}); err != nil {
		return err
	}
	return a.store.DeleteGrant(ctx, subject, scope, role)
}
