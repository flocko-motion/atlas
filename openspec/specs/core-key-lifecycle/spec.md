# core-key-lifecycle Specification

## Purpose
The ADT gives each contributor a `pubkey` and the vocabulary for expiry, but leaves
the key-lifetime mechanism to the implementation. RankeDB models that mechanism:
validity windows, forward rotation into a successor key, and early revocation. This
capability defines planned expiry, rotation, and early expiry as a limiting claim.

## Requirements

### Requirement: Contributor keys carry validity windows
The system SHALL support optional `pubkey_expires` and `pubkey_valid_from` (RFC 3339)
fields on a contributor claim, and SHALL fail verification for any claim whose
`created_at` is greater than or equal to the key's expiry or earlier than its
valid-from.

#### Scenario: A claim after expiry fails verification
- **WHEN** a claim's `created_at` is at or after its contributor key's `pubkey_expires`
- **THEN** verification of that claim fails

### Requirement: Rotation adds an overlapping successor key
The system SHALL rotate a key by adding another contributor with a
`pubkey_valid_from`, and SHALL allow keys to exist independently and overlap in their
validity windows.

#### Scenario: Two keys overlap during rotation
- **WHEN** a successor contributor with `pubkey_valid_from` is added before the predecessor's `pubkey_expires`
- **THEN** both keys are valid during the overlap and claims verify against whichever signed them

### Requirement: Early expiry is a propagating limiting claim
The system SHALL support early expiry by pointing a `contribution/expiry` edge at the
target contributor, carrying a `pubkey_expires` earlier than the planned expiry. The
claim carrying that edge SHALL be either a `contribution/expiry` claim or a new
`contribution/contributor` claim introducing the successor key, and it is a limiting
claim propagated across branches (see `core-limiting-claims`).

#### Scenario: An early-expiry claim ends a key's validity
- **WHEN** a `contribution/expiry` edge carries a `pubkey_expires` earlier than planned
- **THEN** claims by that key dated at or after the new expiry fail verification, and the limit propagates across branches

### Requirement: Keys are cryptographic entities only
The system SHALL treat contributor keys purely as cryptographic entities and SHALL
leave their mapping to application or real-world users to the application layer.

#### Scenario: No user identity is attached in the server
- **WHEN** a contributor key is used to check signatures
- **THEN** the server reasons only about the key, attaching no application user identity
