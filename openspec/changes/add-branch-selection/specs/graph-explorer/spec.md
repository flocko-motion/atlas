## ADDED Requirements

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
offer `$universe`. A scope is browsable only if it has a head to compute a closure from;
`$universe` has none, which is why RankeQL requires an explicit head to read there at
all. It remains reachable as a by-id read, not as a scope a view can be confined to.

#### Scenario: The archive is selectable alongside the branches
- **WHEN** a user opens the scope picker
- **THEN** it offers the whole archive and each named branch, every option carrying a head

#### Scenario: The Universe is not a browsable scope
- **WHEN** the scope picker is inspected
- **THEN** `$universe` is absent, having no head and so no closure to draw

### Requirement: A view may be confined to one scope

The explorer SHALL let a view admit only the claims within one selected scope's closure,
or everything loaded when nothing is selected. Selecting a scope SHALL be applied as a
predicate over the single union graph — swapping what the view admits — and SHALL NOT
clear the store, refetch, or add or remove any node or edge.

#### Scenario: Selecting a branch narrows the view
- **WHEN** a user selects a branch in a view holding claims from several
- **THEN** the view draws only that branch's closure, the store keeping every claim it held

#### Scenario: Switching branches does not reload
- **WHEN** a user switches from one loaded branch to another and back
- **THEN** each switch is immediate, nothing being refetched and no node added or removed

#### Scenario: Two branches can be compared as two views
- **WHEN** a user opens a second view and selects a different branch in it
- **THEN** each view draws its own branch, both reading the one store

### Requirement: Membership is reachability from the scope's head

The explorer SHALL determine whether a claim belongs to a scope by reachability from
that scope's head — a branch's own head, or the archive head for `$archive`. It SHALL
NOT record a single owning scope on a claim, since a claim reached from several branches
belongs to all of them and is one node in the store.

Because claims are immutable, a closure computed for a head SHALL remain valid for that
head, so it MAY be cached and reused; a branch that advances yields a new head and a new
closure, leaving any view still pinned to the previous head correct.

#### Scenario: A shared claim appears in both branches
- **WHEN** a claim is reachable from two branch heads
- **THEN** selecting either branch draws it, the store holding one node for it

#### Scenario: A closure is computed once per head
- **WHEN** a branch is selected, deselected and selected again with its head unchanged
- **THEN** the membership computed the first time is reused

### Requirement: A branch that is listed but not loaded reports itself

The explorer SHALL distinguish a branch whose closure is absent from the store from a
branch whose closure is empty, and SHALL report the former rather than drawing an empty
view. The branch listing describes the archive; the store holds what has been loaded, and
the two may differ.

#### Scenario: An unloaded branch is reported, not blanked
- **WHEN** a user selects a listed branch whose head is not in the store
- **THEN** the explorer reports that the branch is not loaded, rather than drawing nothing

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
