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
`GET /{branch}/head`, `GET /{branch}/claim/{id}`, and `GET /$universe/claim/{id}` —
reachable without JSON and HTTP-cacheable: the by-id form immutably and branch reads
with revalidation.

#### Scenario: A by-id GET is served cacheably without JSON
- **WHEN** a client issues `GET /{branch}/claim/{id}`
- **THEN** the claim is returned without a JSON query body and is cacheable immutably by id

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
