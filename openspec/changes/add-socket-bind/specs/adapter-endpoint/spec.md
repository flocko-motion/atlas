## ADDED Requirements

### Requirement: An endpoint binds either a network address or a local socket
The system SHALL bind an endpoint to a Unix domain socket when its configured address
carries the `unix://` scheme, and to a TCP address otherwise. The socket path SHALL be the
remainder of the address as written, so `unix:///run/rankedb/admin.sock` names
`/run/rankedb/admin.sock`. An address of `unix://` alone SHALL be refused, naming no socket
to bind. Every address without the scheme SHALL bind as it does today.

A socket endpoint SHALL carry the same surface over the same protocol as a network one, so a
client reaching it through a forwarded port SHALL need no knowledge that a socket carries
it, and no part of the wire contract SHALL change.

#### Scenario: A scheme-prefixed address binds a socket
- **WHEN** an endpoint is configured with `"addr": "unix:///run/rankedb/admin.sock"`
- **THEN** it listens on the Unix domain socket at `/run/rankedb/admin.sock` and on no network port

#### Scenario: A scheme naming no path is refused
- **WHEN** an endpoint is configured with `"addr": "unix://"`
- **THEN** the endpoint refuses to build, the address naming no socket

#### Scenario: A network address is unaffected
- **WHEN** an endpoint is configured with `"addr": ":8080"`
- **THEN** it listens on that TCP address exactly as before

#### Scenario: A forwarded socket serves an ordinary client
- **WHEN** a client reaches a socket endpoint through a locally forwarded port
- **THEN** it is served the same routes, media types, and validators as on a network endpoint

### Requirement: The socket file's permissions are the endpoint's access boundary
The system SHALL take a socket endpoint's file permissions from its configuration: a `mode`
giving the permission bits, and an optional `group` owning the socket file. Both SHALL be
set before the endpoint serves its first request.

An endpoint declaring no `mode` SHALL be reachable by the user the server runs as and by no
other, a socket nobody was granted staying as closed as a network endpoint declaring no
origins.

The system SHALL NOT take an owner from the configuration, a socket being owned by the user
that created it.

#### Scenario: An undeclared mode admits the server's user alone
- **WHEN** a socket endpoint is configured with no `mode`
- **THEN** the socket admits the user the server runs as and refuses every other local user

#### Scenario: A declared group admits that group
- **WHEN** a socket endpoint declares `"mode": "0660"` and a `group`
- **THEN** the socket is owned by that group and members of it may connect

#### Scenario: Permissions are in place before the first request
- **WHEN** a socket endpoint starts serving
- **THEN** its mode and group are already set, the endpoint answering nothing beforehand

### Requirement: A socket path is claimed and released cleanly
The system SHALL remove the socket file when the endpoint shuts down.

On start, the system SHALL refuse a path where an endpoint is already listening, one process
serving one configuration and a second instance therefore failing rather than displacing the
first. It SHALL replace a socket file no process is listening on, so an unclean shutdown
needs no manual cleanup before a restart. It SHALL refuse a path holding anything other than
a socket, a mistyped path naming a file that was never ours to remove.

#### Scenario: A shutdown leaves no socket behind
- **WHEN** a socket endpoint shuts down
- **THEN** the socket file is removed

#### Scenario: A second instance is refused
- **WHEN** an instance starts against a path where another instance is listening
- **THEN** it refuses to bind and reports the conflict, the running endpoint continuing to serve

#### Scenario: A stale socket is replaced
- **WHEN** an instance starts against a socket file no process is listening on
- **THEN** it replaces the file and serves

#### Scenario: A path holding a regular file is refused
- **WHEN** an instance starts against a path holding a regular file or a directory
- **THEN** it refuses to bind and removes nothing

### Requirement: An address's form is checked offline
The system SHALL check a configured address's form during offline `verify`, without opening
a listener or touching the filesystem. A socket path exceeding the length the operating
system admits for a socket path SHALL be reported by `verify`, this being the one
misconfiguration a config file cannot show and a launch cannot survive.

A filesystem path carrying no scheme SHALL be reported as a malformed address naming the
`unix://` the address needs, since a value such as `/run/rankedb.sock` is plainly meant as a
socket and would otherwise be reported only as an invalid network address.

#### Scenario: A bare socket path is answered with the scheme it needs
- **WHEN** `verify` runs against an endpoint address of `/run/rankedb.sock`
- **THEN** it reports the address as malformed and names the `unix://` scheme a socket requires

#### Scenario: An overlong socket path fails verification
- **WHEN** `verify` runs against a configuration whose endpoint address is a socket path longer than the platform admits
- **THEN** it reports the address as invalid rather than deferring the failure to launch

#### Scenario: Verification opens no listener
- **WHEN** `verify` runs against a configuration binding a socket
- **THEN** it validates the address form while creating no socket and binding no port
