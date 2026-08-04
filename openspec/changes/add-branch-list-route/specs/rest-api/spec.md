## MODIFIED Requirements

### Requirement: REST binds the paper's read-and-contribute surface
The REST endpoint SHALL expose the full surface as `POST /query` (a RankeQL query)
and `POST /contribute` (an atomic contribution of one or more signed claims), and
SHALL pin the cacheable GET subset `GET /$branches`, `GET /{branch}/head`,
`GET /{branch}/claim/{id}`, and `GET /$universe/claim/{id}`. The by-id GET forms SHALL
be HTTP-cacheable — the claim-by-id form immutably, the branch head and the branch
listing with revalidation. Each by-id claim route SHALL carry a `/content` form
(`GET /{branch}/claim/{id}/content`, `GET /$universe/claim/{id}/content`) that streams
that claim's content, so content is addressed by the claim holding it rather than by a
raw hash and the read stays scoped to the claim's branch.

`GET /$branches` SHALL return the branch table's branches, each carrying its name and
its current head id, so a client discovers what it may address without prior knowledge
of any branch name. It SHALL require the **R** right on `$branches`, the reserved
target the route is named for (see `core-access`).

#### Scenario: The full read surface is a query POST
- **WHEN** a client reads via `POST /query` with a RankeQL body
- **THEN** the response is the result set the query defines

#### Scenario: A by-id claim GET is cacheable without a JSON body
- **WHEN** a client issues `GET /{branch}/claim/{id}`
- **THEN** the claim is returned without a query body and is cacheable immutably by id

#### Scenario: The branch head GET is revalidated
- **WHEN** a client issues `GET /{branch}/head`
- **THEN** the current head id (a moving target) is returned as a cacheable-with-revalidation resource

#### Scenario: Content is fetched through the claim that holds it
- **WHEN** a client issues `GET /{branch}/claim/{id}/content`
- **THEN** the claim's content is streamed without the client knowing whether the bytes were stored inline or as a separate blob

#### Scenario: A client with no branch name discovers what it may read
- **WHEN** a client that knows no branch name issues `GET /$branches`
- **THEN** it receives every branch the table holds, each with its name and current head, and can address the other read routes from that list

#### Scenario: The branch listing is revalidated, not immutable
- **WHEN** a client issues `GET /$branches`
- **THEN** the response is cacheable with revalidation, the heads it carries being moving targets

#### Scenario: Listing branches needs R on the reserved target
- **WHEN** an account holding no `R $branches` grant issues `GET /$branches`
- **THEN** the request is denied by the access decision

### Requirement: Reserved top-level path segments do not collide with branches
The REST contract SHALL keep its fixed top-level routes (`/query`, `/contribute`,
`/health`, `/$branches`, `/system/…`) unambiguous against branch routes: branch reads
always carry a required suffix segment (`/head`, `/claim/{id}`), and a reserved target
begins with `$`, which is illegal in an ordinary branch name. No branch name SHALL be
able to shadow a fixed route, and no fixed route SHALL be ambiguous against
`/{branch}/head` under the router's matching rules.

A route named for a reserved `$`-target SHALL therefore be placeable at the top level,
being unshadowable by construction rather than by an argument about matching order.

#### Scenario: A branch route never shadows a fixed route
- **WHEN** the router resolves `/health` versus `/{branch}/head`
- **THEN** `/health` resolves to the health endpoint and branch reads resolve only with their required suffix

#### Scenario: An operational route cannot be ambiguous with a branch read
- **WHEN** an operational resource would sit at the top level with a single `{id}` segment
- **THEN** it is placed under `/system/` instead, since a root-level `/x/{id}` and `/{branch}/head` match the same path with neither more specific

#### Scenario: A branch cannot be named to reach the reserved route
- **WHEN** a client requests `/$branches`
- **THEN** it resolves to the branch listing, no ordinary branch being nameable with a `$`
