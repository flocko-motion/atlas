## ADDED Requirements

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

## MODIFIED Requirements

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

## REMOVED Requirements

### Requirement: A read is a select/where/limit query tree
**Reason**: restates §RankeQL. Its one server-side obligation — a generator selecting
only within the account's access scope — is kept above as its own requirement.
**Migration**: none; the language is unchanged. Read it from §RankeQL.

### Requirement: select roots a traversal and follows a path
**Reason**: restates §RankeQL and had drifted from it — it fixed a single `depth` where
a step takes a `min`/`max` hop range, made the scope optional where `branch` is
mandatory, and knew neither `$archive` nor the closure-pinning `head`.
**Migration**: none; read `Select` and `PathStep` from §RankeQL.

### Requirement: where is a boolean tree of comparisons
**Reason**: restates §RankeQL, whose `Where` fixes the combinators, the `{field, test}`
leaf and the comparison operators.
**Migration**: none; read `Where` and `Comparison` from §RankeQL.

### Requirement: limit bounds the read
**Reason**: restates §RankeQL and had drifted from it — it placed the content cap and
its overflow handling under `limit`, where they belong to `output.content`, and did not
carry `0` meaning unbounded.
**Migration**: none; read `Limit` and `Output` from §RankeQL.

### Requirement: format sets how much each result carries
**Reason**: restates §RankeQL and had drifted from it — it named a `select.format` that
does not exist and folded two orthogonal axes (`output.shape` and `output.detail`) into
one, leaving `output.form` and `detail: graph` unexpressible.
**Migration**: none; read `Output` from §RankeQL.

### Requirement: Results are totally ordered and pageable
**Reason**: restates §RankeQL and had drifted from it — `order` is a list of keys
applied in priority order, each with a `compare` collation, not a single `{field, dir}`.
The total order and the paging guarantee it rests on are fixed there.
**Migration**: none; read `Order` and §Results and streaming from §RankeQL.
