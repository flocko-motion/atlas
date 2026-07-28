# ranke-explorer — the browser client.
#
# `make` builds the distributable artifacts; `make run` starts the dev server.
# Dependencies install themselves on first use, so a clean checkout needs nothing
# but node.

.PHONY: all run build single bench bench-render check clean help
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

run: node_modules ## Start the dev server (open the URL it prints, then press 'load')
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

clean: ## Remove build output (explorer.html is committed, so it stays)
	@rm -rf dist
