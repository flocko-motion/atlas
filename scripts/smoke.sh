#!/usr/bin/env sh
# smoke.sh — launch ranke-db against the minimal example, verify it serves the REST
# contract, and shut it down. A self-test that the run pipeline (load → resolve →
# assemble → serve) works end to end and that the configured endpoint is actually
# mounted. Arg 1: path to the ranke-db binary.
set -eu

BIN="${1:?usage: smoke.sh <ranke-db-binary>}"
CONFIG="examples/minimal/config.json"
PORT="8089"
ADDR="127.0.0.1:$PORT"

if ! command -v openssl >/dev/null 2>&1; then
	echo "smoke: openssl required to generate a throwaway signer key" >&2
	exit 1
fi

# A throwaway Ed25519 identity for the smoke run, supplied via env() — the
# adapter never generates keys; this is the operator providing one.
RANKE_SIGNER_KEY="$(openssl genpkey -algorithm ed25519)"
export RANKE_SIGNER_KEY

# The configuration is the composition root: an endpoint listens where its own
# section says, so the smoke port is set by editing the config, not by a flag.
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
sed "s/\":8080\"/\"$ADDR\"/" "$CONFIG" > "$WORK/config.json"

echo ">> launching: $BIN run <minimal example on $ADDR>"
"$BIN" run "$WORK/config.json" &
PID=$!
trap 'kill "$PID" 2>/dev/null || true; rm -rf "$WORK"' EXIT

# Wait for the listener, then check the contract's routes are mounted. What smoke
# asserts is that the endpoint the config named is serving the bound surface — not
# what the surface answers: core's execute stage is still a scaffold, so a routed
# read is a 501 today and a 200 once the engine lands. A 404 (or no answer at all)
# is the failure this catches: an endpoint built but never mounted.
code=""
for _ in 1 2 3 4 5 6 7 8 9 10; do
	code="$(curl -s -o /dev/null -m 2 -w '%{http_code}' "http://$ADDR/health" || true)"
	case "$code" in
	000 | "") sleep 0.3 ;;
	*) break ;;
	esac
done

case "$code" in
000 | "") echo "smoke: no answer from GET /health on $ADDR" >&2; exit 1 ;;
404) echo "smoke: GET /health is 404 — the configured endpoint is not mounted" >&2; exit 1 ;;
esac
echo ">> GET /health routed ($code)"

# The query route also answers on the wire, and its boundary check runs ahead of
# core: a query naming no scope is a 400 whatever the engine behind it does.
code="$(curl -s -o /dev/null -m 5 -w '%{http_code}' -X POST "http://$ADDR/query" \
	-H 'content-type: application/json' -d '{"select":{}}')"
if [ "$code" != "400" ]; then
	echo "smoke: POST /query with no scope returned $code, want 400" >&2
	exit 1
fi
echo ">> POST /query rejects a scopeless query (400) OK"

echo ">> shutting down"
kill "$PID" 2>/dev/null || true
wait "$PID" 2>/dev/null || true
trap 'rm -rf "$WORK"' EXIT
echo ">> smoke OK"
