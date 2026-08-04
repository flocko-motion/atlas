#!/usr/bin/env sh
# smoke.sh — launch ranke-db against the minimal example, seed it over its own REST API,
# read the graph back, and shut down. A self-test that the whole pipeline works end to
# end: load → resolve → assemble → serve, then contribute → merge → query.
# Arg 1: the ranke-db binary. Arg 2: the generator binary.
set -eu

BIN="${1:?usage: smoke.sh <ranke-db-binary> <generator-binary>}"
GEN="${2:?usage: smoke.sh <ranke-db-binary> <generator-binary>}"
CONFIG="examples/minimal/config.json"
PORT="8089"
ADDR="127.0.0.1:$PORT"
URL="http://$ADDR"

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

# Wait for the listener before asserting anything about what it answers.
code=""
for _ in 1 2 3 4 5 6 7 8 9 10; do
	code="$(curl -s -o /dev/null -m 2 -w '%{http_code}' "$URL/health" || true)"
	case "$code" in
	000 | "") sleep 0.3 ;;
	*) break ;;
	esac
done

case "$code" in
000 | "") echo "smoke: no answer from GET /health on $ADDR" >&2; exit 1 ;;
200) ;;
*) echo "smoke: GET /health returned $code, want 200" >&2; exit 1 ;;
esac
echo ">> GET /health OK"

# Seeding runs as a client, the way an application contributes: claims signed with its
# own key, sent as a contribution stream. This is the write path's end-to-end check.
echo ">> seeding over POST /contribute"
"$GEN" example "$URL" --wait 5s

# And what was written reads back. The seeded entity is the one claim whose content is
# distinctive enough to find in the result without parsing the sequence.
echo ">> POST /query reads the seeded graph back"
body="$(curl -s -m 10 -X POST "$URL/query" \
	-H 'content-type: application/json' \
	-d '{"select":{"branch":"main"},"output":{"encoding":"json"}}')"
if ! printf '%s' "$body" | grep -q '"type":"entity/person"'; then
	echo "smoke: the seeded entity/person claim did not come back from /query" >&2
	printf 'smoke: got %s\n' "$body" >&2
	exit 1
fi
echo ">> seeded claims read back OK"

# The boundary check runs ahead of core: a query naming no scope is a 400 whatever the
# engine behind it does.
code="$(curl -s -o /dev/null -m 5 -w '%{http_code}' -X POST "$URL/query" \
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
