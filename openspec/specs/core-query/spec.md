# core-query Specification

## Purpose
A read against RankeDB is a tree-structured query, expressible as a JSON object:
`select` generators that produce result sets of claims, `where` filters that narrow
them, and `limit` controls that bound the read. This capability defines the query
language, its result formats and ordering, and the guarantee that lowering a query
to whichever engine a storage layer offers preserves its result set.

## Requirements

### Requirement: A read is a select/where/limit query tree
The system SHALL express a read as a JSON query with `select` generators (defining
result sets of claims), `where` filters (selecting subsets), and `limit` controls,
with `and`, `or`, and `not` combining comparisons and `or` also unioning result
sets. Generators SHALL select only claims within the access scope of the requesting
account.

#### Scenario: A query generates then filters
- **WHEN** a query selects a branch closure and applies a `where` on `type`
- **THEN** the result is the generated set narrowed to claims matching the filter

### Requirement: select roots a traversal and follows a path
The system SHALL root a generator with `select.branch` (a branch's current head,
confining the query to that branch), optionally `select.claim` (a claim id within
the branch), or `select.claim` without a branch (a privileged Universe read, see
`core-access`); and SHALL traverse via `select.path`, a sequence of steps each naming
`edges`, a `dir` (`provenance` default, `uses`, or `connections`), a `depth`, and
optionally the `nodes` its endpoint may be — `edges` and `nodes` being type lists
where a leading `-` excludes a type. Without `path`, the generator SHALL follow every
edge outward to the full closure.

#### Scenario: A path step follows typed edges to a bounded depth
- **WHEN** a step names `edges: [derivation/*]`, `depth: 3`, `nodes: [source/*]`
- **THEN** the traversal follows derivation edges up to three hops and keeps endpoints of type `source/*`

#### Scenario: A branch-less claim root is privileged
- **WHEN** `select.claim` is given without a branch
- **THEN** the read is treated as a privileged Universe access (see `core-access`)

### Requirement: where is a boolean tree of comparisons
The system SHALL evaluate `where` as a boolean tree of comparisons, each testing one
field with `eq`, `ne`, `lt`, `le`, `gt`, `ge`, `in` (set membership), or `glob`
(shell-style wildcard), combined by `and`, `or`, and `not`.

#### Scenario: A glob comparison filters by type
- **WHEN** a `where` tests `type` with `glob: source/*`
- **THEN** only claims whose type matches `source/*` remain

### Requirement: limit bounds the read
The system SHALL bound a read with `limit` controls: `results` caps the number of
claims, `content` caps returned bytes per claim with `overflow` choosing whether an
over-large claim is cut off or omitted, and `time` cancels the query when exceeded.

#### Scenario: Content overflow is handled per setting
- **WHEN** `content` is `4kb` with `overflow: cutoff` and a claim exceeds it
- **THEN** that claim's returned content is cut off at the cap rather than omitted

### Requirement: format sets how much each result carries
The system SHALL set result carriage with `select.format`: `claim` (default) returns
the reached claim alone, and `path` returns the whole route to it.

#### Scenario: path format returns the route
- **WHEN** `format` is `path`
- **THEN** each result carries the full traversal route, not only the endpoint claim

### Requirement: Results are totally ordered and pageable
The system SHALL return results in the total order `(created_at, id)` unless a named
`order` of `{field, dir}` sorts by another field, with claims lacking that field
sorting last. To page, a client SHALL carry the last row's order key —
`(created_at, id)`, or `(field, id)` under a named order — into a `where` on the next
request.

#### Scenario: Paging resumes after the last row
- **WHEN** a client carries the previous page's last order key into the next request's `where`
- **THEN** results resume immediately after that row with no overlap or gap

### Requirement: Queries are declarative and lowering preserves the result set
The system SHALL keep queries declarative and backend-agnostic, lowering each to the
most capable execution engine the storage stack offers (a Cypher/GQL layer ranked
above the native graph walk), falling back to the native walk otherwise. The logical
order is generate, filter, sort, then limit; an optimised engine MAY diverge from
that order as long as the result set is identical (see `conformance-execution`).
`execution.layer` and `execution.trace` SHALL aid comparison across engines.

#### Scenario: Native and accelerated engines agree
- **WHEN** the same query is run against the native walk and against a Cypher/GQL layer
- **THEN** the two result sets are identical
