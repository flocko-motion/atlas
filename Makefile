# ranke-db — repo-level targets.
#
# The OpenAPI spec is the single source of truth; `make generate` produces the
# Go server interface, the TS/JS client, and HTML + PDF reference docs from it.

OPENAPI   := openapi/openapi.yaml
API_DOCS  := docs/api
TS_OUT    := frontend/src/api/generated
RANKE_GO_MOD ?= github.com/flocko-motion/ranke-go

RANKE_GRAPH_REPO ?= https://github.com/flocko-motion/ranke-graph
RANKE_GRAPH_REF  ?= main
PAPERS_DIR       := docs/papers

.PHONY: all help check-tools generate gen-go gen-ts gen-html verify \
        ranke-go-version release major minor patch breaking feature fix docs docs-clean

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
	check npx "Node.js 18+ — https://nodejs.org (provides npx; the TS/HTML/PDF generators run via npx)"; \
	if [ "$$missing" -ne 0 ]; then \
		echo "ERROR: install the tool(s) above, then re-run. (npx fetches swagger-typescript-api + @redocly/cli on first use.)"; exit 1; \
	fi; \
	echo "generation toolchain OK (go + node)"

generate: check-tools gen-go gen-ts gen-html ## Generate Go server, TS client, and HTML docs from the spec

gen-go: ## OpenAPI → Go (strict net/http server interface + models)
	@echo ">> gen-go  → api/openapi.gen.go"
	@go tool oapi-codegen -config api/oapi-codegen.yaml $(OPENAPI)

gen-ts: ## OpenAPI → TypeScript/JS client
	@echo ">> gen-ts  → $(TS_OUT)/api.gen.ts"
	@mkdir -p $(TS_OUT)
	@npx --yes swagger-typescript-api@13 generate -p $(OPENAPI) -o $(TS_OUT) -n api.gen.ts >/dev/null

gen-html: ## OpenAPI → self-contained HTML reference
	@echo ">> gen-html → $(API_DOCS)/index.html"
	@mkdir -p $(API_DOCS)
	@npx --yes @redocly/cli@latest build-docs $(OPENAPI) -o $(API_DOCS)/index.html >/dev/null

# --- Build / test ----------------------------------------------------------

verify: ## Build, vet, test, and gofmt-check the module
	@set -e; \
		go build ./...; \
		go vet ./...; \
		fmt=$$(gofmt -l $$(git ls-files '*.go' | grep -v '\.gen\.go$$')); \
		[ -z "$$fmt" ] || { echo "gofmt needed:"; echo "$$fmt"; exit 1; }; \
		go test ./...
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
