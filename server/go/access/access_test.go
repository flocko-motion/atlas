package access_test

import (
	"context"
	"errors"
	"testing"

	"rankedb/access"
	"rankedb/adapter/access/mem"
)

func newAuthz(roots ...string) (*access.Authz, *mem.Store) {
	store := mem.New()
	return access.New(roots, store), store
}

func req(sub string, sc access.Scope, act access.Action) access.Request {
	return access.Request{Subject: sub, Action: act, Scope: sc}
}

func allow(t *testing.T, a *access.Authz, sub string, sc access.Scope, act access.Action) {
	t.Helper()
	if err := a.Require(context.Background(), req(sub, sc, act)); err != nil {
		t.Fatalf("expected ALLOW for %q on %+v/%s, got: %v", sub, sc, act, err)
	}
}

func deny(t *testing.T, a *access.Authz, sub string, sc access.Scope, act access.Action) *access.Denied {
	t.Helper()
	err := a.Require(context.Background(), req(sub, sc, act))
	var d *access.Denied
	if !errors.As(err, &d) {
		t.Fatalf("expected DENY (*Denied) for %q on %+v/%s, got: %v", sub, sc, act, err)
	}
	return d
}

func TestRootOverridesEverythingEvenEmptyStore(t *testing.T) {
	a, _ := newAuthz("root-sub")
	allow(t, a, "root-sub", access.Tenant("A"), access.AdminTenant)
	allow(t, a, "root-sub", access.Archive("A", "main"), access.WriteRA)
}

func TestDefaultDenyCarriesSubject(t *testing.T) {
	a, _ := newAuthz()
	d := deny(t, a, "stranger", access.Archive("A", "main"), access.ReadRA)
	if d.Subject != "stranger" {
		t.Fatalf("Denied.Subject = %q, want stranger (for the 403 onboarding path)", d.Subject)
	}
}

func TestRARolesLadder(t *testing.T) {
	a, store := newAuthz()
	ctx := context.Background()
	ra := access.Archive("A", "main")

	store.PutGrant(ctx, access.Grant{Subject: "r", Scope: ra, Role: access.RoleRARead})
	allow(t, a, "r", ra, access.ReadRA)
	deny(t, a, "r", ra, access.WriteRA)

	store.PutGrant(ctx, access.Grant{Subject: "w", Scope: ra, Role: access.RoleRAWrite})
	allow(t, a, "w", ra, access.ReadRA)
	allow(t, a, "w", ra, access.WriteRA)
	deny(t, a, "w", ra, access.AdminRA)

	store.PutGrant(ctx, access.Grant{Subject: "ad", Scope: ra, Role: access.RoleRAAdmin})
	allow(t, a, "ad", ra, access.AdminRA)
}

func TestTenantAdminAuthorisesWholeTenant(t *testing.T) {
	a, store := newAuthz()
	store.PutGrant(context.Background(), access.Grant{
		Subject: "boss", Scope: access.Tenant("A"), Role: access.RoleTenantAdmin,
	})
	// Manage the tenant, and any RA action on any archive in it — without an RA grant.
	allow(t, a, "boss", access.Tenant("A"), access.AdminTenant)
	allow(t, a, "boss", access.Archive("A", "main"), access.WriteRA)
	allow(t, a, "boss", access.Archive("A", "other"), access.AdminRA)
}

func TestTenantUserCannotManageOrAccessWithoutRAGrant(t *testing.T) {
	a, store := newAuthz()
	store.PutGrant(context.Background(), access.Grant{
		Subject: "u", Scope: access.Tenant("A"), Role: access.RoleTenantUser,
	})
	deny(t, a, "u", access.Tenant("A"), access.AdminTenant)
	deny(t, a, "u", access.Archive("A", "main"), access.ReadRA)
}

func TestMultiTenantIndependentRoles(t *testing.T) {
	a, store := newAuthz()
	ctx := context.Background()
	store.PutGrant(ctx, access.Grant{Subject: "x", Scope: access.Tenant("A"), Role: access.RoleTenantAdmin})
	store.PutGrant(ctx, access.Grant{Subject: "x", Scope: access.Tenant("B"), Role: access.RoleTenantUser})

	allow(t, a, "x", access.Tenant("A"), access.AdminTenant)
	deny(t, a, "x", access.Tenant("B"), access.AdminTenant)
	// Tenant-admin in A does not leak to B's archives.
	deny(t, a, "x", access.Archive("B", "main"), access.WriteRA)
}

func TestDisabledSubjectDeniedDespiteGrant(t *testing.T) {
	a, store := newAuthz()
	store.PutGrant(context.Background(), access.Grant{
		Subject: "gone", Scope: access.Tenant("A"), Role: access.RoleTenantAdmin,
	})
	store.SetDisabled("gone", true)
	deny(t, a, "gone", access.Tenant("A"), access.AdminTenant)
}

