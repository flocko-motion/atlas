# ranke-explorer — the browser client.
#
# `make` builds the distributable artifacts; `make dev` starts the dev server.
# Dependencies install themselves on first use, so a clean checkout needs nothing
# but node.

.PHONY: all dev build single bench bench-render check test clean help upgrade ranke-ts-version
.DEFAULT_GOAL := all

VITE  := node node_modules/vite/bin/vite.js
NODE  := node --expose-gc --max-old-space-size=4096

all: build single ## Default: build dist/ and the self-contained explorer.html

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-13s\033[0m %s\n", $$1, $$2}'

# Dependencies. Stamped by the directory itself, so it reinstalls when the
# manifests change and stays quiet otherwise.
node_modules: package.json package-lock.json
	@echo ">> installing dependencies"
	@npm install --no-audit --no-fund
	@touch node_modules

dev: node_modules ## Start the dev server (open the URL it prints, then press 'load')
	@echo ">> sources live in the Server tab: a mock archive, or a ranke-db instance"
	@$(VITE) --host

build: node_modules ## Bundle to dist/ — the static artifact to serve
	@$(VITE) build

single: build ## Fold dist/ into one self-contained explorer.html
	@node scripts/inline.mjs

bench: node_modules ## Headless performance numbers (build, heap, layout, shape)
	@$(NODE) src/bench/graph-bench.ts $(ARGS)

bench-render: node_modules ## Frame timings in a real browser (needs a GPU; skips cleanly)
	@node src/bench/browser-bench.ts $(ARGS)

check: node_modules ## Type-check without emitting
	@node_modules/typescript/bin/tsc --noEmit

# Node's own runner, which strips the types — the same way the benches run. Core is
# headless, so its behaviour is testable without a DOM or a browser.
#
# Types are transformed rather than only stripped: the generated client declares an enum,
# which strip-only mode refuses outright, and core/data reaches the contract through it. The
# flag is experimental and warns on every run — the choice is deliberate, so the warning goes.
test: node_modules ## Run the headless core tests
	@node --experimental-transform-types --disable-warning=ExperimentalWarning --test 'src/**/*.test.ts'

clean: ## Remove build output (explorer.html is committed, so it stays)
	@rm -rf dist

# --- Dependencies ----------------------------------------------------------

# ranke-ts is the model this client is built on — the same claims, ids and taxonomy the
# server holds through ranke-go — so it is pinned exactly and bumped deliberately, the way
# the module is next door (see ../scripts/upgrade.sh).
#
# It is fetched LAST: `npm update` has already moved everything within its range, so the
# final install is what makes an explicit pin stick.
#
#   make upgrade                       # everything, ranke-ts to latest
#   RANKE_TS_VERSION=0.5.0 make upgrade  # take that version instead
#
# Revert with: git checkout package.json package-lock.json
RANKE_TS         := @flocko-motion/ranke
RANKE_TS_VERSION ?= latest

upgrade: node_modules ## Upgrade every dependency and ranke-ts, then check, test and build (RANKE_TS_VERSION=x.y.z)
	@echo ">> dependencies: npm update"
	@npm update --save --no-audit --no-fund
	@echo ">> $(RANKE_TS): fetching $(RANKE_TS_VERSION)"
	@npm install --save-exact --no-audit --no-fund $(RANKE_TS)@$(RANKE_TS_VERSION)
	@echo ">> package.json changes:"
	@git diff -U0 -- package.json | grep -E '^[-+][^-+]' | sed 's/^/     /' || echo "     (none)"
	@echo ">> verifying the upgrade"
	@$(MAKE) --no-print-directory check
	@$(MAKE) --no-print-directory test
	@$(MAKE) --no-print-directory build
	@echo ">> upgrade OK — review the diff, then commit package.json/package-lock.json"

ranke-ts-version: node_modules ## Report whether a newer ranke-ts release exists
	@cur=$$(node -p "require('./package.json').dependencies['$(RANKE_TS)']"); \
	latest=$$(npm view $(RANKE_TS) version 2>/dev/null); \
	if [ -n "$$latest" ] && [ "$$cur" != "$$latest" ]; then \
		echo ">> ranke-ts: on $$cur, latest is $$latest — bump: make upgrade"; \
	else \
		echo ">> ranke-ts: on $$cur, which is the latest"; \
	fi
