## Why

An admin endpoint is already expressible, and it still has to sit on a reachable port.
Each entry in `endpoints` carries its own authenticators and its own access checker built
from the accounts it admits alone (`config/config.go:245-268`), and every configured
endpoint mounts concurrently, so an exposure that recognises one service account and
nothing else is a matter of configuration today. What no configuration can say is that such an exposure
should be unreachable from the network.

`rest_http` builds an `http.Server` around a configured `addr` and serves it with
`ListenAndServe`, which is fixed to TCP (`rest_http.go:69-76`). No other place in the
repository opens a listener. A socket path in `addr` is therefore accepted by `verify`,
which checks the field as an unconstrained string, and then fails at launch inside
`ListenAndServe` with `missing port in address` — a configuration that validates and
cannot run.

The use this answers: bind the admin endpoint to a socket, forward it over ssh
(`ssh -L 8080:/run/rankedb/admin.sock host`), and read the instance from the Ranke Explorer
at `http://127.0.0.1:8080`. OpenSSH has forwarded a local port to a remote socket since
6.7. It stays REST over HTTP throughout, and the client learns nothing of the socket.

## What Changes

- The **form of the address decides the network**: a value beginning `/` or `./` binds a
  Unix domain socket, and every other value stays a TCP address. One field, and the
  configuration that fails at launch today starts working.
- The socket file's permissions become the endpoint's access boundary, declared on the
  transport section: **`mode`**, octal, defaulting to `0600`, and an optional **`group`**
  owning the socket. A server and the user who forwards its socket are rarely the same
  user, so a shared group with `0660` is what admits a client; `mode` alone could only
  widen to `0666`, which admits every local user.
- Shutdown removes the socket. A start finding the path already bound refuses, a start
  finding a socket nobody listens on replaces it, and a start finding anything else there
  refuses.
- Offline `verify` checks the address form, including the length limit an operating system
  imposes on a socket path — the one misconfiguration that is invisible in a config file
  and fatal at launch.
- The minimal example gains a second endpoint on a socket, and its README the `ssh -L`
  line, so the pattern is readable rather than reconstructed.

## Capabilities

### Modified Capabilities

- `adapter-endpoint`: an endpoint binds either a network address or a local socket, the
  socket file's permissions being its access boundary, and the address form is checked
  offline.

## Impact

- **`adapters/endpoints/rest_http/`**: `rest_http.go` — choose the network from the address
  form, `net.Listen` and `srv.Serve(l)` in place of `ListenAndServe`, set the mode and group
  before serving, remove the socket on shutdown.
- **`config/`**: the offline form check, alongside the referential check `admit` already
  gets.
- **`examples/minimal/`**: a socket endpoint admitting an admin account, and the tunnel
  line in the README.
- **MCP has nothing to inherit**: `mcp_http` is a five-line stub holding a package clause and
  no implementation, and `endpoints.New` answers `mcp transport not yet implemented` rather
  than routing to it. The requirements here are written at the endpoint level rather than at
  the REST backend, so an MCP transport must satisfy them when it is built.
- **Not affected**: `openapi/openapi.yaml` and everything generated from it. The wire
  contract is unchanged, so neither the Go server interface, the TS client, nor `frontend/`
  moves.
- **Not affected**: `core-access`, `adapter-auth`. Per-endpoint isolation is already
  specified and already built; a socket removes an exposure rather than granting anything.
- **Not affected**: the cross-origin posture. Through a forwarded port the browser's origin
  is `http://127.0.0.1:<local port>`, which `allowedOrigins` already admits, and the
  embedded `/explorer` reached over the same tunnel is same-origin.
- **Deferred by decision**: peer-credential authentication. A socket can name the uid on the
  other end, and turning that into a subject is a new authenticator rather than a property of
  the bind (see `design.md`).
