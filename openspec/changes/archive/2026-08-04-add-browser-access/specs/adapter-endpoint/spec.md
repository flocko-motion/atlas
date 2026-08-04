## ADDED Requirements

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
