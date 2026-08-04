# core-limiting-claims Specification

## Purpose
A limiting claim documents that another claim is restricted in use — its bytes
deleted, or a contributor's key expired early. RankeDB purges bytes while leaving an
explained gap, and — because a claim limited in one branch must be limited in all —
propagates every limiting claim to each branch that references its target. This
capability defines planned and requested deletion, the explained gap, and the
cross-branch propagation mechanism.

## Requirements

### Requirement: Deletion purges bytes but leaves an explained gap
The system SHALL purge a claim's bytes while leaving a documented gap in their place,
so the removal is explained rather than silent, and verification still passes via the
gap-accepting callback (see `core-verification`).

#### Scenario: A purged claim leaves a verifiable gap
- **WHEN** a claim's bytes are purged
- **THEN** an explained gap remains in their place and the closure still verifies

### Requirement: Planned deletion is an intrinsic delete_by date
The system SHALL support planned deletion via a `delete_by` date the claim carries in
its signed content from creation. Every edge referencing the claim SHALL copy that
date so the gap stays explained wherever the claim is reached. Such a claim SHALL
replicate only into layers configured to allow deletion, and a purge SHALL remove due
claims at an interval set in the configuration.

#### Scenario: A due claim is purged and its references still explain the gap
- **WHEN** a claim's `delete_by` date has passed and the purge interval runs
- **THEN** its bytes are removed and each referencing edge's copied `delete_by` still explains the gap

### Requirement: Requested deletion is an extrinsic contribution/delete claim
The system SHALL support requested deletion via a later claim whose
`contribution/delete` edge names the target by id, documenting the gap once the bytes
are purged. Such a claim is a limiting claim.

#### Scenario: A delete request documents and purges its target
- **WHEN** a `contribution/delete` claim names a target and the purge runs
- **THEN** the target's bytes are removed and the delete claim documents the gap

### Requirement: Early key expiry is a limiting claim
The system SHALL treat an early contributor-key expiry — a `contribution/expiry`
edge carrying an earlier `pubkey_expires` (see `core-key-lifecycle`) — as a limiting
claim, propagated across branches by the same mechanism as deletion.

#### Scenario: An early-expiry claim propagates like a deletion
- **WHEN** an early-expiry limiting claim targets a contributor referenced in several branches
- **THEN** it is propagated to each branch that references the target

### Requirement: Limiting claims propagate across branches via a system branch
The system SHALL reference every limiting claim from a reserved system branch.
Whenever a limiting claim is contributed to any branch, the sequencer SHALL reference
it there too. At startup the sequencer SHALL read that branch in full and build an
in-memory lookup of limiting claims by target, and for each contribution SHALL match
the delta of added claims against the lookup, adding the limiting claims for any
targeted claim to the same branch as part of the same contribution.

#### Scenario: Referencing a limited claim pulls its limiting claim along
- **WHEN** a contribution references a claim that is the target of a limiting claim into a new branch
- **THEN** the sequencer adds the limiting claim to that branch in the same contribution

### Requirement: A claim limited in one branch is limited in all
The system SHALL ensure a limitation is never lost by cross-referencing its target
into another branch, and the sequencer SHALL enforce contribution rules that keep
arbitrary claims from propagating into other branches.

#### Scenario: A deleted claim stays deleted everywhere
- **WHEN** a claim deleted in one branch is referenced from another branch
- **THEN** the deletion applies in that branch too
