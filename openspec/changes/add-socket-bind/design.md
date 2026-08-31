## The address form decides, not a second field

The alternative was a `network` field taking `tcp` or `unix`, with `addr` read according to
it. It was rejected because the two address forms do not overlap: a TCP address carries a
port after a colon and a socket path begins at the root or at `./`. A second field
therefore adds no information the first does not already hold, while adding a pair of
fields that can disagree — and a config saying `"network": "tcp", "addr": "/run/x.sock"` is
a new way to be wrong that nobody needs.

Inference also repairs the present failure directly. `"addr": "/run/rankedb.sock"` is a
configuration someone writes expecting it to work; today it validates and dies at launch.
Under inference that same text is the feature.

What inference costs is one gap, and it is worth naming rather than discovering later. That
a socket path begins at `/` or `./` is a convention this change imposes; it is not a fact
about paths, and `run/rankedb.sock` is a value someone will write. Widening the rule to any
value containing a slash was considered and rejected, since it would read a TCP address of
the form `host/…` as a path where no such address exists today but might. So the rule stands
and the error carries the convention: a value parsing as neither form is reported as
malformed, naming the leading `/` or `./` a socket path needs. The writer gets the correction
rather than a bare complaint about a network address they never meant.

A scheme prefix (`unix:/run/x.sock`) was the third option. It keeps one field and stays
explicit, at the price of a URL-shaped syntax that appears nowhere else in the
configuration, so it was dropped for consistency.

## The mode is configuration, and closed by default

Who may open the socket is an operational choice about one deployment, which puts it in the
configuration by the content/policy razor. The server cannot derive it: it runs as whatever
user the host's admin chose, and the client is whoever that admin decided may reach it.

The default is `0600` — the user the server runs as, and nobody else. That follows
`allowedOrigins`, where declaring nothing keeps a server no browser is meant to reach
unreachable (`cors.go:8-10`). An access boundary that widens only when a config says so is
the same principle applied to a file.

`group` exists because `mode` alone cannot express the ordinary case. When the server runs
as a service user and the tunnel is opened by a person, no permission bits on a
server-owned, server-grouped socket admit that person short of `0666`, which admits every
local account. `0660` with a shared group admits one group, and the server's user being a
member of it is the host admin's arrangement to make.

There is no `owner`. The socket is owned by the uid that created it, and chowning it to a
different owner needs root, which a server should not have.

## The permissions gap at bind, and where the boundary really sits

`net.Listen("unix", path)` creates the file under the process umask, so between the bind
and the `chmod` there is an instant in which the socket may be more open than the config
asked for. Setting the umask around the call does not fix this: umask is process-wide and
another goroutine binding a second endpoint would race on it.

The honest answer is that the containing directory carries the boundary. A socket in a
directory the server creates at `0700` is unreachable during that instant regardless of the
socket's own bits, because a caller cannot traverse the directory to reach it. The mode on
the socket then expresses intent within a directory that is already closed, and the example
config puts the socket somewhere that says so.

## A stale socket is replaced; a live one is refused

Three cases meet at a path that already exists.

A path where an endpoint is listening belongs to a running instance. One process serves one
configuration, so a second instance launched against the same file must fail loudly rather
than unlink the socket underneath the first — which would leave the first process serving a
socket no client can find, with nothing in its log. Liveness is settled by dialling the
path.

A socket file nobody is listening on is the residue of a process that did not shut down
cleanly, and refusing it would make a crash require manual cleanup before a restart. It is
replaced.

Anything else at the path — a regular file, a directory — is a misconfiguration pointing at
something that was not ours, and unlinking it would destroy a file the config merely named
by mistake. It is refused.

## Peer credentials are a separate subject

A Unix socket can report the uid on the other end, which is a stronger claim than any
bearer token, and it is tempting to let the socket authenticate. It is out of scope here.
An endpoint pairs a transport with an authenticator, so deriving a subject from a peer's uid
is a new authenticator behind `adapter-auth` — with its own mapping from uid to account,
its own configuration, and its own conformance case. The bind is complete without it: the
file's permissions decide who may reach the endpoint, and the account's credential decides
what they may do, exactly as on a network port.

## Resolved while planning

Two premises were checked and one was wrong.

**Per-endpoint privilege already exists.** The first reading of the code was that access
cannot vary by endpoint, since `core.Request` carries a credential, an operation and a
branch, and nothing identifying the endpoint (`internal/core/request.go:76-106`). That
reading missed `admit`: `buildEndpoint` gives each endpoint its own `core.Core` over a
checker constructed from the admitted accounts alone (`config/config.go:245-268`), so an
account absent from an endpoint's `admit` list has no grants there and is unknown to it.
An admin exposure therefore needs no access work — only a bind the network cannot reach.

**Grants live on the account, not on the pairing.** An account admitted by two endpoints
carries the same grants at both. Where one exposure should be wider than the other, that is
two accounts, and since accounts here are service accounts rather than people, a second
account is the accurate model rather than a workaround.
