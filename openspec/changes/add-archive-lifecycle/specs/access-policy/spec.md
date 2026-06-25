## ADDED Requirements

### Requirement: Lifecycle control is an orthogonal capability
The system SHALL provide an `operator` role and an `ra.control` action that
authorize archive lifecycle transitions (start, stop, set-readonly, restart) and
nothing else. `operator` SHALL NOT confer data read, write, or admin. `admin`
SHALL confer `ra.control` in addition to its data authority. `operator` MAY be
granted at RA scope (one archive) or tenant scope (every archive in the tenant),
so a health watchdog can control archives it may not read.

#### Scenario: Watchdog controls but cannot read
- **WHEN** subject `W` holds only `(W, RA:X, operator)` and restarts archive `X`, then attempts to read `X`
- **THEN** the restart is permitted and the read is denied

#### Scenario: Tenant-wide watchdog controls every archive in the tenant
- **WHEN** subject `W` holds `(W, tenant:T, operator)` and stops an archive under `T`
- **THEN** the transition is permitted for any archive in `T`

#### Scenario: Admin also controls lifecycle
- **WHEN** subject `A` holds `(A, RA:X, admin)` and restarts `X`
- **THEN** the transition is permitted

### Requirement: Grants are additive
The system SHALL allow a subject to hold multiple roles on the same scope.
Granting a role SHALL add it without removing other roles the subject holds on
that scope; revoking SHALL remove only the named role. This lets an orthogonal
capability (e.g. `operator`) coexist with a data role (e.g. `write`) on one scope
without clobbering it.

#### Scenario: Granting an orthogonal role preserves the existing one
- **WHEN** subject `S` holds `(S, RA:X, write)` and is granted `(S, RA:X, operator)`
- **THEN** `S` may both mint `X` and control its lifecycle (write was not removed)

#### Scenario: Revoking one role leaves the others
- **WHEN** subject `S` holds both `write` and `operator` on `X` and `operator` is revoked
- **THEN** `S` retains `write` on `X`

## MODIFIED Requirements

### Requirement: Access is conferred only by grants
Beyond root, the system SHALL grant access only via grant records of the form
`(subject, scope, role)`, where scope is a tenant or an RA. Tenant roles are
`user` or `admin`. RA data roles form the ladder `read` ⊂ `write` ⊂ `admin`. The
`operator` role is ORTHOGONAL to that ladder: it confers lifecycle control only
(the `ra.control` action) and no data access, and MAY be held at RA or tenant
scope. There are no privilege tiers other than root.

#### Scenario: Grant permits the matching action
- **WHEN** subject `S` holds `(S, RA:X, write)` and requests a write on archive `X`
- **THEN** the action is permitted

#### Scenario: Absent grant denies
- **WHEN** subject `S` holds only `(S, RA:X, read)` and requests a write on `X`
- **THEN** the action is denied

### Requirement: RA read is universe, RA write is sequencer
The system SHALL map RA `read` to Universe operations (read, query, verify),
`write` to advancing the sequencer (minting / committing to branches), and
`admin` to those plus reconfiguring the RA and managing its grants. Lifecycle
control (the `ra.control` action: start/stop/set-readonly/restart) SHALL be
conferred by `operator` or `admin`, and SHALL NOT be implied by `read` or `write`.

#### Scenario: Reader may verify but not mint
- **WHEN** subject `S` holds `(S, RA:X, read)`
- **THEN** `S` may read/query/verify `X` but a mint (sequencer advance) is denied

#### Scenario: Writer cannot control lifecycle
- **WHEN** subject `S` holds `(S, RA:X, write)` and attempts to stop `X`
- **THEN** the request is denied (lifecycle control needs `operator` or `admin`)
