# Minimal example

The smallest launchable `ranke-db` stack: in-memory storage, no auth, an
in-process signing identity.

The config is **secret-free** — `signer.key` is `env(RANKE_SIGNER_KEY)`, so the
file itself holds no key and is safe to commit. The signing identity is supplied
at launch from the environment (a real deployment would point this at a
`vault(...)` reference or an inline key in an age-encrypted config).

## Run it

From the repo root:

```sh
make dev
```

That builds the binary, mints a throwaway Ed25519 signing key, and launches this
config. Point it at another config with `make dev DEV_CONFIG=path/to/config.json`.

By hand, if you prefer — the signer key is an Ed25519 private key in PKCS#8 PEM
form, and the address comes from the config's `endpoints` section, not a flag:

```sh
export RANKE_SIGNER_KEY="$(openssl genpkey -algorithm ed25519)"
ranke-db run examples/minimal/config.json
```

`make smoke` runs the same cycle end-to-end (generate key, launch, health-check,
shut down) as a self-test.

## What a running instance answers today

The REST routes are mounted, but the core behind them is still being built, so
every one of them answers `501 {"code":"unimplemented","error":"core: not
implemented"}`:

| Route | Status |
|---|---|
| `GET /health`, `GET /system/layers` | mounted → 501 |
| `GET /{branch}/head`, `GET /{branch}/claim/{id}` | mounted → 501 |
| `POST /query`, `POST /contribute` | mounted → 501 |

Writing is doubly unavailable: besides the core, the sequencer port has no
backend (`adapters/sequencer`), because ranke-go's write path — `NewContribution`
and `Merge` — is still a draft upstream. So an instance cannot yet be seeded with
claims by any route, over HTTP or otherwise.
