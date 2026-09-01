# architecture Specification

## Purpose
RankeDB is a hexagonal server that composes the `ranke-go` library — the reference
implementation of the Ranke-Graph ADT (foundation paper) — into a running service.
It launches exactly one Ranke-Archive from one configuration and serves it. This
capability defines the shape of the core: the six adapter ports, configuration as
the composition root, the content/policy razor that decides what belongs where, and
the ADT guarantees the server inherits rather than re-proves.

The two reference papers that define this architecture and the underlying data
structure are read under `docs/papers/`, where `make docs-papers` clones them from
ranke-graph — a fetched copy, gitignored, never edited here:

- **`docs/papers/02-ranke-db/ranke-db.typ`** — *RankeDB: Serving the Ranke-Graph*,
  the architecture this capability and its siblings specify.
- **`docs/papers/01-ranke-graph/ranke-graph.typ`** — *Ranke-Graph: A Provenance-First
  Data Structure*, the foundation paper defining the ADT (claims, Universe,
  branch-table head, verification, D1–D8) that RankeDB serves.

These papers are the source of truth; the specs track them and must not diverge.

## Requirements

### Requirement: The core wraps the ADT library and never reimplements it
The core SHALL contain the server's own logic (endpoints, access, persistence
composition, contribution management) and SHALL reach every external technology only
through an adapter port. It SHALL wrap `ranke-go` for the graph model and
verification and SHALL NOT reimplement claims, closures, or the verification
algorithm here.

#### Scenario: Graph logic stays in the library
- **WHEN** a behaviour concerns the claim model, closures, or verification
- **THEN** it is delegated to `ranke-go` and not reimplemented in the server core

#### Scenario: A new backend touches only an adapter
- **WHEN** a new storage or sequencer technology is supported
- **THEN** the change is confined to a single adapter behind its port, with the core unchanged

### Requirement: The hexagon has exactly six adapter ports
The system SHALL expose six adapter ports: four DRIVEN by the core — **storage**
(the Universe), **sequencer** (the branch-table head), **signer** (a signing
identity for signing branch-table claims), and **vault** (a secret store) — and two DRIVING
the core — **auth** (resolving a request to a subject) and **endpoint** (a
transport bound to an authenticator). Each port SHALL be a narrow contract so the
technology behind it stays replaceable.

#### Scenario: External access is confined to a port
- **WHEN** the core reads storage, advances the sequencer, signs, fetches a secret, authenticates a request, or accepts a request
- **THEN** it does so only through the corresponding adapter port

### Requirement: An adapter is verified against its real counterpart
The system SHALL verify each adapter against the real counterpart it exists to
talk to, not a mock — an adapter is only a translation of the port to its
counterpart's protocol, so a mock would test nothing. An adapter whose counterpart
is an external service SHALL, in its own test, spin up that real service (via
podman) and SHALL skip cleanly when it is unavailable, so the offline gate stays
green. An adapter whose counterpart is local (memory, filesystem) SHALL be tested
directly. The core SHALL be tested against the port with an in-memory double,
since the core is not an adapter.

#### Scenario: An external-counterpart adapter drives the real service
- **WHEN** the OpenBao vault adapter is tested
- **THEN** the test spins up a real OpenBao via podman and skips when podman is unavailable, never substituting a mock

#### Scenario: The core is tested against the port, not a backend
- **WHEN** the core's secret resolution is tested
- **THEN** it runs against an in-memory implementation of the vault port, independent of any real backend

### Requirement: A port's adapters share a config-driven conformance suite
The system SHALL define, per adapter port, one conformance test that runs every
backend of that port through the port interface — each backend built from a
config via the port's factory, not constructed directly — so all backends are
held to a single behavioural contract. Each backend SHALL contribute a setup
(returning its config and a teardown) and MAY contribute backend-specific tests;
a backend's run SHALL be setup → optional specific tests → the shared conformance
suite → teardown. Adding a backend SHALL be one registry entry, not a new test
file.

#### Scenario: Every backend runs the same suite through the port
- **WHEN** the signer conformance test runs
- **THEN** each backend (inmemory, OpenBao Transit) is built via `signer.New` from its config and asserted against the same shared suite

#### Scenario: A backend run follows setup, specific, standard, teardown
- **WHEN** a backend's conformance case runs
- **THEN** it sets up against its counterpart, runs any backend-specific tests, runs the shared suite, and tears the counterpart down

### Requirement: One process serves one configuration
The system SHALL serve exactly one Ranke-Archive, assembled from exactly one
configuration supplied at launch, and SHALL NOT support runtime reconfiguration.
Administering many instances (configuring, starting, stopping, monitoring) is a
layer above a single instance.

#### Scenario: The admin cycle is edit-run-observe-stop
- **WHEN** an operator needs to change which adapters are bound or how they are parametrised
- **THEN** they stop the instance, edit the configuration, and start it again — the running process is never reconfigured in place

### Requirement: Configuration is the composition root
The system SHALL determine which adapters are bound and how they are parametrised
solely from the configuration supplied at launch (see `core-configuration`). The
core SHALL assemble the stack from that configuration and from no other source.

#### Scenario: The bound stack reflects the configuration alone
- **WHEN** an instance launches with a given configuration
- **THEN** the adapters it mounts and their parameters are exactly those the configuration specifies

### Requirement: The content/policy razor decides what belongs where
The system SHALL preserve *content* — claims and their provenance, the record of
what happened — in the graph, and SHALL keep *policy and wiring* — which adapters
are bound, where secrets resolve, and who may act on which branches — in the
configuration. A fact the archive should preserve is content; an operational choice
of this server is policy. Access rights describe the server, not the world, and
SHALL live in the configuration, never in the graph.

#### Scenario: An access right is policy, not a claim
- **WHEN** the system records who may read or write a branch
- **THEN** that grant lives in the configuration and is never written as a claim in the graph

