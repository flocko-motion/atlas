// package: api / users
// type:    adapter
// job:     REST endpoints for capability management — admit subjects to a tenant, list its users, grant/revoke roles
// limits:  thin over the authz engine (via core.Authz()); capability management, not identity; subject labels TBD
package api

import (
	"context"
	"fmt"

	schemafapi "github.com/flocko-motion/schemaf/api"

	"rankedb/adapter/grants"
)

// validRoles is the closed set a grant may name; scope-appropriateness
// (tenant user/admin vs RA read/write/admin/operator) is the engine's concern.
var validRoles = map[string]bool{
	string(grants.RoleTenantUser):  true, // "user"
	string(grants.RoleTenantAdmin): true, // "admin" (== RoleRAAdmin value)
	string(grants.RoleRARead):      true, // "read"
	string(grants.RoleRAWrite):     true, // "write"
	string(grants.RoleRAOperator):  true, // "operator"
}

// scopeFor builds a tenant-scope when ra is empty, else an archive scope.
func scopeFor(tenant, ra string) grants.Scope {
	if ra == "" {
		return grants.Tenant(tenant)
	}
	return grants.Archive(tenant, ra)
}

// RoleEntry is one role a subject holds in a tenant (ra empty = the tenant itself).
type RoleEntry struct {
	RA   string `json:"ra,omitempty"`
	Role string `json:"role"`
}

// TenantUser is a subject and the roles it holds within one tenant.
type TenantUser struct {
	Subject string      `json:"subject"`
	Roles   []RoleEntry `json:"roles"`
}

// groupBySubject collapses a tenant's grants into per-subject role lists,
// preserving first-seen order.
func groupBySubject(gs []grants.Grant) []TenantUser {
	idx := map[string]int{}
	var users []TenantUser
	for _, g := range gs {
		i, ok := idx[g.Subject]
		if !ok {
			idx[g.Subject] = len(users)
			users = append(users, TenantUser{Subject: g.Subject})
			i = len(users) - 1
		}
		users[i].Roles = append(users[i].Roles, RoleEntry{RA: g.Scope.RA, Role: string(g.Role)})
	}
	return users
}

// AdmitUserEndpoint admits a subject into a tenant — the baseline membership
// ("enter as valid"). Gated by tenant-admin.
type AdmitUserEndpoint struct{}

// Method is POST.
func (AdmitUserEndpoint) Method() string { return "POST" }

// Path is the tenant's users collection.
func (AdmitUserEndpoint) Path() string { return "/api/tenants/{tenant}/users" }

// Auth requires a valid JWT.
func (AdmitUserEndpoint) Auth() bool { return true }

// Handle admits the body's subject into the tenant.
func (AdmitUserEndpoint) Handle(ctx context.Context, req AdmitUserReq) (AdmitUserResp, error) {
	actor, _ := schemafapi.Subject(ctx)
	if req.Subject == "" {
		return AdmitUserResp{}, fmt.Errorf("subject is required: %w", schemafapi.ErrBadRequest)
	}
	if err := svc.Authz().Admit(ctx, actor, req.Tenant, req.Subject); err != nil {
		return AdmitUserResp{}, mapErr(err)
	}
	return AdmitUserResp{Tenant: req.Tenant, Subject: req.Subject}, nil
}

// AdmitUserReq names the tenant (path) and the subject to admit (body).
type AdmitUserReq struct {
	Tenant  string `path:"tenant"`
	Subject string `json:"subject"`
}

// AdmitUserResp echoes the admitted membership.
type AdmitUserResp struct {
	Tenant  string `json:"tenant"`
	Subject string `json:"subject"`
}

var _ schemafapi.Endpoint[AdmitUserReq, AdmitUserResp] = AdmitUserEndpoint{}

// ListTenantUsersEndpoint lists the tenant's users and their roles. Gated by
// tenant-admin; scoped — only this tenant's grants, never other affiliations.
type ListTenantUsersEndpoint struct{}

// Method is GET.
func (ListTenantUsersEndpoint) Method() string { return "GET" }

// Path is the tenant's users collection.
func (ListTenantUsersEndpoint) Path() string { return "/api/tenants/{tenant}/users" }

// Auth requires a valid JWT.
func (ListTenantUsersEndpoint) Auth() bool { return true }