func TestGrantAndRevokeEnforceTenantAdmin(t *testing.T) {
	a, store := newAuthz()
	ctx := context.Background()
	store.PutGrant(ctx, access.Grant{Subject: "boss", Scope: access.Tenant("A"), Role: access.RoleTenantAdmin})
	store.PutGrant(ctx, access.Grant{Subject: "u", Scope: access.Tenant("A"), Role: access.RoleTenantUser})

	target := access.Grant{Subject: "newbie", Scope: access.Archive("A", "main"), Role: access.RoleRAWrite}

	// A tenant-user cannot grant.
	if err := a.Grant(ctx, "u", target); err == nil {
		t.Fatal("tenant user must not be able to Grant")
	}
	// A tenant-admin can — and the grant takes effect.
	if err := a.Grant(ctx, "boss", target); err != nil {
		t.Fatalf("tenant admin Grant: %v", err)
	}
	allow(t, a, "newbie", access.Archive("A", "main"), access.WriteRA)

	// Revoke removes it.
	if err := a.Revoke(ctx, "boss", "newbie", access.Archive("A", "main"), access.RoleRAWrite); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	deny(t, a, "newbie", access.Archive("A", "main"), access.WriteRA)
}

// TestDecideExplainsTheAnswer pins the Decision contract: a request maps to an
// allowed/denied answer carrying a stable reason, so callers can audit and
// explain "may A do B on C?" without reaching into the engine.
func TestDecideExplainsTheAnswer(t *testing.T) {
	a, store := newAuthz("root-sub")
	ctx := context.Background()
	store.PutGrant(ctx, access.Grant{Subject: "boss", Scope: access.Tenant("A"), Role: access.RoleTenantAdmin})
	store.PutGrant(ctx, access.Grant{Subject: "r", Scope: access.Archive("A", "main"), Role: access.RoleRARead})
	store.PutGrant(ctx, access.Grant{Subject: "gone", Scope: access.Tenant("A"), Role: access.RoleTenantAdmin})
	store.SetDisabled("gone", true)

	cases := []struct {
		name        string
		r           access.Request
		wantAllowed bool
		wantReason  string
	}{
		{"root", req("root-sub", access.Archive("A", "main"), access.AdminRA), true, "root"},
		{"tenant-admin", req("boss", access.Archive("A", "x"), access.WriteRA), true, "tenant-admin"},
		{"grant", req("r", access.Archive("A", "main"), access.ReadRA), true, "grant"},
		{"no grant", req("r", access.Archive("A", "main"), access.WriteRA), false, "no grant"},
		{"disabled", req("gone", access.Tenant("A"), access.AdminTenant), false, "disabled"},
		{"unauthenticated", req("", access.Archive("A", "main"), access.ReadRA), false, "unauthenticated"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, err := a.Decide(ctx, c.r)
			if err != nil {
				t.Fatalf("Decide: %v", err)
			}
			if d.Allowed != c.wantAllowed || d.Reason != c.wantReason {
				t.Fatalf("Decide = {%v, %q}, want {%v, %q}", d.Allowed, d.Reason, c.wantAllowed, c.wantReason)
			}
		})
	}
}

func TestOperatorControlsLifecycleOnly(t *testing.T) {
	a, store := newAuthz()
	ctx := context.Background()
	ra := access.Archive("A", "main")

	// RA-scope operator: may control lifecycle, may NOT read or mint.
	store.PutGrant(ctx, access.Grant{Subject: "watch", Scope: ra, Role: access.RoleRAOperator})
	allow(t, a, "watch", ra, access.ControlRA)
	deny(t, a, "watch", ra, access.ReadRA)
	deny(t, a, "watch", ra, access.WriteRA)

	// A writer may mint but may NOT control lifecycle (orthogonal).
	store.PutGrant(ctx, access.Grant{Subject: "w", Scope: ra, Role: access.RoleRAWrite})
	allow(t, a, "w", ra, access.WriteRA)
	deny(t, a, "w", ra, access.ControlRA)

	// RA admin is the superuser: controls lifecycle too.
	store.PutGrant(ctx, access.Grant{Subject: "ad", Scope: ra, Role: access.RoleRAAdmin})
	allow(t, a, "ad", ra, access.ControlRA)
}

func TestTenantOperatorControlsEveryArchiveInTenant(t *testing.T) {
	a, store := newAuthz()
	// Tenant-scope operator = a watchdog over all archives in the tenant.
	store.PutGrant(context.Background(), access.Grant{
		Subject: "watch", Scope: access.Tenant("A"), Role: access.RoleRAOperator,
	})
	allow(t, a, "watch", access.Archive("A", "main"), access.ControlRA)
	allow(t, a, "watch", access.Archive("A", "other"), access.ControlRA)
	// But not data, and not archives in another tenant.
	deny(t, a, "watch", access.Archive("A", "main"), access.ReadRA)
	deny(t, a, "watch", access.Archive("B", "main"), access.ControlRA)
}

func TestAdditiveGrantsDoNotClobber(t *testing.T) {
	a, store := newAuthz()
	ctx := context.Background()
	ra := access.Archive("A", "main")

	// Holding write + operator on the same scope: both rights coexist.
	store.PutGrant(ctx, access.Grant{Subject: "s", Scope: ra, Role: access.RoleRAWrite})
	store.PutGrant(ctx, access.Grant{Subject: "s", Scope: ra, Role: access.RoleRAOperator})
	allow(t, a, "s", ra, access.WriteRA)
	allow(t, a, "s", ra, access.ControlRA)

	if gs, _ := store.GrantsFor(ctx, "s"); len(gs) != 2 {
		t.Fatalf("GrantsFor(s) = %d grants, want 2 (write + operator, additive)", len(gs))
	}

	// Revoking operator leaves write intact.
	if err := store.DeleteGrant(ctx, "s", ra, access.RoleRAOperator); err != nil {
		t.Fatalf("DeleteGrant: %v", err)
	}
	allow(t, a, "s", ra, access.WriteRA)
	deny(t, a, "s", ra, access.ControlRA)
}
