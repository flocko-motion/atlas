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

