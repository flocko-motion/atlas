# Minimal example

The smallest launchable `ranke-db` stack: in-memory storage, no auth, an
in-process signing identity.

The config is **secret-free** — `signer.key` is `env(RANKE_SIGNER_KEY)`, so the
file itself holds no key and is safe to commit. The signing identity is supplied
at launch from the environment (a real deployment would point this at a
`vault(...)` reference or an inline key in an age-encrypted config).

## Run it

The signer key is an Ed25519 private key in PKCS#8 PEM form. Generate a
throwaway one for local use and launch:

```sh
export RANKE_SIGNER_KEY="$(openssl genpkey -algorithm ed25519)"
ranke-db run examples/minimal/config.json
# serves on :8080 — try: curl localhost:8080/healthz
```

`make smoke` does exactly this end-to-end (generate key, launch, health-check,
shut down) as a self-test.
