## MODIFIED Requirements

### Requirement: REST/HTTP carries the full surface plus cacheable GET routes
The system SHALL expose over REST/HTTP the full surface as `POST /query` and
`POST /contribute`, and SHALL pin a subset of the query language as GET routes —
`GET /$branches`, `GET /{branch}/head`, `GET /{branch}/claim/{id}`, and
`GET /$universe/claim/{id}` — reachable without JSON and HTTP-cacheable: the by-id form
immutably, and branch reads with revalidation.

The listing SHALL be reachable without prior knowledge of any branch name, so a client
can discover what the archive holds before addressing the routes that take a branch as
a path parameter.

#### Scenario: A by-id GET is served cacheably without JSON
- **WHEN** a client issues `GET /{branch}/claim/{id}`
- **THEN** the claim is returned without a JSON query body and is cacheable immutably by id

#### Scenario: A client bootstraps from no branch name
- **WHEN** a client with no configured branch name connects to an instance
- **THEN** `GET /$branches` gives it the branch names the other GET routes require
