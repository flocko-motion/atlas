package api

import (
	"context"
	"errors"
	"testing"

	schemafapi "github.com/flocko-motion/schemaf/api"

	"rankedb/access"
	"rankedb/adapter/config"
	configmem "rankedb/adapter/config/mem"
	"rankedb/adapter/grants"
	grantsmem "rankedb/adapter/grants/mem"
	"rankedb/assembler"
	"rankedb/core"
)

// route is the non-generic facet every endpoint shares.
type route interface {
	Method() string
	Path() string
	Auth() bool
}

// TestCapabilityRoutes pins each capability endpoint's HTTP contract (and keeps
// Method/Path/Auth reachable until codegen wires the Provider).
func TestCapabilityRoutes(t *testing.T) {
	cases := map[string]struct {
		e      route
		method string
		path   string
	}{
		"admit":   {AdmitUserEndpoint{}, "POST", "/api/tenants/{tenant}/users"},
		"list":    {ListTenantUsersEndpoint{}, "GET", "/api/tenants/{tenant}/users"},
		"grant":   {GrantRoleEndpoint{}, "POST", "/api/tenants/{tenant}/users/{subject}/grants"},
		"revoke":  {RevokeRoleEndpoint{}, "DELETE", "/api/tenants/{tenant}/users/{subject}/grants"},
		"subs":    {ListSubjectsEndpoint{}, "GET", "/api/users"},
		"disable": {SetUserDisabledEndpoint{}, "PATCH", "/api/users/{subject}"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if c.e.Method() != c.method || c.e.Path() != c.path {
				t.Fatalf("%s = %s %s, want %s %s", name, c.e.Method(), c.e.Path(), c.method, c.path)
			}
			if !c.e.Auth() {
				t.Fatalf("%s must require auth", name)
			}
		})
	}
}

func TestGroupBySubject(t *testing.T) {
	gs := []grants.Grant{
		{Subject: "a", Scope: grants.Tenant("t"), Role: grants.RoleTenantUser},
		{Subject: "a", Scope: grants.Archive("t", "main"), Role: grants.RoleRAWrite},
		{Subject: "b", Scope: grants.Tenant("t"), Role: grants.RoleTenantAdmin},
	}
	users := groupBySubject(gs)
	if len(users) != 2 || users[0].Subject != "a" || len(users[0].Roles) != 2 || users[1].Subject != "b" {
		t.Fatalf("groupBySubject = %+v", users)
	}
}

func TestAdmitValidation(t *testing.T) {
	// Empty subject → 400, before any authz call.
	_, err := AdmitUserEndpoint{}.Handle(context.Background(), AdmitUserReq{Tenant: "acme"})
	if !errors.Is(err, schemafapi.ErrBadRequest) {
		t.Fatalf("empty subject should be 400; got %v", err)
	}
}

func TestGrantRejectsUnknownRole(t *testing.T) {
	_, err := GrantRoleEndpoint{}.Handle(context.Background(),
		GrantRoleReq{Tenant: "acme", Subject: "x", Role: "wizard"})
	if !errors.Is(err, schemafapi.ErrBadRequest) {
		t.Fatalf("unknown role should be 400; got %v", err)
	}
}

func TestAdmitNeedsTenantAdmin(t *testing.T) {
	c := core.New(access.New(nil, grantsmem.New()), configmem.NewFrom(config.Entries{}), assembler.Deps{})
	if err := c.Reconcile(context.Background()); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	Use(c)
	// No authenticated actor → not a tenant-admin → 403.
	_, err := AdmitUserEndpoint{}.Handle(context.Background(),
		AdmitUserReq{Tenant: "acme", Subject: "newbie"})
	if !errors.Is(err, schemafapi.ErrForbidden) {
		t.Fatalf("admit without tenant-admin should be 403; got %v", err)
	}
}
