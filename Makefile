# ranke-db — repo-level targets.
#
# The OpenAPI spec is the single source of truth; `make generate` produces every
# artifact from it into openapi/ (alongside the spec) — the Go server, the TS/JS
# client, and the HTML + Markdown references — and symlinks the two references
# under docs/openapi/.

OPENAPI   := openapi/openapi.yaml
API_OUT   := openapi
# The explorer reads an instance through this client, and `frontend/` builds on its own
# (its Makefile and package.json are not wired into this one). So the generated client is
# copied in and committed, as `frontend/explorer.html` is: `make -C frontend` needs nothing
# from here, and `check-generated` notices when the copy goes stale.
EXPLORER_CLIENT := frontend/src/core/data/openapi.gen.ts
# openapi.yaml $refs rql.schema.json, which no generator resolves on its own, so
# every one of them reads this bundle: the same document with the external schema
# lifted into components/schemas.
OPENAPI_GEN := openapi/openapi.gen.yaml

# The node generators, pinned exactly. A floating `latest` or major range lets an
# upstream release rewrite the artifacts, which trips check-generated for a reason
# that is not ours. Bump deliberately, then regenerate.
REDOCLY     := @redocly/cli@2.44.1
SWAGGER_TS  := swagger-typescript-api@13.12.6
WIDDERSHINS := widdershins@4.0.1
RANKE_GO_MOD ?= github.com/flocko-motion/ranke-go
# The library's other half. frontend/ keeps its own report, but nothing invokes it, so a
# stale pin sat unseen for eleven days while the two halves disagreed about the wire format.
# The pin is read straight out of package.json rather than delegated to that target, which
# would drag node_modules into `verify` and wire frontend/ into a build it stays out of.
RANKE_TS_PKG ?= @flocko-motion/ranke
FRONTEND_PKG := frontend/package.json
RANKE_GO_VERSION ?= latest
# ask = prompt before raising the go directive; keep = leave it; or a version.
GO_VERSION ?= ask

RANKE_GRAPH_REPO ?= https://github.com/flocko-motion/ranke-graph
RANKE_GRAPH_REF  ?= main
PAPERS_DIR       := docs/papers
# Everything at the top of the paper repo is reference material and gets pulled;
# these are the exceptions (its own tooling). Dotdirs never match the glob.
PAPERS_SKIP      := scripts

.PHONY: all help check-tools generate verify tidy build smoke test dev seed \
        ranke-go-version upgrade release major minor patch breaking feature fix docs docs-clean \
        pull-rql-schema check-rql-schema check-generated release-gate

BIN := bin/ranke-db
GEN := bin/generator

.DEFAULT_GOAL := all

all: generate verify ## Default: regenerate from the spec, then build/vet/test

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

# --- The RQL schema, pulled from ranke-graph -------------------------------

# openapi.yaml $refs this file for the whole read language, so the query type is
# ranke-graph's and this repo holds no second copy of it. Committed, which keeps
# `make generate` offline and reproducible; run pull-rql-schema to take a newer
# revision and review the diff.
#
# Taken from the spec on `main`, not from the latest release: the specification is the
# source of truth, and an implementation that disagrees with it has the bug. A release is
# a snapshot of that source, so vendoring one means serving whatever the spec said when it
# was cut — which is how this contract came to require `output.content.overflow` for a day
# after R-QCONTENT made it optional. Point RQL_SCHEMA_URL at a release to pin one instead.
RQL_SCHEMA     := $(API_OUT)/rql.schema.json
RQL_SCHEMA_URL ?= https://raw.githubusercontent.com/flocko-motion/ranke-graph/refs/heads/$(RANKE_GRAPH_REF)/spec/rql.schema.json

pull-rql-schema: ## Pull rql.schema.json from the ranke-graph spec into openapi/
	@command -v curl > /dev/null || { echo "curl not found"; exit 1; }
	@command -v jq > /dev/null || { echo "jq not found"; exit 1; }
	@echo ">> pull     → $(RQL_SCHEMA)  ($(RQL_SCHEMA_URL))"
	@tmp=$$(mktemp); \
	curl -fsSL "$(RQL_SCHEMA_URL)" -o "$$tmp" || { rm -f "$$tmp"; echo "download failed"; exit 1; }; \
	jq -e 'has("$$defs") and (.["$$defs"] | has("Query"))' "$$tmp" > /dev/null \
		|| { rm -f "$$tmp"; echo "downloaded file is not the RQL schema"; exit 1; }; \
	mv "$$tmp" $(RQL_SCHEMA); \
	chmod 644 $(RQL_SCHEMA)
	@git diff --stat -- $(RQL_SCHEMA)

