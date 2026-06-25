// package: access / engine
// type:    logic
// job:     decide whether a subject may perform an action on a scope ("may A do B on C?")
// limits:  authz only — authn is schemaf's, secrets are the vault's; never touches the verifiable data model (anyone may verify). Reads grants from the store (-> adapter/grants), holds none itself.
//
// schemaf does authentication (a JWT over an opaque subject); this package
// decides what an authenticated subject may do, enforced at the API — it is
// minting-side server policy only.
//
// The engine is queryable: an access question is a Request ("may Subject do
// Action on Scope?"), an answer is a Decision (Allowed + a Reason). Decide
// returns the Decision; Require is the gate wrapper that turns a deny into an
// error. The grant records the engine reasons over live in adapter/grants; this
// package is pure logic over them.
//
// Model (see openspec/specs/access-policy): the only intrinsic distinction is
// root vs. not. `root` is env-seeded and overrides everything. Every other
// capability is a grant `(subject, scope, role)` (see grants.Role): tenant
// user/admin; RA read⊂write⊂admin plus the orthogonal operator (lifecycle). A
// tenant-admin grant authorises any action within that tenant. Default-deny.
package access

import (
	"context"
	"fmt"
	"strings"

	"rankedb/adapter/grants"
)

// Action is an operation a request wants to perform on a scope.
type Action string

const (
	ReadRA      Action = "ra.read"      // read/query/verify the universe
	WriteRA     Action = "ra.write"     // advance the sequencer (mint)
	ControlRA   Action = "ra.control"   // lifecycle: start/stop/set-readonly/restart
	AdminRA     Action = "ra.admin"     // reconfigure RA, manage its grants
	AdminTenant Action = "tenant.admin" // manage tenant users/grants/RAs/config
)

// Request is an access question: "may Subject do Action on Scope?"
type Request struct {
	Subject string
	Action  Action
	Scope   grants.Scope
}

// Decision is the answer to a Request: whether it is allowed, and why (the
// reason is useful for audit logs and for explaining a denial).
type Decision struct {
	Allowed bool
	Reason  string // "root" | "tenant-admin" | "tenant-operator" | "grant" | "no grant" | "disabled" | "unauthenticated"
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

// Authz resolves access decisions against the env root set and the grant store.
type Authz struct {
	roots map[string]struct{}
	store grants.Store
}

// New builds an Authz. rootSubjects come from the env (RANKE_ROOT_SUBJECT) and
// override all checks — break-glass, evaluated before the store.
func New(rootSubjects []string, store grants.Store) *Authz {
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

// Visible reports whether subject can see anything in tenant — root, or holding
// any grant scoped to that tenant. The API uses it for existence-hiding: a
// subject with no visibility into a tenant gets 404 (not 403) for its resources.
func (a *Authz) Visible(ctx context.Context, subject, tenant string) (bool, error) {
	if subject == "" {
		return false, nil
	}
	if a.IsRoot(subject) {
		return true, nil
	}
	held, err := a.store.GrantsFor(ctx, subject)
	if err != nil {
		return false, err
	}
	for _, g := range held {
		if g.Scope.Tenant == tenant {
			return true, nil
		}
	}
	return false, nil
}

// Decide answers a Request. The error is non-nil only on a store failure; a
// refusal is a Decision{Allowed: false}, not an error. Order: root → disabled
// → tenant-admin/operator of the scope's tenant → matching RA grant → deny.
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
	held, err := a.store.GrantsFor(ctx, req.Subject)
	if err != nil {
		return Decision{}, err
	}
	for _, g := range held {
		// A tenant-scope grant: admin authorises any action in the tenant; an
		// operator authorises lifecycle control of any archive in the tenant.
		if g.Scope.IsTenant() && g.Scope.Tenant == req.Scope.Tenant {
			if g.Role == grants.RoleTenantAdmin {
				return Decision{Allowed: true, Reason: "tenant-admin"}, nil
			}
			if g.Role == grants.RoleRAOperator && req.Action == ControlRA {
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
func raRoleAllows(role grants.Role, action Action) bool {
	switch action {
	case ReadRA:
		return role == grants.RoleRARead || role == grants.RoleRAWrite || role == grants.RoleRAAdmin
	case WriteRA:
		return role == grants.RoleRAWrite || role == grants.RoleRAAdmin
	case ControlRA:
		// Orthogonal: operator (lifecycle only) or admin (RA superuser); NOT read/write.
		return role == grants.RoleRAOperator || role == grants.RoleRAAdmin
	case AdminRA:
		return role == grants.RoleRAAdmin
	default:
		return false // tenant actions are not satisfiable by RA roles
	}
}

// Grant creates/updates a grant, enforcing that actor may manage g's tenant
// (tenant-admin or root). This is the single guarded entry point for modifying
// authorization — a tenant `user` cannot reach it.
func (a *Authz) Grant(ctx context.Context, actor string, g grants.Grant) error {
	if err := a.Require(ctx, Request{Subject: actor, Action: AdminTenant, Scope: grants.Tenant(g.Scope.Tenant)}); err != nil {
		return err
	}
	return a.store.PutGrant(ctx, g)
}

// Revoke removes one (subject, scope, role) grant, enforcing the same
// tenant-management authority as Grant. Other roles the subject holds on that
// scope are left intact (grants are additive).
func (a *Authz) Revoke(ctx context.Context, actor, subject string, scope grants.Scope, role grants.Role) error {
	if err := a.Require(ctx, Request{Subject: actor, Action: AdminTenant, Scope: grants.Tenant(scope.Tenant)}); err != nil {
		return err
	}
	return a.store.DeleteGrant(ctx, subject, scope, role)
}

// Admit makes subject a member of tenant — the baseline tenant-`user`
// capability ("enter as valid"), the first onboarding step. Gated by
// tenant-admin (via Grant). Role grants build on it.
func (a *Authz) Admit(ctx context.Context, actor, tenant, subject string) error {
	return a.Grant(ctx, actor, grants.Grant{Subject: subject, Scope: grants.Tenant(tenant), Role: grants.RoleTenantUser})
}

// TenantUsers returns the grants scoped to tenant (its users and their roles),
// gated by tenant-admin. Scoped by construction: only this tenant's grants are
// returned — never a subject's affiliations elsewhere.
func (a *Authz) TenantUsers(ctx context.Context, actor, tenant string) ([]grants.Grant, error) {
	if err := a.Require(ctx, Request{Subject: actor, Action: AdminTenant, Scope: grants.Tenant(tenant)}); err != nil {
		return nil, err
	}
	return a.store.GrantsIn(ctx, tenant)
}

// Subjects returns the global subject list — root only (the only principal that
// sees the full cross-tenant picture).
func (a *Authz) Subjects(ctx context.Context, actor string) ([]grants.Subject, error) {
	if !a.IsRoot(actor) {
		return nil, &Denied{Subject: actor, Reason: "root only"}
	}
	return a.store.Subjects(ctx)
}

// SetDisabled enables or disables a subject globally (denied everywhere despite
// a valid token) — root only, since it crosses tenant boundaries.
func (a *Authz) SetDisabled(ctx context.Context, actor, subject string, disabled bool) error {
	if !a.IsRoot(actor) {
		return &Denied{Subject: actor, Reason: "root only"}
	}
	return a.store.SetDisabled(ctx, subject, disabled)
}
