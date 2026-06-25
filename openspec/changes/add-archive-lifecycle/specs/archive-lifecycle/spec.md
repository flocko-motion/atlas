## ADDED Requirements

### Requirement: Archives have an explicit runtime lifecycle
The system SHALL model each configured archive as a stateful runtime entity in
exactly one of five states: `stopped` (configured but not serving — admin
intent), `starting` (transient, while the assembler builds it), `running`
(serving reads and writes), `running-readonly` (serving reads/query/verify but
refusing writes), and `failed` (assembly or runtime fault). The server core
SHALL own and report each archive's current state.

#### Scenario: Configured but stopped does not serve
- **WHEN** an archive is configured with state `stopped`
- **THEN** the core does not serve operations on it and reports it as `stopped`

#### Scenario: Running serves reads and writes
- **WHEN** an archive is `running`
- **THEN** authorized reads and writes (mints) are served

#### Scenario: Read-only serves reads but refuses writes
- **WHEN** an archive is `running-readonly`
- **THEN** authorized reads/query/verify are served and any write (mint) is refused

### Requirement: failed is distinct from stopped
The system SHALL distinguish a `failed` archive (a fault — bad config, missing
vault key, unreachable storage, or a runtime error) from a `stopped` archive (a
deliberate admin action). A `failed` archive SHALL be retryable (a transition
back to `starting`) and SHALL surface its fault in health reporting.

#### Scenario: Assembly fault yields failed, not stopped
- **WHEN** the assembler cannot build an archive (e.g. its vault key is absent)
- **THEN** the archive enters `failed` (not `stopped`) and the fault is reported

#### Scenario: Failed archive can be retried
- **WHEN** an operator retries a `failed` archive
- **THEN** it transitions to `starting` and, if assembly now succeeds, to `running`

### Requirement: Operations gate on lifecycle state independently of authorization
The system SHALL require an operation to pass BOTH the authorization decision and
the archive's runtime-state check, and SHALL treat a state-based refusal as an
operational error (e.g. 503/409), distinct from an authorization denial (403). A
subject's grant SHALL NOT override the archive's state.

#### Scenario: Authorized write on a read-only archive is refused operationally
- **WHEN** a subject with a `write` grant requests a mint on a `running-readonly` archive
- **THEN** the request is refused as an operational unavailability (not a 403)

#### Scenario: Operation on a stopped or failed archive is unavailable
- **WHEN** any operation targets a `stopped` or `failed` archive
- **THEN** the core reports the archive unavailable rather than serving the operation

### Requirement: Lifecycle transitions are explicit and controlled
The system SHALL expose lifecycle transitions (start, stop, set-readonly, restart)
as explicit operations gated by the `ra.control` action, so that controlling an
archive's lifecycle is authorized separately from reading or writing its data.

#### Scenario: Stop to reconfigure, then start
- **WHEN** an authorized controller stops an archive, its config is changed, and it is started again
- **THEN** the archive transitions `running → stopped → starting → running` and serves the new config

#### Scenario: Lifecycle control does not require data access
- **WHEN** a subject authorized for `ra.control` but with no read/write grant restarts an archive
- **THEN** the transition is permitted even though that subject may not read or mint the archive

### Requirement: Lifecycle desired-state is persisted and restored on boot
The system SHALL persist each archive's DESIRED lifecycle state — `running`,
`running-readonly`, or `stopped` — as durable operational state, updated by the
`ra.control` transitions. On startup the system SHALL reconcile each archive
toward its persisted desired state rather than defaulting every archive to
running. The transient runtime states `starting` and `failed` SHALL NOT be
persisted: `failed` is a runtime outcome, retried from the desired state on the
next boot. Only the TARGET (desired) state is persisted — a FIELD in the
archive's config (alongside its storage/sequencer/stacking definition), updated
by the `ra.control` transitions, not a separate store. The CURRENT (runtime)
state is held in memory and recomputed by reconciliation; it is never persisted.

#### Scenario: Stopped stays stopped across restarts
- **WHEN** an archive is stopped and the server later restarts
- **THEN** the archive remains stopped — it is not assembled or served — until explicitly started

#### Scenario: Desired-running is reconciled on boot
- **WHEN** an archive's desired state is `running` and the server restarts
- **THEN** the core assembles it and brings it to `running` (or `failed` if assembly errors)

#### Scenario: Failure is not persisted as intent
- **WHEN** an archive with desired state `running` fails to assemble (entering `failed`) and the server restarts
- **THEN** the core retries assembly — desired state is still `running` — rather than leaving it `failed`
