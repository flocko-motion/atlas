# RankeDB — repo-level convenience targets.
#
# The implementation lives under server/ (schemaf project); see README.md.
# This Makefile holds repo-wide helpers that aren't part of schemaf.

RANKE_GRAPH_REPO ?= https://github.com/flocko-motion/ranke-graph
RANKE_GRAPH_REF  ?= main
PAPERS_DIR       := docs/papers

.PHONY: help docs docs-clean

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

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
