## 1. Read the scheme and bind through a listener

The shape here was built once already (`pr-sd-1e4834`, rejected on the groups below) and
was right: keep it.

- [x] 1.1 `parseAddr` reads `unix://` and returns the network and the path, refusing
      `unix://` with no path. Anything without the scheme is a TCP address as before.
- [x] 1.2 `net.Listen` on the parsed network followed by `srv.Serve(ln)` in place of
      `ListenAndServe`, binding in `Serve` rather than in `New` so building an endpoint from
      config claims neither a port nor a socket file.
- [x] 1.3 Test both forms bind and serve a real request over the listener they claim, the TCP
      case proving the default path did not move.

## 2. The socket's permissions

The socket lands under the process umask without this — 0755 on a usual 0022 — so every
local account may connect to the exposure that was moved off the network to prevent exactly
that.

- [x] 2.1 Read `mode` from the transport section as an octal string, defaulting to `0600`,
      and reject a value that is not octal permission bits at build time rather than at bind.
- [x] 2.2 Read the optional `group` as a name or gid, resolving a name to a gid, and set the
      socket's group after bind.
- [x] 2.3 Apply the mode after bind and before `Serve`, so nothing is answered under the
      umask's bits.
- [x] 2.4 Test that the default admits the server's own user, that a declared mode is what
      the file carries, and that a `group` naming no group on the host fails at launch with a
      clear message.

## 3. Claiming and releasing the path

A socket file cannot be told from a live one by `os.Stat` — both are sockets — so only a
dial distinguishes residue from a running instance.

- [x] 3.1 On start, dial the path first: a successful connection means an endpoint is
      listening, and the bind refuses with a message naming the conflict. Without this a
      second instance unlinks the first's socket and takes over the path, leaving a live
      process serving an inode no client can reach and nothing in its log to say so.
- [x] 3.2 A socket file that refuses a connection is stale — unlink it and bind.
- [x] 3.3 A path holding anything other than a socket refuses, unlinking nothing.
- [x] 3.4 Remove the socket in `shutdown`, on both the context-cancelled and `Close` paths,
      and once only when both run. `net.Listen` returns a listener that unlinks on close, so
      this is already the behaviour — state it in the adapter rather than inheriting it from a
      default, since a listener supplied from elsewhere would silently drop it.
- [x] 3.5 Test the three start cases and that a clean shutdown leaves no file. The
      live-listener case needs two instances against one path.

## 4. Offline validation

- [x] 4.1 Check the address form in `verify`: a `unix://` path within the platform's
      `sun_path` limit (107 bytes on Linux), a TCP address parsing as host and port. No
      filesystem access, no listener — `verify` stays offline.
- [x] 4.2 Report a filesystem path carrying no scheme as a malformed address, naming the
      `unix://` it needs, so `/run/rankedb.sock` is answered with its correction rather than
      with a complaint about a network address.
- [x] 4.3 Test that an overlong socket path fails `verify`, that `/run/rankedb.sock` is
      reported with the scheme named, and that a configuration binding a socket verifies with
      no socket created.

## 5. The example and its documentation

- [x] 5.1 Add a second endpoint to `examples/minimal/config.json`: `unix://` under a
      directory the config names, `mode` `0660`, admitting an admin service account that the
      network endpoint does not admit.
- [x] 5.2 Note in the README that the socket's directory carries the boundary during the
      instant between bind and chmod, and create it at `0700`.
- [x] 5.3 Document the tunnel in the README — the `ssh -L 8080:/run/rankedb/admin.sock host`
      line and the explorer reading `http://127.0.0.1:8080` — so the pattern reads without
      being reconstructed.

## 6. Green gate

- [x] 6.1 `make verify` green.
