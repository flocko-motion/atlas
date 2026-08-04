## MODIFIED Requirements

### Requirement: REST binds the paper's read-and-contribute surface
The REST endpoint SHALL expose the full surface as `POST /query` (a RankeQL query)
and `POST /contribute` (an atomic contribution of one or more signed claims), and
SHALL pin the cacheable GET subset:

- `GET /branches` — the branch table's branches, each by name and current head
- `GET /branches/{branch}/head` — that branch's current head id
- `GET /branches/{branch}/claims/{id}` — a claim within that branch's closure
- `GET /branches/{branch}/claims/{id}/content` — that claim's content
- `GET /archive/claims/{id}` — a claim within the archive head's closure, across all branches
- `GET /archive/claims/{id}/content` — that claim's content
- `GET /universe/claims/{id}` — a claim anywhere the Universe holds, reached without any closure
- `GET /universe/claims/{id}/content` — that claim's content

The by-id claim forms SHALL be HTTP-cacheable immutably, their ids content-addressing
the bytes; the branch listing and the branch head SHALL be cacheable with revalidation,
both carrying heads that move. Content SHALL be addressed by the claim holding it rather
than by a raw hash, so the read stays scoped as the claim's route is scoped.

`GET /branches` SHALL be reachable without prior knowledge of any branch name, so a
client discovers what it may address, and SHALL require the **R** right on the reserved
`$branches` target (see `core-access`).

#### Scenario: The full read surface is a query POST
- **WHEN** a client reads via `POST /query` with a RankeQL body
- **THEN** the response is the result set the query defines

#### Scenario: A by-id claim GET is cacheable without a JSON body
- **WHEN** a client issues `GET /branches/{branch}/claims/{id}`
- **THEN** the claim is returned without a query body and is cacheable immutably by id

#### Scenario: The branch head GET is revalidated
- **WHEN** a client issues `GET /branches/{branch}/head`
- **THEN** the current head id (a moving target) is returned as a cacheable-with-revalidation resource

#### Scenario: Content is fetched through the claim that holds it
- **WHEN** a client issues `GET /branches/{branch}/claims/{id}/content`
- **THEN** the claim's content is streamed without the client knowing whether the bytes were stored inline or as a separate blob

#### Scenario: A client with no branch name discovers what it may read
- **WHEN** a client that knows no branch name issues `GET /branches`
- **THEN** it receives every branch the table holds, each with its name and current head, and can address the branch routes from that list

#### Scenario: Listing branches needs R on the reserved target
- **WHEN** an account holding no `R $branches` grant issues `GET /branches`
- **THEN** the request is denied by the access decision

### Requirement: Every claim read names one of the three scopes
The REST contract SHALL address each claim read as its scope followed by the claim
within it, one shape serving all three scopes the access model defines:

- `/branches/{branch}/claims/{id}` — confined to that branch's closure, needing **R** on the branch
- `/archive/claims/{id}` — confined to the archive head's closure across every branch, needing **R** on `$archive`
- `/universe/claims/{id}` — reached without any closure, needing **R** on `$universe`

The scope SHALL be an explicit path segment rather than something a reader infers from
what is absent. No path segment SHALL carry a `$`.

The contract SHALL document each scope collection as the reserved scope it reads, naming
it: `/archive/…` reads `$archive`, the closure of the whole Ranke-Archive, and
`/universe/…` reads `$universe`. That is the same scope a RankeQL body names in
`select.branch` and the same target a grant is written against, so a reader sees one
concept across the route, the query language and the access policy rather than three
that happen to correspond.

Naming the correspondence SHALL NOT introduce a second path. Each scope SHALL have
exactly one route; `{branch}` SHALL mean an ordinary branch, so a reserved name supplied
there names a branch that does not exist and is answered as not-found.

#### Scenario: A reader can connect the route to the grant that gates it
- **WHEN** a client reads the contract for `GET /archive/claims/{id}`
- **THEN** it states that the route reads the `$archive` scope, which is what `R $archive` grants and what `select.branch: "$archive"` names

#### Scenario: A reserved name in the branch position is not a second route
- **WHEN** a client requests `/branches/$archive/claims/{id}`
- **THEN** the response is not-found, `{branch}` naming an ordinary branch and each scope having exactly one route

The three SHALL be genuinely distinct in what they reach. A claim the Universe holds but
the current head does not reach SHALL be returned by the Universe route and reported as
not-found by the archive route, since the privileged read exists precisely to reach an
archive by head id alone — restoring from a Universe and a head kept outside the server
(see `core-access`).

#### Scenario: The three scopes read alike in shape
- **WHEN** a client compares the three claim routes
- **THEN** each names a scope and then a claim within it, differing only in which scope

#### Scenario: The Universe reaches beyond the head's closure
- **WHEN** a claim is present in the Universe but outside the archive head's closure
- **THEN** `GET /universe/claims/{id}` returns it and `GET /archive/claims/{id}` reports not-found

#### Scenario: The archive scope spans branches
- **WHEN** a claim lies in the head's closure but in a different branch from the one a client would name
- **THEN** `GET /archive/claims/{id}` returns it without the client naming any branch

#### Scenario: Each scope is separately gated
- **WHEN** an account holds `R $archive` but no `R $universe`
- **THEN** `GET /archive/claims/{id}` is permitted and `GET /universe/claims/{id}` is denied by the access decision

#### Scenario: No path segment carries a reserved name
- **WHEN** the route set is inspected
- **THEN** no path contains a `$`-prefixed segment, those names appearing only in grants and in RankeQL bodies

### Requirement: Reserved top-level path segments do not collide with branches
The REST contract SHALL keep branch names out of the root namespace by addressing every
branch-scoped route under the `/branches` collection, so that a branch name occupies a
second path segment and never a first. No branch name SHALL be able to shadow a fixed
route, whatever it is called, and a new fixed route SHALL require no defensive prefix to
stay clear of branch reads.

#### Scenario: A branch cannot shadow a fixed route
- **WHEN** a branch is named `health`, `query` or `claims`
- **THEN** it is addressed under `/branches/{name}/…` and the fixed routes resolve unaffected

#### Scenario: The collection name may also be a branch name
- **WHEN** a branch is named `branches`
- **THEN** `GET /branches` lists the table and `GET /branches/branches/head` reads that branch's head

#### Scenario: A new root resource needs no prefix
- **WHEN** a resource is added at the root
- **THEN** it collides with no branch read, the root namespace carrying no wildcard segment
