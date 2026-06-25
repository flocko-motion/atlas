# access-policy Specification

## Purpose
TBD - created by archiving change add-access-policy. Update Purpose after archive.
## Requirements
### Requirement: Identity is delegated to schemaf
The system SHALL identify the caller solely by the schemaf-authenticated subject
(`api.Subject(ctx)`) and SHALL NOT implement authentication, token issuance, or a
user store of its own. A request without a valid token is rejected by schemaf.

#### Scenario: Unauthenticated request
- **WHEN** a request arrives without a valid JWT
- **THEN** schemaf rejects it with 401 and no authorization check runs

#### Scenario: Authenticated subject is the principal
- **WHEN** a request carries a valid JWT with subject `S`
- **THEN** the authorization layer treats `S` as the principal for all decisions

### Requirement: Default deny
The system SHALL deny any action for which the principal holds no applicable grant
and is not root. Authentication alone confers no access.

#### Scenario: Authenticated but ungranted
- **WHEN** subject `S` with no grants requests any archive operation
- **THEN** the system denies the request

### Requirement: Denial reveals the caller's own subject id
When denying an authenticated principal for lack of any grant, the system SHALL
return 403 with a body containing the principal's own subject id and a hint to
request access from an admin.

#### Scenario: Onboarding via denial
- **WHEN** subject `S` with no grants is denied
- **THEN** the 403 body includes `S` (its own id) so it can be handed to an admin out-of-band

### Requirement: Root is an env-seeded break-glass override
The system SHALL treat any subject listed in `RANKE_ROOT_SUBJECT` (comma-separated)
as root, permitting every action, and SHALL evaluate this BEFORE consulting the grant
store so it works even when the store is empty or unreachable.

#### Scenario: Root acts with an empty store
- **WHEN** a subject in `RANKE_ROOT_SUBJECT` requests any action and the grant store has no rows
- **THEN** the action is permitted

#### Scenario: Root id alone is not power
- **WHEN** a request names a root subject but presents no valid JWT for it
- **THEN** the request is rejected (the env value only names root; acting as root requires a valid token)

### Requirement: Access is conferred only by grants
Beyond root, the system SHALL grant access only via grant records of the form
`(subject, scope, role)`, where scope is a tenant or an RA, tenant roles are `user`
or `admin`, and RA roles are `read`, `write`, or `admin`. There are no privilege
tiers other than root.

#### Scenario: Grant permits the matching action
- **WHEN** subject `S` holds `(S, RA:X, write)` and requests a write on archive `X`
- **THEN** the action is permitted

#### Scenario: Absent grant denies
- **WHEN** subject `S` holds only `(S, RA:X, read)` and requests a write on `X`
- **THEN** the action is denied

### Requirement: RA read is universe, RA write is sequencer
The system SHALL map RA `read` to Universe operations (read, query, verify), `write`
to advancing the sequencer (minting / committing to branches), and `admin` to those
plus reconfiguring the RA and managing its grants.

#### Scenario: Reader may verify but not mint
- **WHEN** subject `S` holds `(S, RA:X, read)`
- **THEN** `S` may read/query/verify `X` but a mint (sequencer advance) is denied

### Requirement: Tenant admin manages users; tenant user does not
The system SHALL allow a tenant `admin` to create, modify, and revoke grants and
memberships within that tenant, and SHALL forbid a tenant `user` from modifying any
user's grants.

#### Scenario: Tenant user cannot grant
- **WHEN** subject `S` holds `(S, tenant:T, user)` and attempts to grant a role in `T`
- **THEN** the request is denied

#### Scenario: Tenant admin can grant
- **WHEN** subject `A` holds `(A, tenant:T, admin)` and grants a role in `T`
- **THEN** the grant is created

### Requirement: Multi-tenant membership with independent roles
The system SHALL allow a subject to hold grants in multiple tenants with independent
roles, and SHALL determine the operative tenant from the request (e.g. the path), not
from session state.

#### Scenario: Admin in one tenant, user in another
- **WHEN** subject `S` holds `(S, tenant:A, admin)` and `(S, tenant:B, user)`
- **THEN** `S` may manage users under tenant `A` but may not manage users under tenant `B`

### Requirement: Tenant-scoped visibility
The system SHALL filter all read/list results to the querier's administered scope: a
tenant admin sees only subjects and grants within its tenant(s); a subject sees only
its own grants across tenants; root sees all. A tenant admin SHALL NOT learn a
subject's grants or memberships in other tenants.

#### Scenario: Cross-tenant affiliation is hidden
- **WHEN** subject `S` is a member of both tenant `A` and tenant `B`, and the admin of `A` views `S`
- **THEN** only `S`'s grants in `A` are returned, never its grants in or membership of `B`

### Requirement: Lazy provisioning, no global directory
The system SHALL NOT maintain a directory of all authenticated subjects. An access
record SHALL be created lazily on first grant, and a grant MAY be issued for a subject
that has not yet authenticated.

#### Scenario: Grant before first login
- **WHEN** an admin grants a role to subject id `S` that has never authenticated
- **THEN** the record is created, and when `S` later authenticates it already holds that access

### Requirement: Hide resources outside the visible scope
The system SHALL return 404 (not 403) when a principal requests a tenant or RA it has
no grant to see, so that existence is not disclosed.

#### Scenario: Probing a foreign tenant
- **WHEN** subject `S` with no grant in tenant `B` requests an archive under `B`
- **THEN** the system returns 404

### Requirement: Immediate revocation and subject disablement
The system SHALL deny access as soon as a grant is removed, and SHALL deny a subject
marked disabled regardless of a still-valid token, by resolving the principal's
current state on every request.

#### Scenario: Revoked grant
- **WHEN** a grant `(S, RA:X, write)` is removed and `S` next requests a write on `X`
- **THEN** the request is denied

#### Scenario: Disabled subject with valid token
- **WHEN** subject `S` is marked disabled and presents a still-valid JWT
- **THEN** the request is denied

### Requirement: Access policy does not affect verification
The system's access policy SHALL be minting-side server policy only and SHALL NOT
affect the verifiability of an archive. Verifying an archive's claims and provenance
SHALL require no server-side authorization.

#### Scenario: Independent verification
- **WHEN** a third party obtains an archive's claims and content
- **THEN** they can verify signatures, content integrity, and the monotonicity rule with no grant from this server