# --- Code/doc generation from the OpenAPI spec -----------------------------

check-tools: ## Verify the generation toolchain is installed (reports all missing at once)
	@missing=0; \
	check() { command -v "$$1" >/dev/null 2>&1 || { printf "  missing: %-6s → %s\n" "$$1" "$$2"; missing=1; }; }; \
	check go  "https://go.dev/dl/"; \
	check npx "Node.js 18+ — https://nodejs.org (provides npx; the TS client and API docs run via npx)"; \
	if [ "$$missing" -ne 0 ]; then \
		echo "ERROR: install the tool(s) above, then re-run. (npx fetches swagger-typescript-api, @redocly/cli, and widdershins on first use.)"; exit 1; \
	fi; \
	echo "generation toolchain OK (go + node)"

generate: check-tools ## Generate every artifact from the spec into openapi/ (Go server, TS client, HTML, Markdown) + the explorer's copy of the client + docs/openapi/ symlinks
	@echo ">> bundle   → $(OPENAPI_GEN)"
	@npx --yes $(REDOCLY) bundle $(OPENAPI) -o $(OPENAPI_GEN) >/dev/null
	@echo ">> gen-go   → $(API_OUT)/openapi.gen.go"
	@go tool oapi-codegen -config $(API_OUT)/oapi-codegen.yaml $(OPENAPI_GEN)
	@echo ">> gen-ts   → $(API_OUT)/openapi.gen.ts"
	@npx --yes $(SWAGGER_TS) generate -p $(OPENAPI_GEN) -o $(API_OUT) -n openapi.gen.ts >/dev/null
	@echo ">> copy     → $(EXPLORER_CLIENT)"
	@cp $(API_OUT)/openapi.gen.ts $(EXPLORER_CLIENT)
	@echo ">> gen-html → $(API_OUT)/openapi.html"
	@npx --yes $(REDOCLY) build-docs $(OPENAPI_GEN) -o $(API_OUT)/openapi.html >/dev/null
	@echo ">> gen-md   → $(API_OUT)/openapi.md"
	@npx --yes $(WIDDERSHINS) $(OPENAPI_GEN) -o $(API_OUT)/openapi.md --summary --code >/dev/null
	@echo ">> link     → docs/openapi/openapi.{html,md}"
	@mkdir -p docs/openapi
	@ln -sf ../../$(API_OUT)/openapi.html docs/openapi/openapi.html
	@ln -sf ../../$(API_OUT)/openapi.md docs/openapi/openapi.md

# --- Build / test ----------------------------------------------------------

tidy: ## Sync go.mod/go.sum with imports (adds transitive deps)
	@go mod tidy

build: ## Compile both binaries into bin/ (the server and the seeding client)
	@echo ">> build → $(BIN)"
	@go build -o $(BIN) ./cmd/ranke-db
	@echo ">> build → $(GEN)"
	@go build -o $(GEN) ./cmd/generator

DEV_CONFIG ?= examples/minimal/config.json
SEED_URL   ?= http://localhost:8080
# Shape knobs for SEED=chain; ignored by the example shape.
CONTRIBUTIONS ?= 20
CLAIMS        ?= 10

# SEED picks a shape: example (a handful of claims over three branches), release (the release
# process the slides draw — four signing identities, two packages meeting at one artifact),
# chain (as many contributions as CONTRIBUTIONS × CLAIMS asks for), or big — a chain sized past
# the explorer's lens threshold, so the timeline's stretching is exercised on a graph that
# needs it.
BIG_CONTRIBUTIONS ?= 1200
BIG_CLAIMS        ?= 20
RELEASES          ?= 3

SEED_ARGS = $(strip $(if $(filter big,$(SEED)), \
	chain --contributions $(BIG_CONTRIBUTIONS) --claims $(BIG_CLAIMS), \
	$(if $(filter chain,$(SEED)),chain --contributions $(CONTRIBUTIONS) --claims $(CLAIMS), \
	$(if $(filter release,$(SEED)),release --releases $(RELEASES),example))))

