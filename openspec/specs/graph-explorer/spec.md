# graph-explorer Specification

## Purpose
TBD - created by archiving change add-branch-selection. Update Purpose after archive.
## Requirements
### Requirement: The explorer discovers an archive's branches

The explorer SHALL obtain the branches an archive holds through its data-source port,
each by name and current head, and SHALL answer that from every backend the port
supports — a REST connection reading the server's branch listing, and generated mock data
reporting the branches it produced. A user SHALL therefore be able to see what branches
exist without a server and without knowing a branch name in advance.

#### Scenario: Branches are listed from a server
- **WHEN** the explorer is connected to a ranke-db instance
- **THEN** it lists the archive's branches, each with its current head

#### Scenario: Branches are listed with no server
- **WHEN** the explorer runs against generated mock data
- **THEN** it lists the branches that data holds, through the same code path as a server connection

### Requirement: The selectable scopes are those that have a head

The explorer SHALL offer as a view scope the whole archive (`$archive`, the closure of
the archive head) and each named branch (the closure of its own head), and SHALL NOT
offer `$universe`. A scope is browsable only if it has a head its closure is rooted at;
`$universe` has none, which is why RankeQL requires an explicit head to read there at
all. It remains reachable as a by-id read, not as a scope a view can be confined to.

#### Scenario: The archive is selectable alongside the branches
- **WHEN** a user opens the scope picker
- **THEN** it offers the whole archive and each named branch, every option carrying a head

#### Scenario: The Universe is not a browsable scope
- **WHEN** the scope picker is inspected
- **THEN** `$universe` is absent, having no head and so no closure to name

### Requirement: A view may be confined to one scope

The explorer SHALL let a view admit only the claims a selected scope contains, or
everything loaded when nothing is selected. Selecting a scope SHALL be applied as a
predicate over the single union graph — swapping what the view admits — and SHALL NOT
clear the store, re-read any claim, or add or remove any node or edge.

#### Scenario: Selecting a branch narrows the view
- **WHEN** a user selects a branch in a view holding claims from several
- **THEN** the view draws only the claims that branch contains, the store keeping every claim it held

#### Scenario: Switching branches re-reads no claim
- **WHEN** a user switches from one answered branch to another and back
- **THEN** no claim body is read again and no node is added or removed

#### Scenario: Two branches can be compared as two views
- **WHEN** a user opens a second view and selects a different branch in it
- **THEN** each view draws its own branch, both reading the one store

### Requirement: Membership is the engine's answer, not the client's

The explorer SHALL obtain a scope's membership by asking its data source a scoped query for
identities only — `select.branch` naming the scope and `output.detail: id` asking for the
ids in it — and SHALL NOT traverse the graph itself to decide what a scope contains.
Membership is reachability from that scope's head, and computing it is the query engine's
work: a client that walked closures would be a second query engine, in the layer furthest
from the data and least able to optimise it.

The explorer SHALL then draw the intersection of that answer with the claims it holds, so
switching scopes costs a query for identities and a lookup per claim rather than re-reading
any claim body.

It SHALL NOT record a single owning scope on a claim: a claim reached from several branches
belongs to all of them and is one node in the store, so membership is a set the answer
defines and never a label the claim carries.

A scope the explorer has not yet asked about SHALL admit everything, so a view is never
blank for want of an answer.

#### Scenario: The engine decides what a branch contains
- **WHEN** a user selects a branch
- **THEN** the explorer asks the source for the identities in that scope and confines the view to them, computing no closure of its own

#### Scenario: A shared claim appears under both branches
- **WHEN** a claim is named by the answers for two branches
- **THEN** selecting either draws it, the store holding one node for it

#### Scenario: An answer already held is not asked for again
- **WHEN** a branch is selected, deselected and selected again
- **THEN** the identities already returned for it are reused

#### Scenario: An unanswered scope hides nothing
- **WHEN** a view names a scope whose identities have not been returned
- **THEN** it admits every claim rather than drawing empty

### Requirement: Claims a scope names but the session lacks are reported

The explorer SHALL report how many claims a scope's answer names that the session has not
read, rather than drawing the overlap silently. A graph is loaded and scopes are asked about
afterwards, so an answer may name claims never read — the archive having advanced, or the
load having been capped — and the difference between "this scope is small" and "this session
is behind the archive" is the question a user is asking.

#### Scenario: A shortfall is counted, not hidden
- **WHEN** a scope's answer names claims absent from the loaded graph
- **THEN** the explorer reports how many are missing alongside the scope

### Requirement: A branch name is discovered, never assumed

The explorer SHALL obtain every branch name it uses from the archive's branch listing,
and SHALL carry no built-in default. A branch name is a fact about the archive in front
of it, so assuming one — `main` or any other — is a guess about someone else's data that
is wrong whenever it is wrong.

