# adapter-signer Specification

## Purpose
The signer port provides the server's signing identity: a `crypto.Signer` used to
sign the branch-table claims the sequencer contributes. It binds a claim's hash to a
public key so the resulting id verifies, and it obtains its key material from the
vault. This capability defines the port contract.

## Requirements

### Requirement: The signer signs claim hashes for a public identity
The system SHALL provide a signer implementing a `crypto.Signer` that produces a
signature binding a hash to the corresponding public key using a self-describing
scheme (for example Ed25519), so that a claim's id verifies against that public key.

#### Scenario: A signature verifies against the identity's pubkey
- **WHEN** the signer signs the hash of a canonical claim
- **THEN** the resulting signature verifies against the identity's public key

### Requirement: The signing key is supplied via the vault
The system SHALL obtain the signer's private key material through the vault port
rather than from a literal in the parsed configuration, so secrets stay resolvable
through `env`/`vault` references (see `adapter-vault`, `core-configuration`).

#### Scenario: The signing key resolves from the configured vault
- **WHEN** the signer is initialised at run
- **THEN** its private key is resolved from the configured vault, not from a plaintext config literal

### Requirement: The identity is registered as a contributor
The system SHALL ensure the signer's identity corresponds to a `contribution/contributor`
claim registered in the graph, so signatures it produces verify as authentic
contributions (see `core-contribution`).

#### Scenario: The signing identity has a contributor claim
- **WHEN** the signer's identity is used to sign a branch-table claim
- **THEN** a matching contributor claim exists in the graph against which the signature verifies