# The signing key is minted per run and never written down: this stack keeps
# nothing between launches, so a throwaway identity is the honest default. The
# address comes from the config's endpoints section, not a flag.
#
# Seeding runs as a client, which is what a contributor is — so the generator goes into
# the background with --wait, contributes as soon as /health answers, and exits, while
# the server keeps the foreground and ctrl-c.
dev: build ## Run a dev server from DEV_CONFIG (SEED=example|release|chain|big to seed it once it answers)
	@command -v openssl >/dev/null 2>&1 || { echo "ERROR: dev needs openssl to mint a throwaway signing key"; exit 1; }
	@addr=$$(grep -o '"addr"[[:space:]]*:[[:space:]]*"[^"]*"' $(DEV_CONFIG) | head -1 | sed -E 's/.*"([^"]*)"$$/\1/'); \
		url="http://localhost$${addr:-:8080}"; \
		echo ">> $(DEV_CONFIG) — ephemeral signing key, nothing persisted between runs"; \
		echo ">> serving on  $$url"; \
		echo ">> try:  curl $$url/health  ·  curl $$url/branches  ·  curl $$url/branches/main/head"; \
		echo ">> ctrl-c to stop"; \
		$(if $(SEED),$(GEN) $(SEED_ARGS) "$$url" --wait 15s &,) \
		RANKE_SIGNER_KEY="$$(openssl genpkey -algorithm ed25519)" $(BIN) run --dev $(DEV_CONFIG)

# Seeding a server that is already up. An in-memory stack dies with its process, so
# against the default config this only reaches an instance started by `make dev`.
seed: build ## Seed a running server over its REST API (SEED_URL, SEED=example|release|chain|big)
	@$(GEN) $(SEED_ARGS) $(SEED_URL) --wait 5s

smoke: build ## Launch against the minimal example, seed it over the API, read it back, shut down
	@./scripts/smoke.sh $(BIN) $(GEN)

test: ## Test all packages; scope with make test/<pkg> (e.g. test/config, test/adapters/vault/openbao)
	@echo ">> go test ./..."
	@go test ./...

# make test/<pkg-path> runs one package subtree, verbose — e.g. make test/adapters
# (all adapters), make test/adapters/vault/openbao (one impl), make test/config.
# One pattern rule routes any repo-relative path ($* captures the rest).
test/%:
	@echo ">> go test -v ./$*/..."
	@go test -v ./$*/...

verify: generate ## Regenerate from the spec, then build, vet, test, gofmt-check, and lint the module
	@set -e; \
		go build ./...; \
		go vet ./...; \
		fmt=$$(gofmt -l $$(git ls-files '*.go' | grep -v '\.gen\.go$$')); \
		[ -z "$$fmt" ] || { echo "gofmt needed:"; echo "$$fmt"; exit 1; }; \
		go test ./...; \
		if command -v brokkr >/dev/null 2>&1; then brokkr lint; \
		elif command -v sindri >/dev/null 2>&1; then sindri lint; \
		else echo ">> lint: neither brokkr nor sindri on PATH; skipping" >&2; fi
	@$(MAKE) -s ranke-go-version
	@$(MAKE) -s ranke-ts-version

upgrade: ## Upgrade all deps, tools and ranke-go to latest, tidy, then verify; asks before raising the go directive (GO_VERSION=keep|1.26.5, RANKE_GO_VERSION=vX.Y.Z)
	@GO_VERSION=$(GO_VERSION) \
		RANKE_GO_MOD=$(RANKE_GO_MOD) \
		RANKE_GO_VERSION=$(RANKE_GO_VERSION) \
		./scripts/upgrade.sh

ranke-ts-version: ## Recommend a ranke-ts bump if a newer release exists
	-@[ -f $(FRONTEND_PKG) ] && { \
		cur=$$(sed -n 's|.*"$(RANKE_TS_PKG)": *"[^0-9]*\([^"]*\)".*|\1|p' $(FRONTEND_PKG)); \
		latest=$$(npm view $(RANKE_TS_PKG) version 2>/dev/null); \
		if [ -n "$$cur" ] && [ -n "$$latest" ] && [ "$$cur" != "$$latest" ]; then \
			echo ">> ranke-ts: on $$cur, latest is $$latest — bump: make -C frontend upgrade"; \
		fi; \
	} || true

ranke-go-version: ## Recommend a ranke-go bump if a newer release exists
	-@grep -q "$(RANKE_GO_MOD)" go.mod 2>/dev/null && { \
		cur=$$(go list -m -f '{{.Version}}' $(RANKE_GO_MOD) 2>/dev/null); \
		latest=$$(go list -m -f '{{.Version}}' $(RANKE_GO_MOD)@latest 2>/dev/null); \
		if [ -n "$$latest" ] && [ "$$cur" != "$$latest" ]; then \
			echo ">> ranke-go: on $$cur, latest is $$latest — bump: go get $(RANKE_GO_MOD)@latest"; \
		fi; \
	} || true

