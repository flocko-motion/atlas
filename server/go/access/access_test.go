package access_test

import (
	"context"
	"errors"
	"testing"

	"rankedb/access"
	"rankedb/adapter/grants"
	"rankedb/adapter/grants/mem"
)

func newAuthz(roots ...string) (*access.Authz, *mem.Store) {
	store := mem.New()
	return access.New(roots, store), store
}

func req(sub string, sc grants.Scope, act access.Action) access.Request {
	return access.Request{Subject: sub, Action: act, Scope: sc}
}

func allow(t *testing.T, a *access.Authz, sub string, sc grants.Scope, act access.Action) {
	t.Helper()
	if err := a.Require(context.Background(), req(sub, sc, act)); err != nil {
		t.Fatalf("expected ALLOW for %q on %+v/%s, got: %v", sub, sc, act, err)
	}
}

func deny(t *testing.T, a *access.Authz, sub string, sc grants.Scope, act access.Action) *access.Denied {
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
	allow(t, a, "root-sub", grants.Tenant("A"), access.AdminTenant)
	allow(t, a, "root-sub", grants.Archive("A", "main"), access.WriteRA)
}

func TestDefaultDenyCarriesSubject(t *testing.T) {
	a, _ := newAuthz()
	d := deny(t, a, "stranger", grants.Archive("A", "main"), access.ReadRA)
	if d.Subject != "stranger" {
		t.Fatalf("Denied.Subject = %q, want stranger (for the 403 onboarding path)", d.Subject)
	}
}

func TestRARolesLadder(t *testing.T) {
	a, store := newAuthz()
	ctx := context.Background()
	ra := grants.Archive("A", "main")

	store.PutGrant(ctx, grants.Grant{Subject: "r", Scope: ra, Role: grants.RoleRARead})
	allow(t, a, "r", ra, access.ReadRA)
	deny(t, a, "r", ra, access.WriteRA)

	store.PutGrant(ctx, grants.Grant{Subject: "w", Scope: ra, Role: grants.RoleRAWrite})
	allow(t, a, "w", ra, access.ReadRA)
	allow(t, a, "w", ra, access.WriteRA)
	deny(t, a, "w", ra, access.AdminRA)

	store.PutGrant(ctx, grants.Grant{Subject: "ad", Scope: ra, Role: grants.RoleRAAdmin})
	allow(t, a, "ad", ra, access.AdminRA)
}

func TestTenantAdminAuthorisesWholeTenant(t *testing.T) {
	a, store := newAuthz()
	store.PutGrant(context.Background(), grants.Grant{
		Subject: "boss", Scope: grants.Tenant("A"), Role: grants.RoleTenantAdmin,
	})
	// Manage the tenant, and any RA action on any archive in it — without an RA grant.
	allow(t, a, "boss", grants.Tenant("A"), access.AdminTenant)
	allow(t, a, "boss", grants.Archive("A", "main"), access.WriteRA)
	allow(t, a, "boss", grants.Archive("A", "other"), access.AdminRA)
}

func TestTenantUserCannotManageOrAccessWithoutRAGrant(t *testing.T) {
	a, store := newAuthz()
	store.PutGrant(context.Background(), grants.Grant{
		Subject: "u", Scope: grants.Tenant("A"), Role: grants.RoleTenantUser,
	})
	deny(t, a, "u", grants.Tenant("A"), access.AdminTenant)
	deny(t, a, "u", grants.Archive("A", "main"), access.ReadRA)
}

func TestMultiTenantIndependentRoles(t *testing.T) {
	a, store := newAuthz()
	ctx := context.Background()
	store.PutGrant(ctx, grants.Grant{Subject: "x", Scope: grants.Tenant("A"), Role: grants.RoleTenantAdmin})
	store.PutGrant(ctx, grants.Grant{Subject: "x", Scope: grants.Tenant("B"), Role: grants.RoleTenantUser})

	allow(t, a, "x", grants.Tenant("A"), access.AdminTenant)
	deny(t, a, "x", grants.Tenant("B"), access.AdminTenant)
	// Tenant-admin in A does not leak to B's archives.
	deny(t, a, "x", grants.Archive("B", "main"), access.WriteRA)
}

