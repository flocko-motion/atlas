#import "vocabulary.typ": *

= Endpoints and authentication <ch:endpoints>

`endpoints` is an array, and each entry is one address the instance serves.

#example[
#listing[
```json
"endpoints": [{
  "transport": {"type": "rest", "addr": ":8080"},
  "auth":      [{"type": "apikey", "keys": [{"account": "ops", "sha256": "…"}]}],
  "admit":     ["ops"]
}]
```
]
]

#item("transport")[
  Where and how it listens.
]

#item("auth")[
  How a request's credential resolves to an account. An array, because one
  endpoint may accept more than one kind of credential.
]

#item("admit")[
  The accounts reachable here, by name.
]

An instance may serve several endpoints at once — a public port and a socket
for administration, say — and each carries its own authentication and its own
admitted accounts. Nothing is shared between them but the archive underneath.

== `transport` <sec:transport>

#item("type")[
  `"rest"` (or `"rest_http"`) serves the REST API over HTTP. `"mcp"` (or
  `"mcp_http"`) is reserved for the MCP transport, which the server refuses at
  launch until it is implemented.
]

#item("addr")[
  What to listen on. `":8080"` and `"127.0.0.1:8080"` are TCP;
  `"unix:///run/rankedb/admin.sock"` is a Unix domain socket. The scheme is
  what decides — a bare path is refused with the correction, and an overlong
  socket path is refused too, both by `verify --level syntax` rather than at
  launch.
]

#item("mode")[
  For a socket, its permission bits as an octal string. Default `"0600"` — the
  user the server runs as, and nobody else.
]

#item("group")[
  For a socket, a group name or numeric gid that also reaches it. This is how
  one host group is let in without opening the socket to everyone; a name that
  does not exist fails the launch rather than leaving a socket nobody can
  connect to.
]

#item("allowedOrigins")[
  A comma-separated list of browser origins allowed to call this endpoint.
  Absent means none, so a server nobody browses stays closed. Written as one
  string, so `env(...)` can supply it per deployment.
]

#warning[`"allowedOrigins": "*"` admits every origin. It suits a development
instance and nothing else.]

A socket suits an endpoint that should not be reachable over the network at
all. Forwarding one to a workstation is enough to use it:

#listing[
```sh
ssh -L 8080:/run/rankedb/admin.sock host
```
]

after which `http://127.0.0.1:8080` reaches the endpoint exactly as a network
address would.

== `auth` <sec:auth>

Each entry configures one authenticator, and each authenticator consumes one
credential scheme. A request is routed by the scheme it presents — never
guessed from the token's bytes — so two entries claiming one scheme are refused
at launch. A request presenting two credentials at once is answered `400`, and
one presenting none is answered `401` unless the endpoint configures `noauth`.

=== `noauth` <sec:noauth>

Authenticates every request as one fixed account. It consumes no credential and
so acts as the endpoint's fallback.

#item("subject")[
  The account every request acts as. Default `"anonymous"`.
]

#warning[`noauth` admits whoever can reach the address. Use it on a socket only
the operator reaches, or on a machine nobody else is on.]

=== `apikey` <sec:apikey>

Presented as the `X-API-Key` header. Keys shorter than 16 characters are
rejected.

#item("keys")[
  An array of `{"account": …, "sha256": …}`. The configuration holds each key's
  SHA-256 digest as lowercase hex, never the key itself, so a leaked file
  yields no usable credential — and the server cannot tell you a key it has
  forgotten.
]

=== `jwt` <sec:jwt>

Presented as `Authorization: Bearer …`.

#item("algorithm")[
  One of `HS256`, `HS384`, `HS512`, `RS256`, `RS384`, `RS512`, `ES256`,
  `ES384`, `ES512`, `PS256`, `PS384`, `PS512`, `EdDSA`. Taken from the
  configuration and never from the token.
]

#item("key")[
  A PEM public key, or the shared secret for an `HS*` algorithm. Exclusive with
  `jwks_url`, and one of the two is required.
]

#item("jwks_url")[
  A JWKS endpoint, for an issuer that rotates its keys.
]

#item("jwks_refresh")[
  How often that endpoint is refetched, as a duration. Default `5m`.
]

#item("account_claim")[
  The claim carrying the account name. Default `sub`.
]

#item("audience")[
  Held to, when set. Absent, the audience goes unchecked.
]

#item("issuer")[
  Held to, when set. Absent, the issuer goes unchecked.
]

=== `macaroon` <sec:macaroon>

Presented as `Authorization: Macaroon …`. The token's identifier is the account
and its first-party caveats are read as grants, which narrow what that account
may do for this request alone (@sec:attenuation).

#item("root_key")[
  The symmetric secret shared with whatever mints macaroons for this server.
]

#warning[A macaroon `root_key` verifies and mints alike: anyone holding it can
issue a token for any account. Prefer `env(...)` or `vault(...)` for it, and
for `key` and every other secret here, so the launch artifact carries the
wiring rather than the credentials.]

== `admit` <sec:admit>

`admit` lists the accounts this endpoint can act as. An account absent from
every `admit` list is defined and unreachable, and an `admit` entry naming an
account the roster does not define fails `verify --level syntax`.

Admission is per endpoint, and it is what separates one endpoint from another:
an authenticator may resolve a request to any account it likes, but a request
lands on the endpoint's own roster, and an account it does not admit gets
nothing.

#note[Two endpoints admitting the same account give it the same rights at both.
To narrow what one endpoint allows, give it a different account rather than a
smaller slice of this one.]
