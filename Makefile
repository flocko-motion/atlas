# ranke-db — repo-level targets.
#
# The OpenAPI spec is the single source of truth; `make generate` produces every
# artifact from it into openapi/ (alongside the spec) — the Go server, the TS/JS
# client, and the HTML + Markdown references — and symlinks the two references
# under docs/openapi/.

OPENAPI   := openapi/openapi.yaml
API_OUT   := openapi
RANKE_GO_MOD ?= github.com/flocko-motion/ranke-go
RANKE_GO_VERSION ?= latest
# ask = prompt before raising the go directive; keep = leave it; or a version.
GO_VERSION ?= ask

RANKE_GRAPH_REPO ?= https://github.com/flocko-motion/ranke-graph
RANKE_GRAPH_REF  ?= main
PAPERS_DIR       := docs/papers
# Everything at the top of the paper repo is reference material and gets pulled;
# these are the exceptions (its own tooling). Dotdirs never match the glob.
PAPERS_SKIP      := scripts

.PHONY: all help check-tools generate verify tidy build smoke test dev \
        ranke-go-version upgrade release major minor patch breaking feature fix docs docs-clean

BIN := bin/ranke-db

.DEFAULT_GOAL := all

all: generate verify ## Default: regenerate from the spec, then build/vet/test

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

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

generate: check-tools ## Generate every artifact from the spec into openapi/ (Go server, TS client, HTML, Markdown) + docs/openapi/ symlinks
	@echo ">> gen-go   → $(API_OUT)/openapi.gen.go"
	@go tool oapi-codegen -config $(API_OUT)/oapi-codegen.yaml $(OPENAPI)
	@echo ">> gen-ts   → $(API_OUT)/openapi.gen.ts"
	@npx --yes swagger-typescript-api@13 generate -p $(OPENAPI) -o $(API_OUT) -n openapi.gen.ts >/dev/null
	@echo ">> gen-html → $(API_OUT)/openapi.html"
	@npx --yes @redocly/cli@latest build-docs $(OPENAPI) -o $(API_OUT)/openapi.html >/dev/null
	@echo ">> gen-md   → $(API_OUT)/openapi.md"
	@npx --yes widdershins@4 $(OPENAPI) -o $(API_OUT)/openapi.md --summary --code >/dev/null
	@echo ">> link     → docs/openapi/openapi.{html,md}"
	@mkdir -p docs/openapi
	@ln -sf ../../$(API_OUT)/openapi.html docs/openapi/openapi.html
	@ln -sf ../../$(API_OUT)/openapi.md docs/openapi/openapi.md

# --- Build / test ----------------------------------------------------------

tidy: ## Sync go.mod/go.sum with imports (adds transitive deps)
	@go mod tidy

build: ## Compile the ranke-db binary to bin/
	@echo ">> build → $(BIN)"
	@go build -o $(BIN) ./cmd/ranke-db

DEV_CONFIG ?= examples/minimal/config.json

# The signing key is minted per run and never written down: this stack keeps
# nothing between launches, so a throwaway identity is the honest default. The
# address comes from the config's endpoints section, not a flag.
dev: build ## Run a dev server from DEV_CONFIG with an ephemeral signing key
	@command -v openssl >/dev/null 2>&1 || { echo "ERROR: dev needs openssl to mint a throwaway signing key"; exit 1; }
	@addr=$$(grep -o '"addr"[[:space:]]*:[[:space:]]*"[^"]*"' $(DEV_CONFIG) | head -1 | sed -E 's/.*"([^"]*)"$$/\1/'); \
		echo ">> $(DEV_CONFIG) — ephemeral signing key, nothing persisted between runs"; \
		echo ">> serving on  http://localhost$${addr:-:8080}"; \
		echo ">> routes are mounted (/health, /system/layers, /{branch}/head, POST /query, POST /contribute)"; \
		echo ">> but every handler answers 501 core: not implemented — the core is still being built"; \
		echo ">> ctrl-c to stop"; \
		RANKE_SIGNER_KEY="$$(openssl genpkey -algorithm ed25519)" $(BIN) run $(DEV_CONFIG)

smoke: build ## Launch ranke-db run against the minimal example, check health, shut down
	@./scripts/smoke.sh $(BIN)

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

upgrade: ## Upgrade all deps, tools and ranke-go to latest, tidy, then verify; asks before raising the go directive (GO_VERSION=keep|1.26.5, RANKE_GO_VERSION=vX.Y.Z)
	@GO_VERSION=$(GO_VERSION) \
		RANKE_GO_MOD=$(RANKE_GO_MOD) \
		RANKE_GO_VERSION=$(RANKE_GO_VERSION) \
		./scripts/upgrade.sh

ranke-go-version: ## Recommend a ranke-go bump if a newer release exists
	-@grep -q "$(RANKE_GO_MOD)" go.mod 2>/dev/null && { \
		cur=$$(go list -m -f '{{.Version}}' $(RANKE_GO_MOD) 2>/dev/null); \
		latest=$$(go list -m -f '{{.Version}}' $(RANKE_GO_MOD)@latest 2>/dev/null); \
		if [ -n "$$latest" ] && [ "$$cur" != "$$latest" ]; then \
			echo ">> ranke-go: on $$cur, latest is $$latest — bump: go get $(RANKE_GO_MOD)@latest"; \
		fi; \
	} || true

# --- Release / papers (unchanged) ------------------------------------------

release: ## Release: clean → merge to default via PR → tag merged tip → push (bump: major|minor|patch, aliases breaking|feature|fix)
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
