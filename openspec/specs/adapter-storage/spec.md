# adapter-storage Specification

## Purpose
The storage port is where a Ranke-Archive's bytes live. At its foundation is a
content-addressed blob store of three functions; over it sits the typed Universe
that reads and writes claims and content as records; and above that are the
composition primitives — stack, partition, and branch-routing — that combine
Universes for redundancy, performance, capability, or physical separation. This
capability defines the port contract and its composition, keeping any keyed-byte
store usable as the ground beneath an archive.

## Requirements

### Requirement: The blob store is content-addressed and immutable
A storage adapter SHALL provide `put(key, blob)`, `get(key)`, and `has(key)`, keyed
by a claim or content id. `put` SHALL be idempotent — a key's bytes never change —
`get` on an absent key SHALL fail the call, and `has` SHALL report presence. This
three-function contract is the minimal interface a backend must satisfy to serve as
a storage layer.

#### Scenario: Put is idempotent
- **WHEN** the same key is put twice
- **THEN** the stored bytes are unchanged and the second put is a no-op on content

#### Scenario: Get on a missing key fails
- **WHEN** `get` is called for a key the store does not hold
- **THEN** the call fails rather than returning empty bytes

### Requirement: The Universe is a typed interface over a blob store
The system SHALL provide a Universe that reads and writes claims and content as typed
records, decoding and encoding at the boundary, and SHALL derive a working Universe
from any blob store via the default construction (`NewBlobUniverse`). An adapter MAY
add bulk, streaming, or querying operations for performance without changing the
port contract.

#### Scenario: A blob-store-only backend yields a Universe
- **WHEN** an adapter implements only `get`/`put`/`has`
- **THEN** the default Universe construction produces a fully functional typed Universe over it

#### Scenario: Optional bulk operations do not change the contract
- **WHEN** an adapter adds native bulk or streaming operations
- **THEN** callers still interact through the same Universe interface and persistence agnosticism is preserved

### Requirement: Universes compose into a stack
The system SHALL provide a stacking primitive that combines ordered Universe layers,
each marked eager or lazy and optionally capped by a size threshold. A write SHALL
descend the stack, each eager layer storing it and each lazy layer passing it
through; a read SHALL descend likewise and, on a miss, backfill the lazy layers on
the way back up. The stack SHALL itself implement the Universe interface.

#### Scenario: A read miss backfills the cache
- **WHEN** a read misses a lazy upper layer but hits a lower layer
- **THEN** the value is returned and written back into the lazy layer it missed

#### Scenario: A size cap keeps large content out of a fast layer
- **WHEN** a layer is capped and content exceeds the threshold
- **THEN** that content is not stored in the capped layer

### Requirement: Universes compose into a partition
The system SHALL provide a partition primitive that routes each read and write to one
of `n` Universe instances, choosing the instance for a given `id` by `id mod n`, so
the `n` shards together hold the whole archive.

#### Scenario: An id routes to its shard
- **WHEN** a claim with a given `id` is read or written through a partition of `n` shards
- **THEN** the request is routed to shard `id mod n`

### Requirement: Universes compose by branch
The system SHALL provide a UniverseBranch primitive that routes each access by the
branch the claim is reached through, allowing physical separation of branches across
backends (useful for multi-tenancy), at the cost of redundant storage for claims
shared by multiple branches.

#### Scenario: A branch routes to its backend
- **WHEN** a claim is accessed through a given branch under a UniverseBranch composition
- **THEN** the request is routed to the backend configured for that branch

### Requirement: Replication is driven by a verification pass
The system SHALL replicate an archive by running a verification pass over a closure
against a replicating stack: reading and verifying each claim triggers the
composition's write-through into every configured layer. Arbitrary replication
strategies SHALL be expressible from the composition primitives.

#### Scenario: Verifying a closure fills every layer
- **WHEN** a verification pass runs over a closure against a fully-replicating stack
- **THEN** each claim in the closure is present on every eager layer afterwards

### Requirement: A layer may surface a query engine
The system SHALL let a layer offer a query engine that the read planner can target,
and the stack SHALL surface the most capable engine among its layers, falling back to
the native graph walk when no layer offers more (see `core-query`). A query's
meaning SHALL be unchanged by which layer answers it.

#### Scenario: The most capable engine is chosen
- **WHEN** a stack includes a layer offering a Cypher/GQL engine and layers offering only the native walk
- **THEN** the planner lowers the read to the Cypher/GQL engine

### Requirement: Direct backend access sits outside the port
The system's scoping and verification SHALL apply only to requests that pass through
RankeDB. A client holding a backing store's own credentials MAY read and write that
store directly with the platform's full power and none of RankeDB's guarantees; where
the guarantees must hold, the backend credentials are withheld and only RankeDB is
exposed.

#### Scenario: A holder of backend credentials bypasses scoping
- **WHEN** a client with a backing store's native credentials accesses it directly
- **THEN** RankeDB's scoping and verification do not apply, by design, to that direct access
