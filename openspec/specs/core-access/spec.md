# core-access Specification

## Purpose
Access control in RankeDB is server policy declared in the configuration: system
accounts, each holding grants of CRUDA rights over branches named by globs. This
capability defines the rights and their meaning, branch-glob scoping, the reserved
`$universe` privileged head-id read, tenancy as a configuration pattern, runtime
macaroon attenuation, and the invariant that access never affects verifiability.

## Requirements

### Requirement: Access is expressed as system accounts with branch-glob grants
The system SHALL express access as system accounts defined in the configuration,
each with a set of grants; a grant SHALL confer a set of rights to one or more
branches named by a glob. There SHALL be no runtime mutation of accounts or grants
(see `core-configuration`).

#### Scenario: A grant permits a matching action on a matching branch
- **WHEN** account `webapp` holds `CR foo_*` and reads or contributes to branch `foo_bar`
- **THEN** the action is permitted

#### Scenario: A branch outside the glob is denied
- **WHEN** account `webapp` holds only `CR foo_*` and requests branch `bar_baz`
- **THEN** the action is denied

### Requirement: Rights are the CRUDA letters
The system SHALL encode rights as CRUDA: **C** contribute claims to the branch,
**R** read, **U** update by overlaying a claim with a newer version, **D** delete,
and **A** admin — creating new branches and hiding existing ones by minting a BTH
that omits them.

#### Scenario: Admin creates a branch, contributor writes it
- **WHEN** `provisioner A foo_*` creates branch `foo_bar` and `webapp CR foo_*` contributes to it
- **THEN** both actions are permitted by their respective grants

### Requirement: Deleting a shared claim requires D on every holding branch
The system SHALL require the **D** right on every branch that holds a claim before
its bytes are purged, because a physical purge removes bytes from the shared Universe
(see `core-limiting-claims`).

#### Scenario: Deletion denied without D on all holding branches
- **WHEN** a claim is held in branches `a` and `b` and the account holds `D` on `a` only
- **THEN** the purge is denied

### Requirement: Head-id reads are privileged via the reserved $universe branch
The system SHALL treat reading a graph by head id directly — bypassing the branch
table — as a privileged action conferred only through the reserved branch name
`$universe`, to which only **R** applies. Because `$` is illegal in ordinary branch
names, no ordinary glob SHALL confer this privilege by accident.

#### Scenario: $universe read reaches any head id
- **WHEN** an account holds `R $universe` and reads a graph by head id
- **THEN** the read is permitted for any head the Universe holds

#### Scenario: An ordinary glob never matches $universe
- **WHEN** an account holds `R *`
- **THEN** it does not gain `$universe` access, since `*` cannot match the reserved name

### Requirement: Tenancy is a configuration pattern, not a primitive
The system SHALL allow tenancy to be modelled as a closed block of grants and
branches: a tenant's grants reach only that tenant's branches, and those branches
are reachable only through the tenant's grants. Tenancy SHALL be a configuration
pattern rather than a first-class primitive.

#### Scenario: A tenant's grants reach no foreign branch
- **WHEN** a tenant's grant set is defined over its branch set
- **THEN** no grant in the set confers access to a branch outside it, and no outside grant reaches a branch in it

### Requirement: Wider grants may be attenuated at runtime via macaroons
The system SHALL support deriving tighter-scoped access tokens from wider grants at
runtime through macaroon attenuation, enforcing branch boundaries from the
application layer without provisioning additional system accounts.

#### Scenario: An attenuated token narrows to a branch subset
- **WHEN** a macaroon derived from a wider grant is attenuated to branch `foo_bar`
- **THEN** the bearer may act only within `foo_bar`, never the wider original scope

### Requirement: Access is minting-side policy only
The system SHALL apply access policy only to requests that pass through it, and SHALL
NOT let access policy affect the verifiability of an archive. Verifying an archive's
claims and provenance SHALL require no grant from this server.

#### Scenario: Verification needs no grant
- **WHEN** a third party obtains an archive's claims and content
- **THEN** they can verify signatures, content integrity, and monotonicity with no grant from this server
