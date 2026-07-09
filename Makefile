# ranke-db — repo-level targets.
#
# The OpenAPI spec is the single source of truth; `make generate` produces every
# artifact from it into api/ — the Go server, the TS/JS client, and the HTML +
# Markdown references — and symlinks the two references under docs/api/.

OPENAPI   := openapi/openapi.yaml
API_OUT   := api
RANKE_GO_MOD ?= github.com/flocko-motion/ranke-go

RANKE_GRAPH_REPO ?= https://github.com/flocko-motion/ranke-graph
RANKE_GRAPH_REF  ?= main
PAPERS_DIR       := docs/papers

.PHONY: all help check-tools generate verify tidy build smoke test \
        ranke-go-version release major minor patch breaking feature fix docs docs-clean

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

generate: check-tools ## Generate every artifact from the spec into api/ (Go server, TS client, HTML, Markdown) + docs/api/ symlinks
	@echo ">> gen-go   → $(API_OUT)/openapi.gen.go"
	@go tool oapi-codegen -config $(API_OUT)/oapi-codegen.yaml $(OPENAPI)
	@echo ">> gen-ts   → $(API_OUT)/openapi.gen.ts"
	@npx --yes swagger-typescript-api@13 generate -p $(OPENAPI) -o $(API_OUT) -n openapi.gen.ts >/dev/null
	@echo ">> gen-html → $(API_OUT)/openapi.html"
	@npx --yes @redocly/cli@latest build-docs $(OPENAPI) -o $(API_OUT)/openapi.html >/dev/null
	@echo ">> gen-md   → $(API_OUT)/openapi.md"
	@npx --yes widdershins@4 $(OPENAPI) -o $(API_OUT)/openapi.md --summary --code >/dev/null
	@echo ">> link     → docs/api/openapi.{html,md}"
	@mkdir -p docs/api
	@ln -sf ../../$(API_OUT)/openapi.html docs/api/openapi.html
	@ln -sf ../../$(API_OUT)/openapi.md docs/api/openapi.md

# --- Build / test ----------------------------------------------------------

tidy: ## Sync go.mod/go.sum with imports (adds transitive deps)
	@go mod tidy

build: ## Compile the ranke-db binary to bin/
	@echo ">> build → $(BIN)"
	@go build -o $(BIN) ./cmd/ranke-db

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

docs: ## Pull the latest ranke-graph papers into docs/papers/ for reference
	@echo ">> fetching ranke-graph papers into $(PAPERS_DIR)/"
	@tmp=$$(mktemp -d) && \
		git clone --depth 1 --branch $(RANKE_GRAPH_REF) $(RANKE_GRAPH_REPO) $$tmp >/dev/null 2>&1 && \
		rm -rf $(PAPERS_DIR) && mkdir -p $(PAPERS_DIR) && \
		cp -r $$tmp/[0-9]*-* $(PAPERS_DIR)/ && \
		{ [ -d $$tmp/shared ] && cp -r $$tmp/shared $(PAPERS_DIR)/ || true; } && \
		cp $$tmp/LICENSE $(PAPERS_DIR)/LICENSE 2>/dev/null || true; \
		rm -rf $$tmp; \
		echo ">> pulled $$(find $(PAPERS_DIR) -name '*.typ' | wc -l | tr -d ' ') paper(s)"

docs-clean: ## Remove the pulled paper references
	rm -rf $(PAPERS_DIR)
