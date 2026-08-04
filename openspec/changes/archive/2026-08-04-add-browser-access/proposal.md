## Why

The Ranke Explorer is a browser client, and a browser will not let a page read an origin
that has not admitted it. The REST endpoint sends no `Access-Control-*` headers and answers
no preflight, so every route the explorer needs is unreachable from a page — the client and
the API work together over any other transport and not the one they are for.

## What Changes

- The REST endpoint answers a browser's cross-origin checks for the origins its config
  declares: `allowedOrigins`, a comma-separated list (`*` for any), on the transport section.
- Declaring none stays closed, which is the current behaviour and the right default for a
  server nobody browses.
- `ETag` is **exposed**, without which a script cannot read it and the contract's conditional
  reads are unusable from a browser.
- No origin is granted credentialed access. A credential rides in a header here, never in a
  cookie, so there is nothing for a browser to attach on a user's behalf.
- The minimal example admits the explorer's dev origins, so `make dev` and
  `make -C frontend run` work together out of the box.

## Capabilities

### Modified Capabilities

- `adapter-endpoint`: the REST transport answers cross-origin checks for configured origins.

## Impact

- **`adapters/endpoints/rest_http/`**: `cors.go`, wrapped outermost — a preflight carries no
  credential and must be answered before one is extracted.
- **`examples/minimal/config.json`**: `allowedOrigins` for the vite dev and preview ports.
- **`openapi/openapi.yaml`**: a "Browsers" section describing the posture.
- **Not affected**: `core-access`. Admitting an origin is not granting a subject anything;
  every request still authenticates and is authorized exactly as before.
