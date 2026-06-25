## Why

The server hosts many Ranke Archives for many tenants; users configure archives
and use them. That makes an archive a long-lived, **stateful** runtime entity, not
a build-once object: an admin stops one to reconfigure it, a fault takes one down,
a watchdog brings one back. The access policy answers *who may act on the data*,
but says nothing about *whether an archive is currently serving* — a separate,
operational concern. We need an explicit archive lifecycle, owned by the server
core, and a way to delegate control of it without handing over the data.

## What Changes

- Introduce the **`core`** logic layer (the server's domain core): it loads config,
  opens the vault, learns which archives exist, builds each via the (per-archive)
  **assembler**, supervises their lifecycle, holds the `access.Authz`, and answers
  `(tenant, ra) → handle` for the REST layer. ("assembler" is reserved for the
  narrow mechanism that builds ONE archive from its config.)
- Define an **archive lifecycle** with five states: `stopped`, `starting`,
  `running`, `running-readonly`, `failed`. `failed` (fault) is distinct from
  `stopped` (intent) and is retryable.
- **Two independent gates** per operation: the access decision (*may this subject?*,
  403 on denial) AND the archive's runtime state (*is the op offered now?*,
  503/409 on unavailability). Minting needs a `write` grant AND `state == running`.
- Add an **orthogonal lifecycle-control right** to the access policy: a new action
  `ra.control` and a new role **`operator`** that confers ONLY lifecycle control
  (start/stop/set-readonly/restart) and no data access. `admin` also confers it.
  `operator` is grantable at RA scope (one archive) or tenant scope (a health
  watchdog over all archives in a tenant).
- Make **grants additive**: a subject MAY hold several roles on one scope (e.g.
  `write` + `operator`); granting adds a role, revoking removes a named role.
  Without this, granting an orthogonal right would clobber an existing data role.

## Capabilities

### New Capabilities
- `archive-lifecycle`: the per-archive runtime states, their transitions, the
  operational gate (state-based availability, separate from authorization), and
  the server core that supervises them.

### Modified Capabilities
- `access-policy`: adds the `operator` role and `ra.control` action (orthogonal to
  the read⊂write⊂admin data ladder), and makes grants additive (a set of roles per
  scope rather than one).

## Impact

- **`go/core`** (new): the logic layer — config + vault → archive registry with
  lifecycle state, access enforcement, `(tenant, ra)` lookup.
- **`go/assembler`** (new): builds one `Archive` from its config + the vault.
- **`go/access`**: new `Action` `ControlRA` (`ra.control`), new `Role` `operator`;
  `raRoleAllows` grants `ra.control` to `operator`/`admin`; `Decide` honours a
  tenant- or RA-scope `operator` grant for `ra.control`.
- **`go/adapter/access/{mem,postgres}`**: grants become additive — store key
  `(subject, scope)` → `(subject, scope, role)`; `PutGrant` adds, `DeleteGrant`
  removes a named role; `Revoke` takes the role.
- **Not affected**: ranke-go and verification (lifecycle is server operations,
  authorization is server policy — neither touches the verifiable artifact). The
  data-role evaluation (read⊂write⊂admin) is unchanged.
