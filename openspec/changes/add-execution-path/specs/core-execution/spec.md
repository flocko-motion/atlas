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

### Requirement: The server serves the library's result set rather than shaping it

The system SHALL take the result set the library produces — already shaped and
serialized to the query's output axes — and SHALL write it to the wire, adding only the
sequence separators the media type requires and the content type that describes them.
It SHALL select the payload by the result's declared kind, and SHALL NOT re-serialise,
re-shape or re-encode it, so the bytes a client receives are the bytes the library
produced.

Where the library does not produce what a query asked for, the fix SHALL be made in the
library, not compensated for here.

#### Scenario: Bytes pass through unaltered
- **WHEN** a query result is written to the response
- **THEN** the library's bytes are copied through, framed but not re-encoded

#### Scenario: The canonical combination stays verifiable
- **WHEN** a claim is read with `detail: claims`, `form: original`, `encoding: cbor`
- **THEN** the bytes on the wire verify against the claim's id, the server having altered nothing

#### Scenario: Serving streams rather than buffers
- **WHEN** a query matches more results than fit comfortably in memory
- **THEN** results are pulled and written incrementally, without buffering the whole set

### Requirement: The execution report is served as the stream's final record

The system SHALL write the execution report the library returns as a final record
after the last result, when the query asked for one, and SHALL leave it typed as the
library typed it so a reader never mistakes it for a result. The system SHALL emit no
report when the query asked for none.

#### Scenario: A requested report closes the stream
- **WHEN** a query sets an execution report verbosity
- **THEN** the response carries the results, then the library's report record

#### Scenario: No report is emitted unless asked for
- **WHEN** a query sets no execution report verbosity
- **THEN** the response carries results only

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
taken from the signer, and SHALL NOT depend on a readable archive or an assembled
sequencer to do so, since health is wanted precisely when the stack is degraded.

#### Scenario: Health answers on a broken stack
- **WHEN** neither the archive nor the sequencer could be assembled
- **THEN** the health request still returns, reporting the signing identity from the signer

### Requirement: A stack without a sequencer does not serve

The system SHALL require a sequencer to serve. The archive every read opens is
`RA_k = (𝒰, k)`, and the sequencer is what holds `k` and hands back the snapshot, so a
stack configured without one has nothing to answer from. Such a configuration SHALL fail
at launch, naming what is missing, rather than starting and failing at the first request.

Serving an archive nothing may write to SHALL therefore be a sequencer backend that
refuses to advance the head, behind the same port. The core SHALL NOT reach an archive by
any route that bypasses the sequencer.

#### Scenario: A configuration with no sequencer fails at launch
- **WHEN** a configuration omits the sequencer section
- **THEN** launch fails with a message naming the sequencer as missing, rather than starting and answering nothing

#### Scenario: Health still answers when the archive does not
- **WHEN** a stack is degraded such that no archive can be opened
- **THEN** a health request still returns, reporting the signing identity from the signer

#### Scenario: Read-only is a backend, not a bypass
- **WHEN** an archive is to be served that nothing may write to
- **THEN** it is served through a sequencer backend that refuses to advance the head, not by opening the archive around the port
