# adapter-auth Specification

## Purpose
The auth port resolves an incoming request to an authenticated subject. The resolved
subject is what the core matches against the system accounts' grants (see
`core-access`). Authentication establishes identity only; the access decision is
made elsewhere. This capability defines the port contract and the mechanisms it
supports.

## Requirements

### Requirement: The authenticator resolves a request to a subject
The system SHALL provide an auth port that maps an endpoint request to an
authenticated subject, or rejects it, and SHALL pass the resolved subject to the core
for the access decision (see `core-access`).

#### Scenario: A valid request yields a subject
- **WHEN** a request carries valid credentials
- **THEN** the authenticator resolves it to a subject the core can authorize

#### Scenario: An invalid request is rejected
- **WHEN** a request carries missing or invalid credentials
- **THEN** the authenticator rejects it and no access decision runs

### Requirement: Multiple authentication mechanisms are supported
The system SHALL support the authentication mechanisms NoAuth, JWT, API Key, and
Macaroon, each resolving its own token form to a subject.

#### Scenario: Each configured mechanism authenticates its token form
- **WHEN** an endpoint is configured with one of NoAuth, JWT, API Key, or Macaroon
- **THEN** requests presenting that mechanism's token are resolved to a subject

### Requirement: Authentication is distinct from authorization
The system SHALL confine the auth port to establishing identity and SHALL leave the
access decision to the grants in `core-access`. An authenticated subject with no
applicable grant SHALL be denied by access, not by authentication.

#### Scenario: Authenticated but ungranted is denied by access
- **WHEN** a subject authenticates successfully but holds no applicable grant
- **THEN** the request is denied by the access decision, not by the authenticator
