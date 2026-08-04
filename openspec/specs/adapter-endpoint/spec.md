# adapter-endpoint Specification

## Purpose
An endpoint binds a transport to an authenticator and carries the full
read-and-contribute surface into the core, translating between an endpoint-specific
request/response format and internal calls. RankeDB ships REST/HTTP (OpenAPI) for
webapps and MCP/HTTP for agents. This capability defines the port contract and the
shipped endpoints.
## Requirements
### Requirement: An endpoint pairs a transport with an authenticator
The system SHALL receive client requests in an endpoint-specific format, extract the
auth token and pass it with the call so the core can match the request against the
account's access rights (see `adapter-auth`, `core-access`), and return results in
that endpoint's format.

#### Scenario: The endpoint passes the token to the core
- **WHEN** a request arrives at an endpoint
- **THEN** the endpoint extracts its auth token and passes it with the internal call for authorization

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

### Requirement: MCP/HTTP exposes the surface as agent tools
The system SHALL expose the read-and-contribute surface over MCP/HTTP as agent tools.

#### Scenario: An agent calls query and contribute as tools
- **WHEN** an agent connects over the MCP endpoint
- **THEN** it can query and contribute through MCP tools carrying the same surface as REST

### Requirement: Multiple endpoints may be configured
The system SHALL allow several endpoint adapters to be configured, exposing the
instance on several ports, addresses, or protocols, and each endpoint SHALL support
every authentication mechanism.

#### Scenario: Two endpoints run with different auth
- **WHEN** the configuration mounts a REST endpoint and an MCP endpoint on different ports
- **THEN** both serve the instance concurrently, each with its configured authentication mechanism

### Requirement: The REST transport admits the browser origins its config declares

The REST/HTTP endpoint SHALL answer a browser's cross-origin checks for each origin its
configuration declares, and SHALL answer none where the configuration declares none — a
server no page is meant to reach staying unreachable by default.

For an admitted origin it SHALL answer a preflight with the methods it serves and the
credential headers the contract enumerates, and SHALL **expose** `ETag`, without which a
script cannot read the validator and the conditional reads the contract describes cannot be
made from a browser.

It SHALL NOT grant any origin credentialed access. A credential is presented in a header
here rather than in a cookie, so there is nothing a browser could attach on a user's behalf,
and admitting an origin SHALL confer no authority: every request is authenticated and
authorized as it was before.

A request carrying no `Origin` SHALL be unaffected.

#### Scenario: A declared origin may read the API
- **WHEN** a page on a declared origin issues a read
- **THEN** the answer admits that origin and exposes `ETag`

#### Scenario: A preflight is answered before the credential stage
- **WHEN** a browser preflights a read with no credential attached
- **THEN** the endpoint answers the check itself, the request never reaching authentication

#### Scenario: An undeclared origin is not admitted
- **WHEN** a page on an origin the configuration does not name issues a read
- **THEN** the answer admits no origin, which the browser reads as refused

#### Scenario: A server declaring no origins stays closed
- **WHEN** no origins are configured
- **THEN** no cross-origin answer is given to any origin

#### Scenario: A non-browser client is unaffected
- **WHEN** a request arrives with no `Origin` header
- **THEN** it is served exactly as before

