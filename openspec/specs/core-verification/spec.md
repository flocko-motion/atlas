# core-verification Specification

## Purpose
Verification establishes that an archive's claims are intact and authentic. The core
uses the ADT reference verification (from `ranke-go`) both as a gate on entry (see
`core-contribution`) and as an on-demand pass that can run at any time to any depth
over any closure. This capability defines the on-demand pass, its depths, external
witnessing of the branch-table head, and verification in the presence of explained
gaps left by deletion.

## Requirements

### Requirement: Any closure is verifiable on demand to a chosen depth
The system SHALL verify any closure from a head id and a storage Universe alone, at
any time, to a chosen depth: completeness (a `has` sweep that every referenced claim
is present), record-correctness (recanonicalise and recheck the id-chain and
signatures), and full-content (re-hash the content bytes).

#### Scenario: A full pass recomputes ids and content hashes
- **WHEN** a full-content verification runs over a closure
- **THEN** every claim's id-chain, signature, and content hash is recomputed and checked

#### Scenario: A completeness sweep checks presence only
- **WHEN** a completeness verification runs over a closure
- **THEN** each referenced claim is checked for presence without re-hashing its content

### Requirement: The retrieved closure is identical across Universes
The system SHALL rely on the collision resistance of the hash so that the closure
retrieved for a given head id is identical whichever Universe serves it.

#### Scenario: The same head id yields the same closure
- **WHEN** the same head id is resolved against two different Universes holding its claims
- **THEN** the retrieved closures are identical

### Requirement: The branch-table head can be witnessed externally
The system SHALL allow the current BTH to be witnessed by an external anchor — a
notary, a trusted counterparty, an RFC 3161 time-stamp authority, a transparency
log, or a public ledger. Because the BTH commits to the entire closure, a single
anchor SHALL fix the whole archive in time, and two anchors SHALL bracket everything
added between them regardless of any self-asserted `created_at`.

#### Scenario: One anchor fixes the whole archive
- **WHEN** the BTH is anchored externally at time `t`
- **THEN** the entire closure it commits to is witnessed as existing at `t`

#### Scenario: Two anchors bracket the interval
- **WHEN** the BTH is anchored at `t1` and again at `t2`
- **THEN** every claim added between them is bounded to `[t1, t2]` regardless of its self-reported timestamp

### Requirement: Verification accepts explained gaps for purged claims
The system SHALL pass verification for a graph whose claims were purged by extending
the verification algorithm with a callback that accepts an explained gap — a
`contribution/delete` reference or a copied `delete_by` date — in place of the
missing bytes (see `core-limiting-claims`).

#### Scenario: A purged claim with an explained gap still verifies
- **WHEN** a closure contains a claim whose bytes were purged but whose gap is explained
- **THEN** verification passes, treating the explained gap as valid in place of the bytes
