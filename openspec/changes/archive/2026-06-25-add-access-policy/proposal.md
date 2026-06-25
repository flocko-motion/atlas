## Why

RankeDB exposes archives over a REST API, but has no access policy: anything
authenticated could do anything. We need to control *who may do what* — without
reinventing authentication (schemaf already owns that) and without entangling the
verifiable data model (anyone can verify an archive regardless of server policy).

## What Changes

- Introduce a **tenant-scoped authorization layer** enforced at the API. schemaf
  provides authentication (a JWT over an opaque subject); this layer decides access.
- Account tiers collapse to **`root` / `user` / `anyone`**; everything else is a
  **grant `(subject, scope, role)`** — tenant `{user, admin}`, RA `{read, write, admin}`.
  No tiers beyond root; `tenant-admin` is a grant, not a class.
- **`root` is env-seeded** (`RANKE_ROOT_SUBJECT`, comma-list) and checked *before* the
  store — a break-glass override, changeable only out-of-band.
- **Default-deny**; access records are created **lazily on first grant** (no global
  user directory). A subject learns its own id from the **403** denial body and hands
  it to an admin out-of-band (no enumeration).
- **Tenant-scoped visibility**: a tenant-admin sees/grants only within its tenant(s);
  cross-tenant grants and memberships are mutually invisible; only the subject (self)
  and `root` see the full cross-tenant picture.
- Response rules: **401** (unauthenticated), **403 + own subject id** (authenticated, no
  grant — the onboarding path), **404** (resource in a tenant you may not see).
- Anchor: RA `read` = Universe read/query/verify; `write` = advance the sequencer
  (mint). Policy is **minting-side only** — it never affects verification.

## Capabilities

### New Capabilities
- `access-policy`: the authorization model (subjects, tenants, grants, roles), its
  enforcement at the API (`authz` resolver), scoped-read visibility, the env root
  seed, and the 401/403/404 onboarding rules.

### Modified Capabilities
<!-- none — no existing specs (this is the first capability) -->

## Impact

- **ranke-db API layer**: every handler builds an `access.Request{Subject, Action, Scope}`
  from `Subject(ctx)` + path + endpoint, and calls `Authz.Require` (the `access` engine
  resolves the `Decision`).
- **`go/access`**: the engine (`Authz`, `Request`/`Decision`/`Decide`/`Require`,
  `Grant`/`Revoke`) and the `Store` seam.
- **`go/adapter/access/{mem,postgres}`**: the `Store` backends. Postgres (migration
  prefix `rankeaccess`) holds `ranke_subjects(id, disabled, token_epoch)` +
  `ranke_grants(subject, scope_tenant, scope_ra, role)`.
- **Environment**: `RANKE_ROOT_SUBJECT` (break-glass root seed).
- **schemaf**: consumes `api.Subject(ctx)` only — no auth code added here.
- **Not affected**: ranke-go and verification — access is server policy, not part of the
  verifiable artifact.
