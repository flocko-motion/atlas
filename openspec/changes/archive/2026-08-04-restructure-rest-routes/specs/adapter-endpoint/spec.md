## MODIFIED Requirements

### Requirement: REST/HTTP carries the full surface plus cacheable GET routes
The system SHALL expose over REST/HTTP the full surface as `POST /query` and
`POST /contribute`, and SHALL pin a subset of the query language as GET routes —
`GET /branches`, `GET /branches/{branch}/head`, `GET /branches/{branch}/claims/{id}`,
`GET /archive/claims/{id}` and `GET /universe/claims/{id}` — reachable without JSON and
HTTP-cacheable: the by-id forms immutably, the branch listing and branch head with
revalidation.

Those routes SHALL be typable as they stand, needing no quoting or escaping, since being
reachable from a browser or `curl` without a JSON body is what they are pinned for.

The listing SHALL be reachable without prior knowledge of any branch name, so a client
can discover what the archive holds before addressing the routes that take a branch.

#### Scenario: A by-id GET is served cacheably without JSON
- **WHEN** a client issues `GET /branches/{branch}/claims/{id}`
- **THEN** the claim is returned without a JSON query body and is cacheable immutably by id

#### Scenario: A client bootstraps from no branch name
- **WHEN** a client with no configured branch name connects to an instance
- **THEN** `GET /branches` gives it the branch names the branch routes require

#### Scenario: A route survives a shell
- **WHEN** a route is typed unquoted into `curl`
- **THEN** the request reaches the path as written, no segment being altered by shell expansion