Discovery SHALL therefore precede any branch-scoped read. Where the listing returns
exactly one branch the explorer MAY select it, there being no choice to make; where it
returns several it SHALL require a selection rather than pick one.

#### Scenario: No branch name exists before the listing is read
- **WHEN** the explorer starts against an archive it has not listed
- **THEN** it holds no branch name, and issues no branch-scoped read until the listing answers

#### Scenario: A sole branch needs no choosing
- **WHEN** the listing returns exactly one branch
- **THEN** the explorer may select it, no alternative existing

#### Scenario: Several branches require a choice
- **WHEN** the listing returns several branches
- **THEN** the explorer asks which, rather than selecting one by name, position or convention

#### Scenario: A read follows the selection
- **WHEN** a user selects a branch and loads
- **THEN** the query names that branch as its scope

### Requirement: The graph model comes from the ADT library

The explorer SHALL take its claim and edge types, its type vocabulary, its content
handling and its wire decoding from the published TypeScript ADT library
(`@flocko-motion/ranke`), and SHALL declare no type of its own that restates any part of
the ADT. The library mirrors `ranke-go`, which owns the model; a second reading of the
model inside a client is a second implementation of it, and the two disagree as soon as
either moves.

Decoding a result sequence SHALL use the library's reader rather than a local parse, and
SHALL stay incremental: the explorer reports a claim count as bytes arrive, so a reader
that required the whole body first would trade that for nothing.

The explorer MAY hold fields the ADT does not define where drawing needs them — the
contribution index a history layout sorts on, the branch a generated archive stamps, and a
display label derived from content. Those describe a drawing rather than a claim, and SHALL
sit beside the library's type rather than inside a copy of it.

#### Scenario: A claim's shape is the library's
- **WHEN** the explorer reads a claim from any source
- **THEN** the type it holds is the library's, and no local declaration restates a claim's fields, its edges, its ids or its type vocabulary

#### Scenario: Sequence framing is the library's
- **WHEN** a query answers with a framed sequence of results
- **THEN** the explorer decodes it with the library's reader, holding no framing rules of its own

#### Scenario: A count climbs while the body arrives
- **WHEN** a read returns tens of thousands of claims
- **THEN** the explorer reports how many have been decoded before the body is complete

#### Scenario: Drawing state stays outside the claim type
- **WHEN** the explorer needs a value the ADT does not define, such as a display label
- **THEN** it holds that value alongside the library's claim rather than in a local claim type carrying both

### Requirement: The REST contract comes from the generated client

The explorer SHALL issue every request through the client generated from
`openapi/openapi.yaml`, and SHALL NOT hand-write a route path, a request body type or a
response type the contract defines. The contract is generated on both sides of the wire,
so a client that transcribes it forfeits the one guarantee generation offers: that a
change to the spec breaks the build rather than the running page.

The RankeQL `Query` SHALL come from the ADT library instead, whose copy is generated from
`rql.schema.json` — the schema the specification releases and `openapi.yaml` implements
locally. The query the explorer sends is then typed by the standard rather than by one
binding of it.

Credentials SHALL remain the explorer's. The authentication kinds an instance accepts
follow from how it was configured, so the explorer SHALL build the request headers and the
generated client SHALL carry them.

#### Scenario: A route is never spelled out by hand
- **WHEN** the explorer reads a branch listing, a head, a resource's info, a claim, or its content
- **THEN** the call goes through the generated client, and no path string appears in explorer code

#### Scenario: The query type comes from the standard
- **WHEN** the explorer builds a query body
- **THEN** its type is the one generated from the released RankeQL schema, not a second copy declared locally

#### Scenario: A contract change fails the build
- **WHEN** a field the explorer sends or reads is renamed in the contract and the client is regenerated
- **THEN** the type check fails, rather than the change surfacing as a runtime error against a live instance

#### Scenario: The explorer decides the credential
- **WHEN** a connection authenticates with an API key, a bearer token or a macaroon
- **THEN** the explorer builds the header and hands it to the generated client

### Requirement: A gap in a dependency is closed in the dependency

The explorer SHALL report a gap in a library it depends on to that library, and SHALL NOT
answer the gap with a local copy of what the library owns. Every hand-written type this
capability removes began as a reasonable stopgap for something upstream had not published
yet, and each one outlived its reason by a long way.

Where a stopgap cannot be avoided, it SHALL name the owner it stands in for, so it is
found again once the gap closes.

#### Scenario: A missing capability is raised upstream
- **WHEN** the explorer needs behaviour the ADT library does not yet publish
- **THEN** the need is raised against that library, and no local type restating the ADT is added in its place

#### Scenario: An unavoidable stopgap names its owner
- **WHEN** work must proceed before a dependency can supply what it owns
- **THEN** the local stand-in records which dependency owns it, so the gap is closed rather than forgotten