# --- Release / papers (unchanged) ------------------------------------------

# --- Release gate ----------------------------------------------------------
#
# The artifacts can lie in two silent ways: the RQL schema moved in ranke-graph, or
# someone edited the spec here and never regenerated. Either ships a
# contract the code does not implement, so releasing checks both and refuses.

check-rql-schema: ## Fail if the vendored RQL schema differs from the ranke-graph spec
	@command -v curl > /dev/null || { echo "curl not found"; exit 1; }
	@echo ">> check    → $(RQL_SCHEMA) against the spec"
	@tmp=$$(mktemp); \
	curl -fsSL "$(RQL_SCHEMA_URL)" -o "$$tmp" \
		|| { rm -f "$$tmp"; echo "   cannot reach $(RQL_SCHEMA_URL)"; exit 1; }; \
	if diff -q "$$tmp" $(RQL_SCHEMA) > /dev/null 2>&1; then \
		rm -f "$$tmp"; echo "   matches the spec"; \
	else \
		echo ""; diff -u $(RQL_SCHEMA) "$$tmp" | head -40; rm -f "$$tmp"; \
		echo ""; \
		echo "   the RQL schema moved in ranke-graph — the source of truth is ahead of this copy."; \
		echo "   Take it:  make pull-rql-schema && make generate"; \
		exit 1; \
	fi

# Asks whether regenerating changes anything, rather than whether the tree matches
# HEAD — so uncommitted work in hand is not mistaken for drift. `generate` is
# byte-idempotent, so a changed hash means the artifacts were stale.
GEN_ARTIFACTS := $(OPENAPI_GEN) $(API_OUT)/openapi.gen.go $(API_OUT)/openapi.gen.ts \
                 $(API_OUT)/openapi.html $(API_OUT)/openapi.md $(EXPLORER_CLIENT)

check-generated: ## Fail if `make generate` would change anything (spec edited without regenerating); auto-commits the rebuild
	@sums=$$(mktemp); \
	md5sum $(GEN_ARTIFACTS) > "$$sums" 2>/dev/null || true; \
	$(MAKE) --no-print-directory generate; \
	if md5sum --status -c "$$sums" 2>/dev/null; then \
		rm -f "$$sums"; echo ">> check    → generated artifacts are current"; \
	else \
		rm -f "$$sums"; echo ""; \
		echo "   the artifacts were stale — the spec changed without a regenerate."; \
		echo "   Committing the rebuild (only these paths, nothing else in the tree):"; \
		git status --short -- $(GEN_ARTIFACTS) | sed 's/^/     /'; \
		git commit --quiet -m "chore: regenerate stale OpenAPI artifacts" -- $(GEN_ARTIFACTS); \
		echo "   committed — re-run to confirm the tree is now current."; \
		exit 1; \
	fi

release-gate: check-rql-schema check-generated ## Run the pre-release contract checks without releasing

release: release-gate ## Release: clean → merge to default via PR → tag merged tip → push (bump: major|minor|patch, aliases breaking|feature|fix)
	@./scripts/release.sh $(filter major minor patch breaking feature fix,$(MAKECMDGOALS))

major minor patch breaking feature fix:
	@:

docs: ## Pull the latest ranke-graph documents (papers, spec, glossary) into docs/papers/
	@echo ">> fetching ranke-graph documents into $(PAPERS_DIR)/"
	@tmp=$$(mktemp -d) && \
		git clone --depth 1 --branch $(RANKE_GRAPH_REF) $(RANKE_GRAPH_REPO) $$tmp >/dev/null 2>&1 && \
		rm -rf $(PAPERS_DIR) && mkdir -p $(PAPERS_DIR) && \
		for d in $$tmp/*/; do \
			name=$$(basename $$d); \
			case " $(PAPERS_SKIP) " in *" $$name "*) continue ;; esac; \
			cp -r $$d $(PAPERS_DIR)/; \
		done && \
		cp $$tmp/LICENSE $(PAPERS_DIR)/LICENSE 2>/dev/null || true; \
		rm -rf $$tmp; \
		echo ">> pulled $$(find $(PAPERS_DIR) -name '*.typ' | wc -l | tr -d ' ') document(s):"; \
		find $(PAPERS_DIR) -name '*.typ' | sort | sed 's|^|     |'

docs-clean: ## Remove the pulled paper references
	rm -rf $(PAPERS_DIR)
