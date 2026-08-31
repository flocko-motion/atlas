## 1. Choose the network from the address form

- [ ] 1.1 Read `addr` as today, then decide the network from its form: a leading `/` or `./`
      is a socket path, everything else a TCP address. Keep the `:8080` default.
- [ ] 1.2 Replace `ListenAndServe` with `net.Listen` on the chosen network followed by
      `srv.Serve(l)`, so the listener is the adapter's and the socket cases can act on it.
      `Serve` still reports `http.ErrServerClosed` on a graceful stop, so the existing
      `Serve`/`shutdown` control flow is unchanged.
- [ ] 1.3 Test both forms bind and serve one route, the TCP case proving the default path
      did not move.

## 2. The socket's permissions

- [ ] 2.1 Read `mode` from the transport section as an octal string, defaulting to `0600`,
      and reject a value that is not octal permission bits at build time rather than at bind.
- [ ] 2.2 Read the optional `group` as a name or gid, resolving a name to a gid, and set the
      socket's group after bind.
- [ ] 2.3 Apply the mode after bind and before `Serve`, so nothing is answered under the
      umask's bits.
- [ ] 2.4 Test that the default admits the server's own user, that a declared mode is what
      the file carries, and that a `group` naming no group on the host fails at launch with a
      clear message.

## 3. Claiming and releasing the path

- [ ] 3.1 On start, dial the path first: a successful connection means an endpoint is
      listening, and the bind refuses with a message naming the conflict.
- [ ] 3.2 A socket file that refuses a connection is stale — unlink it and bind.
- [ ] 3.3 A path holding anything other than a socket refuses, unlinking nothing.
- [ ] 3.4 Remove the socket in `shutdown`, on both the context-cancelled and `Close` paths,
      and once only when both run.
- [ ] 3.5 Test the three start cases and that a clean shutdown leaves no file. The
      live-listener case needs two instances against one path.

## 4. Offline validation

- [ ] 4.1 Check the address form in `verify`: a socket path within the platform's `sun_path`
      limit (107 bytes on Linux), a TCP address parsing as host and port. No filesystem
      access, no listener — `verify` stays offline.
- [ ] 4.2 Report a value parsing as neither form as a malformed address, naming the leading
      `/` or `./` a socket path needs, so `run/rankedb.sock` is answered with its correction
      rather than with a complaint about a network address.
- [ ] 4.3 Test that an overlong socket path fails `verify`, that `run/rankedb.sock` is
      reported with the convention named, and that a configuration binding a socket verifies
      with no socket created.

## 5. The example and its documentation

- [ ] 5.1 Add a second endpoint to `examples/minimal/config.json`: a socket under a directory
      the config names, `mode` `0660`, admitting an admin service account that the network
      endpoint does not admit.
- [ ] 5.2 Note in the README that the socket's directory carries the boundary during the
      instant between bind and chmod, and create it at `0700`.
- [ ] 5.3 Document the tunnel in the README — the `ssh -L 8080:/run/rankedb/admin.sock host`
      line and the explorer reading `http://127.0.0.1:8080` — so the pattern reads without
      being reconstructed.

## 6. Green gate

- [ ] 6.1 `make verify` green.
