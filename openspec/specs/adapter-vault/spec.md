# adapter-vault Specification

## Purpose
The vault port is a secret key-value store from which the configuration resolves
`vault(KEY)` references at launch. It is built from the configuration's env-only
vault section before other `vault(...)` fields resolve, and it is consulted only at
`run`, never during offline validation. This capability defines the port contract.

## Requirements

### Requirement: The vault resolves secrets by key
The system SHALL provide a vault port that resolves secret values by key for
`vault(KEY)` references appearing in the configuration.

#### Scenario: A vault reference resolves to its secret
- **WHEN** a configuration field is `vault(signing_key)` and the vault holds `signing_key`
- **THEN** the field resolves to that secret's value

### Requirement: The vault is built from the env-only vault section
The system SHALL build the vault at `run` from the configuration's vault section
using environment-resolved values only, and SHALL do so before resolving any
`vault(...)` reference elsewhere in the configuration (see `core-configuration`).

#### Scenario: The vault is built before vault fields resolve
- **WHEN** an instance launches
- **THEN** the vault is constructed from its env-only section first, then `vault(...)` fields are resolved against it

### Requirement: Vault access is launch-time only
The system SHALL consult the vault only at `run`, and offline `verify` SHALL NOT
touch the vault.

#### Scenario: Verify runs without the vault
- **WHEN** `verify` validates a configuration
- **THEN** it completes without constructing or querying the vault
