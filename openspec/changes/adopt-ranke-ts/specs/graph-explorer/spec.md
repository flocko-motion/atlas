## ADDED Requirements

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

The RankeQL `Query` SHALL come from the ADT library instead. Both the library and this
repository generate their copy from `rql.schema.json`, the schema the specification
releases, so either would be faithful; taking it from the library keeps the claim model
and the language a claim is read with in one dependency, and the explorer then imports
the ADT from one place rather than meeting half of it through a transport client.

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
