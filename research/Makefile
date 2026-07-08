# openCypher TCK conformance harness for mstrYoda/goraphdb
#
# The harness runs the openCypher Technology Compatibility Kit against goraphdb
# and prints a conformance summary plus a per-scenario CSV.
#
# The TCK feature files are NOT vendored — they are fetched on demand from the
# openCypher repo into tck/features/ (gitignored). Any target that needs them
# triggers the download automatically.

GO       ?= go
TIMEOUT  ?= 600s
TCK_REPO ?= https://github.com/opencypher/openCypher
TCK_REF  ?= master
# Narrow the run to a subset, e.g.: make test SUBSET=features/clauses/create
SUBSET   ?=

FEATURES := tck/features

.PHONY: help test smoke conformance report subset features clean distclean tidy

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

# Fetch the TCK feature files via a blobless sparse clone (~2MB of .feature
# files, not the whole repo). This is an order-only prerequisite below, so it
# runs once and is skipped while tck/features/ exists.
$(FEATURES):
	@echo ">> openCypher TCK not present — fetching into $(FEATURES)/"
	@tmp=$$(mktemp -d) && \
		git clone --depth 1 --filter=blob:none --sparse --branch $(TCK_REF) $(TCK_REPO) $$tmp >/dev/null 2>&1 && \
		git -C $$tmp sparse-checkout set tck/features >/dev/null 2>&1 && \
		mkdir -p tck && cp -r $$tmp/tck/features $(FEATURES) && \
		cp $$tmp/LICENSE tck/LICENSE-openCypher 2>/dev/null || true; \
		rm -rf $$tmp; \
		echo ">> vendored $$(find $(FEATURES) -name '*.feature' | wc -l | tr -d ' ') feature files"

features: $(FEATURES) ## Download the openCypher TCK feature files if absent

smoke: ## Sanity-check that goraphdb opens and runs basic queries (no TCK needed)
	cd $(CURDIR) && TCK_FEATURES= $(GO) test ./adapter/ -run TestSmoke -v

test: | $(FEATURES) ## Run the FULL TCK suite (verbose) and print the conformance report
	cd $(CURDIR) && TCK_FEATURES="$(SUBSET)" $(GO) test ./tck/ -run TestTCK -v -timeout $(TIMEOUT)

conformance: | $(FEATURES) ## Run the full suite quietly; print only the conformance report
	@cd $(CURDIR) && $(GO) test ./tck/ -run TestTCK -v -timeout $(TIMEOUT) 2>&1 \
		| sed -E 's/\x1b\[[0-9;]*m//g' \
		| sed -n '/openCypher TCK/,/=====$$/p'

subset: | $(FEATURES) ## Run a subset: make subset SUBSET=features/clauses/create
	@if [ -z "$(SUBSET)" ]; then echo "usage: make subset SUBSET=features/<path>"; exit 2; fi
	cd $(CURDIR) && TCK_FEATURES="$(SUBSET)" $(GO) test ./tck/ -run TestTCK -v -timeout $(TIMEOUT)

report: ## Summarise the last run from tck/tck-results.csv
	@test -f tck/tck-results.csv || { echo "no results yet — run 'make test' first"; exit 1; }
	@echo "Outcome breakdown (n = $$(tail -n +2 tck/tck-results.csv | wc -l | tr -d ' ')):"
	@tail -n +2 tck/tck-results.csv | cut -d, -f1 | sort | uniq -c | sort -rn

tidy: ## Tidy module dependencies
	cd $(CURDIR) && $(GO) mod tidy

clean: ## Remove generated run artifacts (keeps the downloaded TCK)
	rm -f tck/tck-results.csv

distclean: clean ## Also remove the downloaded TCK feature files
	rm -rf $(FEATURES) tck/LICENSE-openCypher
