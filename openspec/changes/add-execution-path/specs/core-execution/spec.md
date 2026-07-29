## ADDED Requirements

### Requirement: Execute dispatches on the operation and yields a Stream

The system SHALL resolve an authorized request to exactly one execution path selected
by its operation, and SHALL return either a `Stream` carrying the response body or an
error drawn from the pipeline's sentinels. Each path SHALL delegate the work to the
reference library and SHALL implement no graph behaviour of its own. The stage SHALL
NOT re-run authentication, and SHALL record its progress in the request's report as
the earlier stages do.

#### Scenario: Each operation has one path
- **WHEN** an authorized request for any supported operation reaches the execute stage
- **THEN** exactly one execution path runs for it and returns a Stream or an error

#### Scenario: An unbuilt operation is reported as unimplemented
- **WHEN** an operation's path is not yet built
- **THEN** the stage returns the not-implemented sentinel, which categorises as `unimplemented`

### Requirement: A request reads from one archive snapshot

The system SHALL resolve the immutable archive snapshot once per request and answer
every read in that request from it, so a request observes one consistent state even if
the head advances while it runs.

#### Scenario: A long read is not disturbed by a concurrent merge
- **WHEN** a merge advances the head while a query is streaming results
- **THEN** the query continues against the snapshot it opened and its results stay consistent

### Requirement: A response body is an item renderer plus a framing

The system SHALL render a response body from two orthogonal parts: an **item**, which
serialises one result under the request's output axes, and a **framing**, which writes
a sequence of items with its separators and declares the content type. A framing SHALL
be usable with any item and an item under any framing, so that no execution path
carries a serialiser of its own.

Both SHALL operate lazily, so that neither a large result set nor a large blob is held
whole in memory.

#### Scenario: One item type serves several framings
- **WHEN** the same result is returned under `application/json-seq` and under `application/cbor-seq`
- **THEN** the same item renderer produces both bodies, differing only in the framing and the encoding axis

#### Scenario: A single-value response reuses the machinery
- **WHEN** an operation returns exactly one value, such as a branch head or a health report
- **THEN** it is rendered as a single item under a single-object framing, not by a bespoke serialiser

#### Scenario: Results stream rather than buffer
- **WHEN** a query matches more results than fit comfortably in memory
- **THEN** items are pulled and written incrementally, without buffering the whole set

### Requirement: A stream may carry items of more than one kind

The system SHALL allow one stream to carry items of distinct kinds and SHALL keep each
kind distinguishable to a reader. In particular, when a query requests an execution
report, the stream SHALL carry that report as a final record after the last result,
typed so that a reader never mistakes it for a result.

#### Scenario: A requested report closes the stream
- **WHEN** a query sets an execution report verbosity
- **THEN** the stream carries the results, then one final report record naming the layer that served the query, what it was lowered to, and per-stage timings

#### Scenario: The report is not mistakable for data
- **WHEN** a reader consumes a stream carrying a report
- **THEN** the report record is distinguishable from the result records by its type, not by its position alone

#### Scenario: No report is emitted unless asked for
- **WHEN** a query sets no execution report verbosity
- **THEN** the stream carries results only

### Requirement: Canonical output passes through unaltered

The system SHALL return, for the output combination `detail: claims` + `form:
original` + `encoding: cbor`, the library's canonical serialization unaltered, so the
result stays re-hashable and signature-checkable against the claim's id. The renderer
SHALL NOT re-encode those bytes. Every other combination SHALL be treated as a
convenience projection and SHALL NOT be presented as verifiable.

#### Scenario: The canonical combination round-trips
- **WHEN** a claim is read with `detail: claims`, `form: original`, `encoding: cbor`
- **THEN** the returned bytes verify against the claim's id without reconstruction

#### Scenario: A convenience rendering is not claimed to verify
- **WHEN** a claim is read with `encoding: json`
- **THEN** the response is a projection and carries no guarantee of verifiability against the id

### Requirement: Library failures resolve to the pipeline's sentinels

The system SHALL map each failure returned by the library onto the sentinel its
category requires, so an endpoint translates in one place. An absent claim, branch or
content and one lying outside the requested scope's closure SHALL both resolve to the
not-found sentinel, the two being indistinguishable to a caller.

#### Scenario: Out-of-closure is indistinguishable from absent
- **WHEN** a claim exists in the Universe but not within the named branch's closure
- **THEN** the read returns not-found, identical to the response for an unknown id

#### Scenario: An unexpected failure is internal
- **WHEN** the library returns an error matching no known category
- **THEN** it resolves to the internal category rather than being reported as a client error

### Requirement: Verification runs are a server-managed registry

The system SHALL manage verification runs itself — allocating each an id, tracking its
status from running through complete, stopped or error, retaining its report for later
retrieval, and supporting listing, cancellation and deletion — the library's
verification handle being a live in-process value with no identity, persistence or
cancellation of its own. This is operational state rather than graph state.

The system SHALL bound the number of concurrently active runs and SHALL refuse a
request that would exceed it.

#### Scenario: A run is startable, pollable and retrievable
- **WHEN** a verification run is started
- **THEN** it is assigned an id, reports as running, and its report stays retrievable by that id after it completes

#### Scenario: A cancelled run keeps its partial findings
- **WHEN** an active run is cancelled
- **THEN** it reports as stopped and the findings gathered before cancellation are retained

#### Scenario: Exceeding the run limit is refused
- **WHEN** starting another run would exceed the configured limit of active runs
- **THEN** the request is refused as busy

### Requirement: Health is answerable without a working archive

The system SHALL answer a health request with liveness and the stack's signing identity
without depending on a readable archive or a running sequencer, so that health stays
reportable precisely when the stack is degraded.

#### Scenario: Health answers on a broken stack
- **WHEN** the archive cannot be opened
- **THEN** the health request still returns, reporting the signing identity
