# adapter-sequencer Specification

## Purpose
The sequencer port maintains the branch-table head (BTH) — the mutable marker that
points at the archive's current branch table — under concurrent reads and writes, and
keeps its history so a failed storage write can be rolled back. This capability
defines the port contract; the merge pipeline that drives it (verification,
timestamping, atomic advance) is `core-contribution`.

## Requirements

### Requirement: The sequencer exposes bth, bthLen, and add
The system SHALL provide a sequencer port with `bth(n)` returning the id of the
latest branch table at `n = 0` or a historical one at `n < 0`, `bthLen()` returning
the length of the BTH history, and `add(id)` appending a new id to the top of the
history.

#### Scenario: The latest and a historical head are addressable
- **WHEN** `bth(0)` and `bth(-1)` are called
- **THEN** `bth(0)` returns the current branch table and `bth(-1)` the one before it

### Requirement: The BTH history is retained for rollback
The system SHALL keep the history of the BTH, not only its latest value, so that a
failed storage write — a claim that did not persist — can be recovered by rolling the
BTH back to the last working state.

#### Scenario: Rollback restores the last working head
- **WHEN** a claim referenced by a new branch table fails to persist
- **THEN** the BTH is rolled back to the last working head so the archive stays retrievable

### Requirement: The sequencer is a graph-registered contributor
The system SHALL register the sequencer in the graph as a contributor holding a
private key, so it can sign every branch-table claim it creates; the key and the
signing are supplied by the vault and signer ports (see `adapter-vault`,
`adapter-signer`, `core-contribution`).

#### Scenario: The sequencer signs with its configured identity
- **WHEN** the sequencer creates a branch-table claim
- **THEN** it signs the claim with its registered contributor key obtained via the signer and vault