// Handle returns the tenant's users grouped with their roles.
func (ListTenantUsersEndpoint) Handle(ctx context.Context, req ListTenantUsersReq) (ListTenantUsersResp, error) {
	actor, _ := schemafapi.Subject(ctx)
	gs, err := svc.Authz().TenantUsers(ctx, actor, req.Tenant)
	if err != nil {
		return ListTenantUsersResp{}, mapErr(err)
	}
	return ListTenantUsersResp{Users: groupBySubject(gs)}, nil
}

// ListTenantUsersReq names the tenant (path).
type ListTenantUsersReq struct {
	Tenant string `path:"tenant"`
}

// ListTenantUsersResp is the tenant's users with their roles.
type ListTenantUsersResp struct {
	Users []TenantUser `json:"users"`
}

var _ schemafapi.Endpoint[ListTenantUsersReq, ListTenantUsersResp] = ListTenantUsersEndpoint{}

// GrantRoleEndpoint grants a role to a tenant user (tenant role when ra is
// empty, else an RA role). Gated by tenant-admin.
type GrantRoleEndpoint struct{}

// Method is POST.
func (GrantRoleEndpoint) Method() string { return "POST" }

// Path is a user's grants collection.
func (GrantRoleEndpoint) Path() string { return "/api/tenants/{tenant}/users/{subject}/grants" }

// Auth requires a valid JWT.
func (GrantRoleEndpoint) Auth() bool { return true }

// Handle grants the body's role (optionally RA-scoped) to the path subject.
func (GrantRoleEndpoint) Handle(ctx context.Context, req GrantRoleReq) (GrantRoleResp, error) {
	actor, _ := schemafapi.Subject(ctx)
	if !validRoles[req.Role] {
		return GrantRoleResp{}, fmt.Errorf("unknown role %q: %w", req.Role, schemafapi.ErrBadRequest)
	}
	g := grants.Grant{Subject: req.Subject, Scope: scopeFor(req.Tenant, req.RA), Role: grants.Role(req.Role)}
	if err := svc.Authz().Grant(ctx, actor, g); err != nil {
		return GrantRoleResp{}, mapErr(err)
	}
	return GrantRoleResp{Subject: req.Subject, RA: req.RA, Role: req.Role}, nil
}

// GrantRoleReq names the tenant + subject (path) and the role + optional ra (body).
type GrantRoleReq struct {
	Tenant  string `path:"tenant"`
	Subject string `path:"subject"`
	RA      string `json:"ra,omitempty"`
	Role    string `json:"role"`
}

// GrantRoleResp echoes the granted role.
type GrantRoleResp struct {
	Subject string `json:"subject"`
	RA      string `json:"ra,omitempty"`
	Role    string `json:"role"`
}

var _ schemafapi.Endpoint[GrantRoleReq, GrantRoleResp] = GrantRoleEndpoint{}

// RevokeRoleEndpoint revokes one role from a tenant user, leaving other roles
// intact (grants are additive). Gated by tenant-admin.
type RevokeRoleEndpoint struct{}

// Method is DELETE.
func (RevokeRoleEndpoint) Method() string { return "DELETE" }

// Path is a user's grants collection.
func (RevokeRoleEndpoint) Path() string { return "/api/tenants/{tenant}/users/{subject}/grants" }

// Auth requires a valid JWT.
func (RevokeRoleEndpoint) Auth() bool { return true }

// Handle revokes the body's role (optionally RA-scoped) from the path subject.
func (RevokeRoleEndpoint) Handle(ctx context.Context, req GrantRoleReq) (GrantRoleResp, error) {
	actor, _ := schemafapi.Subject(ctx)
	if !validRoles[req.Role] {
		return GrantRoleResp{}, fmt.Errorf("unknown role %q: %w", req.Role, schemafapi.ErrBadRequest)
	}
	if err := svc.Authz().Revoke(ctx, actor, req.Subject, scopeFor(req.Tenant, req.RA), grants.Role(req.Role)); err != nil {
		return GrantRoleResp{}, mapErr(err)
	}
	return GrantRoleResp{Subject: req.Subject, RA: req.RA, Role: req.Role}, nil
}

var _ schemafapi.Endpoint[GrantRoleReq, GrantRoleResp] = RevokeRoleEndpoint{}
