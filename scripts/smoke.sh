#!/usr/bin/env sh
# smoke.sh — launch ranke-db against the minimal example, verify it serves, and
# shut it down. A self-test that the run pipeline (load → resolve → assemble →
# serve) works end to end. Arg 1: path to the ranke-db binary.
set -eu

BIN="${1:?usage: smoke.sh <ranke-db-binary>}"
CONFIG="examples/minimal/config.json"
ADDR="127.0.0.1:8089"

if ! command -v openssl >/dev/null 2>&1; then
	echo "smoke: openssl required to generate a throwaway signer key" >&2
	exit 1
fi

# A throwaway Ed25519 identity for the smoke run, supplied via env() — the
# adapter never generates keys; this is the operator providing one.
RANKE_SIGNER_KEY="$(openssl genpkey -algorithm ed25519)"
export RANKE_SIGNER_KEY

echo ">> launching: $BIN run $CONFIG (addr $ADDR)"
"$BIN" run --addr "$ADDR" "$CONFIG" &
PID=$!
trap 'kill "$PID" 2>/dev/null || true' EXIT

# Wait for the listener, then probe health.
ok=0
for _ in 1 2 3 4 5 6 7 8 9 10; do
	if curl -fsS "http://$ADDR/healthz" >/dev/null 2>&1; then
		ok=1
		break
	fi
	sleep 0.3
done

if [ "$ok" -ne 1 ]; then
	echo "smoke: server did not become healthy" >&2
	exit 1
fi

echo ">> /healthz OK — shutting down"
kill "$PID" 2>/dev/null || true
wait "$PID" 2>/dev/null || true
trap - EXIT
echo ">> smoke OK"