func TestDisabledSubjectDeniedDespiteGrant(t *testing.T) {
	a, store := newAuthz()
	store.PutGrant(context.Background(), grants.Grant{
		Subject: "gone", Scope: grants.Tenant("A"), Role: grants.RoleTenantAdmin,
	})
	store.SetDisabled(context.Background(), "gone", true)
	deny(t, a, "gone", grants.Tenant("A"), access.AdminTenant)
}

func TestGrantAndRevokeEnforceTenantAdmin(t *testing.T) {
	a, store := newAuthz()
	ctx := context.Background()
	store.PutGrant(ctx, grants.Grant{Subject: "boss", Scope: grants.Tenant("A"), Role: grants.RoleTenantAdmin})
	store.PutGrant(ctx, grants.Grant{Subject: "u", Scope: grants.Tenant("A"), Role: grants.RoleTenantUser})

	target := grants.Grant{Subject: "newbie", Scope: grants.Archive("A", "main"), Role: grants.RoleRAWrite}

	// A tenant-user cannot grant.
	if err := a.Grant(ctx, "u", target); err == nil {
		t.Fatal("tenant user must not be able to Grant")
	}
	// A tenant-admin can — and the grant takes effect.
	if err := a.Grant(ctx, "boss", target); err != nil {
		t.Fatalf("tenant admin Grant: %v", err)
	}
	allow(t, a, "newbie", grants.Archive("A", "main"), access.WriteRA)

	// Revoke removes it.
	if err := a.Revoke(ctx, "boss", "newbie", grants.Archive("A", "main"), grants.RoleRAWrite); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	deny(t, a, "newbie", grants.Archive("A", "main"), access.WriteRA)
}

