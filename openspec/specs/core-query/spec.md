# core-query Specification

## Purpose
A read against RankeDB is a RankeQL query — the declarative `Query` type the normative
spec fixes (§RankeQL). This capability does not define that language; it cites it, and
fixes what the server owes around it: that a generator reads only within the requesting
account's scope, and that lowering a query to whichever engine a storage layer offers
preserves its result set.

## Requirements

### Requirement: Queries are declarative and lowering preserves the result set
The system SHALL keep queries declarative and backend-agnostic, lowering each to the
most capable execution engine the storage stack offers (a Cypher/GQL layer ranked
above the native graph walk) and falling back to the native walk otherwise. An
optimised engine MAY reorder or lower the evaluation steps §RankeQL fixes, provided
the delivered result set is identical; the native reference engine is the oracle (see
`conformance-suite`). `execution.layer` and `execution.report` SHALL aid comparison
across engines.

#### Scenario: Native and accelerated engines agree
- **WHEN** the same query is run against the native walk and against a Cypher/GQL layer
- **THEN** the two result sets are identical

#### Scenario: A pinned layer and a report attribute a difference
- **WHEN** two engines disagree and the query is re-run with `execution.layer` pinned and `execution.report` set
- **THEN** the report names the layer and the lowered query, so the difference can be attributed

### Requirement: The query language is the normative RankeQL type, restated nowhere
A read SHALL be a RankeQL query — the `Query` type the normative spec fixes
(§RankeQL: `select`, `where`, `output`, `order`, `limit`, `execution`), which alone
defines its field names, permitted values, evaluation order and result shapes. This
capability SHALL cite that chapter rather than restate it, so the language cannot
drift from its source. `ranke-go`'s `Query` is the reference implementation, and a
transport binding (see `rest-api`) carries it onto the wire.

#### Scenario: The language is read from its source
- **WHEN** an implementer or a binding needs a query field's name, values or meaning
- **THEN** §RankeQL answers it, and no requirement here duplicates or overrides that answer

#### Scenario: A language change lands in one place
- **WHEN** RankeQL gains or changes a field
- **THEN** the normative chapter changes and this capability needs no edit, because it fixes none of the language

### Requirement: A generator reads only within the requesting account's scope
The system SHALL confine every generator to what the requesting account may read: the
query's scope (`select.branch`) is what the grant is held against, and a `select.head`
given explicitly SHALL only narrow that scope, never widen it past the grant (see
`core-access`). A claim outside the resolved closure SHALL be absent from the result
rather than reported as denied.

#### Scenario: The grant bounds the generator
- **WHEN** an account holding `R` on one branch runs a query scoped to another
- **THEN** the read is denied, because the scope is what the grant is held against

#### Scenario: An explicit head cannot widen the scope
- **WHEN** a query under a branch scope names a `select.head` outside that branch's closure
- **THEN** the query is rejected rather than served from the wider closure
