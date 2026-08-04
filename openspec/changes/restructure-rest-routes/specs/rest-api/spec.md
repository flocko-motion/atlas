## MODIFIED Requirements

### Requirement: REST binds the paper's read-and-contribute surface
The REST endpoint SHALL expose the full surface as `POST /query` (a RankeQL query)
and `POST /contribute` (an atomic contribution of one or more signed claims), and
SHALL pin the cacheable GET subset:

- `GET /branches` — the branch table's branches, each by name and current head
- `GET /branches/{branch}/head` — that branch's current head id
- `GET /branches/{branch}/claims/{id}` — a claim within that branch's closure
- `GET /branches/{branch}/claims/{id}/content` — that claim's content
- `GET /claims/{id}` — a claim anywhere in the archive, the read carrying no branch scope
- `GET /claims/{id}/content` — that claim's content

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

### Requirement: An unscoped claim read is the privileged archive-wide read
The REST contract SHALL express the privileged read — reaching a claim by id anywhere in
the archive, bypassing the branch table — as a claim route carrying **no branch scope**:
`GET /claims/{id}`. The absence of a branch SHALL be what makes the read archive-wide,
so the path's shape carries the distinction rather than a sentinel name.

That route SHALL require the **R** right on the reserved `$universe` target, and SHALL
NOT spell that target in the path. Reserved `$`-names belong to the grants language and
to RankeQL scope values; the URL path SHALL contain none of them.

#### Scenario: A claim is reached without naming a branch
- **WHEN** an account holding `R $universe` issues `GET /claims/{id}`
- **THEN** the claim is returned for any id the Universe holds, without a branch being named

#### Scenario: The privileged read is still gated
- **WHEN** an account holding no `R $universe` grant issues `GET /claims/{id}`
- **THEN** the request is denied by the access decision

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