// TestDecideExplainsTheAnswer pins the Decision contract: a request maps to an
// allowed/denied answer carrying a stable reason, so callers can audit and
// explain "may A do B on C?" without reaching into the engine.
func TestDecideExplainsTheAnswer(t *testing.T) {
	a, store := newAuthz("root-sub")
	ctx := context.Background()
	store.PutGrant(ctx, grants.Grant{Subject: "boss", Scope: grants.Tenant("A"), Role: grants.RoleTenantAdmin})
	store.PutGrant(ctx, grants.Grant{Subject: "r", Scope: grants.Archive("A", "main"), Role: grants.RoleRARead})
	store.PutGrant(ctx, grants.Grant{Subject: "gone", Scope: grants.Tenant("A"), Role: grants.RoleTenantAdmin})
	store.SetDisabled(context.Background(), "gone", true)

	cases := []struct {
		name        string
		r           access.Request
		wantAllowed bool
		wantReason  string
	}{
		{"root", req("root-sub", grants.Archive("A", "main"), access.AdminRA), true, "root"},
		{"tenant-admin", req("boss", grants.Archive("A", "x"), access.WriteRA), true, "tenant-admin"},
		{"grant", req("r", grants.Archive("A", "main"), access.ReadRA), true, "grant"},
		{"no grant", req("r", grants.Archive("A", "main"), access.WriteRA), false, "no grant"},
		{"disabled", req("gone", grants.Tenant("A"), access.AdminTenant), false, "disabled"},
		{"unauthenticated", req("", grants.Archive("A", "main"), access.ReadRA), false, "unauthenticated"},
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
	ra := grants.Archive("A", "main")

	// RA-scope operator: may control lifecycle, may NOT read or mint.
	store.PutGrant(ctx, grants.Grant{Subject: "watch", Scope: ra, Role: grants.RoleRAOperator})
	allow(t, a, "watch", ra, access.ControlRA)
	deny(t, a, "watch", ra, access.ReadRA)
	deny(t, a, "watch", ra, access.WriteRA)

	// A writer may mint but may NOT control lifecycle (orthogonal).
	store.PutGrant(ctx, grants.Grant{Subject: "w", Scope: ra, Role: grants.RoleRAWrite})
	allow(t, a, "w", ra, access.WriteRA)
	deny(t, a, "w", ra, access.ControlRA)

	// RA admin is the superuser: controls lifecycle too.
	store.PutGrant(ctx, grants.Grant{Subject: "ad", Scope: ra, Role: grants.RoleRAAdmin})
	allow(t, a, "ad", ra, access.ControlRA)
}

func TestTenantOperatorControlsEveryArchiveInTenant(t *testing.T) {
	a, store := newAuthz()
	// Tenant-scope operator = a watchdog over all archives in the tenant.
	store.PutGrant(context.Background(), grants.Grant{
		Subject: "watch", Scope: grants.Tenant("A"), Role: grants.RoleRAOperator,
	})
	allow(t, a, "watch", grants.Archive("A", "main"), access.ControlRA)
	allow(t, a, "watch", grants.Archive("A", "other"), access.ControlRA)
	// But not data, and not archives in another tenant.
	deny(t, a, "watch", grants.Archive("A", "main"), access.ReadRA)
	deny(t, a, "watch", grants.Archive("B", "main"), access.ControlRA)
}

func TestAdditiveGrantsDoNotClobber(t *testing.T) {
	a, store := newAuthz()
	ctx := context.Background()
	ra := grants.Archive("A", "main")

	// Holding write + operator on the same scope: both rights coexist.
	store.PutGrant(ctx, grants.Grant{Subject: "s", Scope: ra, Role: grants.RoleRAWrite})
	store.PutGrant(ctx, grants.Grant{Subject: "s", Scope: ra, Role: grants.RoleRAOperator})
	allow(t, a, "s", ra, access.WriteRA)
	allow(t, a, "s", ra, access.ControlRA)

	if gs, _ := store.GrantsFor(ctx, "s"); len(gs) != 2 {
		t.Fatalf("GrantsFor(s) = %d grants, want 2 (write + operator, additive)", len(gs))
	}

	// Revoking operator leaves write intact.
	if err := store.DeleteGrant(ctx, "s", ra, grants.RoleRAOperator); err != nil {
		t.Fatalf("DeleteGrant: %v", err)
	}
	allow(t, a, "s", ra, access.WriteRA)
	deny(t, a, "s", ra, access.ControlRA)
}

func TestAdmitAndTenantUsers(t *testing.T) {
	a, store := newAuthz()
	ctx := context.Background()
	store.PutGrant(ctx, grants.Grant{Subject: "boss", Scope: grants.Tenant("A"), Role: grants.RoleTenantAdmin})

	// A tenant-user cannot admit; a tenant-admin can.
	store.PutGrant(ctx, grants.Grant{Subject: "u", Scope: grants.Tenant("A"), Role: grants.RoleTenantUser})
	if err := a.Admit(ctx, "u", "A", "newbie"); err == nil {
		t.Fatal("tenant user must not admit")
	}
	if err := a.Admit(ctx, "boss", "A", "newbie"); err != nil {
		t.Fatalf("Admit: %v", err)
	}
	// newbie is now a member (visible).
	if v, _ := a.Visible(ctx, "newbie", "A"); !v {
		t.Fatal("admitted subject should be visible in the tenant")
	}

	// TenantUsers lists this tenant's grants for the admin, and only those.
	gs, err := a.TenantUsers(ctx, "boss", "A")
	if err != nil {
		t.Fatalf("TenantUsers: %v", err)
	}
	if len(gs) == 0 {
		t.Fatal("TenantUsers should include boss/u/newbie grants")
	}
	for _, g := range gs {
		if g.Scope.Tenant != "A" {
			t.Fatalf("TenantUsers leaked a non-A grant: %+v", g)
		}
	}
	// A non-admin cannot list.
	if _, err := a.TenantUsers(ctx, "u", "A"); err == nil {
		t.Fatal("tenant user must not list users")
	}
}

func TestSubjectsAndDisableAreRootOnly(t *testing.T) {
	a, store := newAuthz("root-sub")
	ctx := context.Background()
	store.PutGrant(ctx, grants.Grant{Subject: "boss", Scope: grants.Tenant("A"), Role: grants.RoleTenantAdmin})

	// Subjects: root yes, tenant-admin no.
	if _, err := a.Subjects(ctx, "root-sub"); err != nil {
		t.Fatalf("root Subjects: %v", err)
	}
	if _, err := a.Subjects(ctx, "boss"); err == nil {
		t.Fatal("non-root must not list all subjects")
	}

	// SetDisabled: root only, and it takes effect.
	if err := a.SetDisabled(ctx, "boss", "boss", false); err == nil {
		t.Fatal("non-root must not disable")
	}
	if err := a.SetDisabled(ctx, "root-sub", "boss", true); err != nil {
		t.Fatalf("root SetDisabled: %v", err)
	}
	if d, _ := a.Decide(ctx, access.Request{Subject: "boss", Action: access.AdminTenant, Scope: grants.Tenant("A")}); d.Allowed || d.Reason != "disabled" {
		t.Fatalf("disabled boss should be denied with reason disabled; got %+v", d)
	}
}
