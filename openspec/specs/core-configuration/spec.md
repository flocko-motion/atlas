# core-configuration Specification

## Purpose
Configuration is RankeDB's launch artifact and composition root: one JSON document
that names and parametrises the adapters an instance mounts, resolves the secrets
they need, and declares the static system accounts and grants. This capability
defines its structure, the `env()` / `vault()` reference resolution, optional
whole-file age encryption, and the split between offline validation and launch-time
resolution.

## Requirements

### Requirement: Configuration is a single JSON launch artifact
The system SHALL take one JSON configuration document as the launch artifact for one
instance, specifying the full set of adapters to mount to the core and their
parametrisation. One configuration SHALL describe exactly one instance — the
analogue of a single database within a database service.

#### Scenario: Launch binds exactly the configured adapters
- **WHEN** an instance launches with a configuration
- **THEN** it mounts exactly the adapters the configuration names, parametrised as it specifies

### Requirement: String fields may be literal, env, or vault references
The system SHALL accept any configuration string field as a literal value, an
`env(VAR)` reference, or a `vault(KEY)` reference. At `run` the system SHALL resolve
them in order: environment variables first, then build the vault from the env-only
vault section, then resolve `vault(...)` references against that vault.

#### Scenario: An env reference resolves from the environment
- **WHEN** a field is `env(DB_URL)` and `DB_URL` is set
- **THEN** the field resolves to the environment value at run

#### Scenario: A vault reference resolves from the built vault
- **WHEN** a field is `vault(signing_key)` and the vault section defines that key
- **THEN** the field resolves from the vault after it is built

### Requirement: The configuration may be age-encrypted as a whole
The system SHALL accept an optionally age-encrypted configuration file and SHALL
decrypt it in the core before parsing. Encrypted configuration lets secrets live in
the file for personal or single-user deployments, while `env`/`vault` references
support secret-less parametrised configuration for enterprise infrastructure.

#### Scenario: An encrypted config is decrypted before parse
- **WHEN** the configuration file is age-encrypted
- **THEN** the core decrypts it and then parses the plaintext JSON

### Requirement: Validation is offline and secret-free
The system SHALL provide a `verify` that parses and schema-checks a configuration
only, without touching the environment or the vault, and SHALL perform reference
resolution only at `run`. Offline validation SHALL succeed with no secrets present.

#### Scenario: Verify succeeds without secrets
- **WHEN** `verify` runs against a configuration in an environment with no variables or vault set
- **THEN** it parses and schema-checks successfully without resolving any `env`/`vault` reference

### Requirement: Accounts and grants are a static configuration section
The system SHALL declare system accounts and their grants in a static section of the
configuration, and SHALL NOT expose runtime mutation of grants (see `core-access`).
Access policy is fixed for the life of an instance and changed only by the
edit-run cycle.

#### Scenario: Grants are not mutable at runtime
- **WHEN** an instance is running
- **THEN** no API changes its accounts or grants; changing them requires editing the configuration and relaunching
