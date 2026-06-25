# RankeDB — repo-level convenience targets.
#
# The implementation lives under server/ (schemaf project); see README.md.
# This Makefile holds repo-wide helpers that aren't part of schemaf.

RANKE_GRAPH_REPO ?= https://github.com/flocko-motion/ranke-graph
RANKE_GRAPH_REF  ?= main
PAPERS_DIR       := docs/papers

.PHONY: help verify ranke-go-version release major minor patch breaking feature fix docs docs-clean

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

# Scoped Go check for the new ranke adapter/control-plane packages
# (adapter/* + seal). Deliberately NOT the full `schemaf.sh test`: no codegen,
# no TS, no stale api/db tests — just build/vet/fmt/test of what we're adding.
# Widen VERIFY_PKGS as more lands; fold into schemaf.sh test once the rest
# of the server is caught up.
VERIFY_PKGS ?= . ./adapter/... ./seal/... ./access/... ./assembler/... ./core/... ./api/...

# Generated Go files are owned by their generator (schemaf codegen), never
# hand-maintained — so the brokkr linters skip them. The Go linters only scan
# .go, hence the two patterns: source (*.gen.go) and generated tests (*.gen_test.go).
BROKKR_IGNORE ?= --ignore='*.gen.go' --ignore='*.gen_test.go'

RANKE_GO_MOD ?= github.com/flocko-motion/ranke-go

verify: ## Build/vet/fmt/test the new ranke packages + brokkr lint (no codegen)
	@cd server/go && set -e; \
		go build $(VERIFY_PKGS); \
		go vet $(VERIFY_PKGS); \
		fmt=$$(gofmt -l main.go adapter seal access assembler core api); [ -z "$$fmt" ] || { echo "gofmt needed:"; echo "$$fmt"; exit 1; }; \
		go test $(VERIFY_PKGS); \
		brokkr lint $(BROKKR_IGNORE)
	@$(MAKE) -s ranke-go-version

# Advisory (non-fatal, best-effort): once ranke-go is a dependency, recommend a
# bump when a newer release exists. Policy: track latest, bump when necessary.
# Inert until ranke-go is required; tolerant of being offline.
ranke-go-version: ## Recommend a ranke-go bump if a newer release exists
	-@cd server/go && grep -q "$(RANKE_GO_MOD)" go.mod 2>/dev/null && { \
		cur=$$(go list -m -f '{{.Version}}' $(RANKE_GO_MOD) 2>/dev/null); \
		latest=$$(go list -m -f '{{.Version}}' $(RANKE_GO_MOD)@latest 2>/dev/null); \
		if [ -n "$$latest" ] && [ "$$cur" != "$$latest" ]; then \
			echo ">> ranke-go: on $$cur, latest is $$latest — bump: go get $(RANKE_GO_MOD)@latest"; \
		fi; \
	} || true

release: ## Release: clean → merge to default via PR → tag merged tip → push (bump: major|minor|patch, aliases breaking|feature|fix)
	@./scripts/release.sh $(filter major minor patch breaking feature fix,$(MAKECMDGOALS))

# Absorb the positional bump word in `make release <bump>` so it isn't treated
# as a missing target. No ## doc → stays out of `make help`.
major minor patch breaking feature fix:
	@:

# Pull fresh reference copies of the ranke-graph papers (Typst sources) into
# docs/papers/. These are read-only references, regenerable any time, and
# gitignored — never edit them here; edit them in the ranke-graph repo.
docs: ## Pull the latest ranke-graph papers into docs/papers/ for reference
	@echo ">> fetching ranke-graph papers into $(PAPERS_DIR)/"
	@tmp=$$(mktemp -d) && \
		git clone --depth 1 --branch $(RANKE_GRAPH_REF) $(RANKE_GRAPH_REPO) $$tmp >/dev/null 2>&1 && \
		rm -rf $(PAPERS_DIR) && mkdir -p $(PAPERS_DIR) && \
		cp -r $$tmp/[0-9]*-* $(PAPERS_DIR)/ && \
		{ [ -d $$tmp/shared ] && cp -r $$tmp/shared $(PAPERS_DIR)/ || true; } && \
		cp $$tmp/LICENSE $(PAPERS_DIR)/LICENSE 2>/dev/null || true; \
		rm -rf $$tmp; \
		echo ">> pulled $$(find $(PAPERS_DIR) -name '*.typ' | wc -l | tr -d ' ') paper(s) into $(PAPERS_DIR)/ ($(RANKE_GRAPH_REF))"

docs-clean: ## Remove the pulled paper references
	rm -rf $(PAPERS_DIR)
