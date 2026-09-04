# PiCode — make targets
# Quality gates are the contract (AGENTS.md); `make ci` mirrors GitHub Actions.

.PHONY: help hooks hooks-check dev ui web docs docs-videos docs-videos-check docs-videos-fresh build restart deploy install test test-js fmt fmt-check vet ci-docs ci clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "} {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

# Lockfiles and integrity hashes still decide the exact dependency graph; these
# flags avoid advisory/funding network calls and prefer setup-node's warm cache.
NPM_CI_FLAGS ?= --prefer-offline --no-audit --no-fund

# .git/hooks is not versioned, so the guards live in .githooks and every
# target that a human or an agent runs first points git at them. Idempotent
# and silent once set; runs from any worktree (config is per clone).
hooks: ## Point git at the repo's hooks (.githooks — keeps the root on main)
	@./scripts/hooks-enable.sh

hooks-check: hooks ## Prove the guards work (policy matrix on a throwaway repo)
	./scripts/hooks-selftest.sh

dev: hooks ## Run the Go server (HTTPS, port 8445+; serves last `make web` build)
	go run ./cmd/picode

ui: ## Vite HMR on :5173 (proxies /api and /ws to https://localhost:8445)
	cd web && npm run dev

# npm ci wipes and reinstalls, so it is gated on the lockfile rather than run
# by every target that needs node_modules. The stamp is the manifest npm writes
# on a successful install, not the directory: an empty node_modules with a
# fresh timestamp satisfies make and then the build dies on `vite: not found`,
# which is exactly what happened once.
NODE_STAMP := web/node_modules/.package-lock.json

$(NODE_STAMP): web/package-lock.json
	cd web && npm ci $(NPM_CI_FLAGS)
	@touch $(NODE_STAMP)

web: $(NODE_STAMP) ## Build the React UI into internal/web/public (ADR-0008)
	cd web && npm run build

WWW_STAMP := www/node_modules/.package-lock.json

$(WWW_STAMP): www/package-lock.json
	cd www && npm ci $(NPM_CI_FLAGS)
	@touch $(WWW_STAMP)

docs: openapi llms $(WWW_STAMP) ## Build the VitePress public site (GitHub Pages)
	cd www && npm run build

openapi: ## Generate the OpenAPI spec from the server's route registration
	mkdir -p www/public/api
	go run ./cmd/picode-openapi > www/public/api/openapi.json

llms: ## Generate llms.txt (machine-readable map of the docs site)
	node scripts/docs-llms.mjs

fixture: ## Run the docs fixture daemon (synthetic seeded UI, 127.0.0.1:18740)
	go run ./cmd/picode-docs-fixture

# Parity principle (docs/benchmarks/2026-09-03-docs-harness.md): the site's
# images are generated from the current UI, never hand-placed. UI change ⇒
# re-run docs-shots, or docs-check fails.
docs-shots: web ## Capture the current UI into www/img (needs agent-browser on PATH)
	go build -o bin/picode-docs-fixture ./cmd/picode-docs-fixture
	fuser -k 18740/tcp 2>/dev/null || true
	./bin/picode-docs-fixture & pid=$$!; trap 'kill $$pid 2>/dev/null' EXIT; \
		sleep 3; node scripts/docs-shots.mjs


VALE_VERSION ?= 3.12.0
VALE := bin/vale

$(VALE): ## Pinned Vale binary (prose linter), downloaded once into bin/
	@mkdir -p bin
	@asset=""; case "$$(uname -s)/$$(uname -m)" in \
		Linux/x86_64) asset="vale_$(VALE_VERSION)_Linux_64-bit" ;; \
		Linux/aarch64) asset="vale_$(VALE_VERSION)_Linux_arm64" ;; \
		Darwin/x86_64) asset="vale_$(VALE_VERSION)_macOS_64-bit" ;; \
		Darwin/arm64) asset="vale_$(VALE_VERSION)_macOS_arm64" ;; \
		*) echo "unsupported platform for vale: $$(uname -s)/$$(uname -m)"; exit 1 ;; \
	esac; \
	curl -fsSL "https://github.com/errata-ai/vale/releases/download/v$(VALE_VERSION)/$$asset.tar.gz" | tar -xz -C bin vale
	@chmod +x $(VALE)

vale: $(VALE) ## Prose lint on the public docs (spelling + repetition; error gate)
	$(VALE) --config=.vale.ini --minAlertLevel=error www/*.md www/guide/*.md
docs-videos: ## Capture stills + render the three docs tutorial videos into www/public/video (needs agent-browser)
	go build -o bin/picode-docs-fixture ./cmd/picode-docs-fixture
	fuser -k 18740/tcp 2>/dev/null || true
	./bin/picode-docs-fixture & pid=$$!; trap 'kill $$pid 2>/dev/null' EXIT; \
		sleep 3; node scripts/docs-video-stills.mjs
	cd docs-videos && npx hyperframes@0.8.27 render --composition index.html --quality high --output renders/create-agent.mp4 --quiet
	cd docs-videos && npx hyperframes@0.8.27 render --composition compositions/automate-it.html --quality high --output renders/automate-it.mp4 --quiet
	cd docs-videos && npx hyperframes@0.8.27 render --composition compositions/take-it-anywhere.html --quality high --output renders/take-it-anywhere.mp4 --quiet
	node scripts/docs-video-manifest.mjs
docs-videos-check: ## Fast integrity check for committed video inputs and MP4s (no capture/render)
	node scripts/docs-video-manifest.mjs --check
docs-videos-fresh: ## Strict manual audit: report tutorials whose captured UI surfaces changed
	node scripts/docs-video-manifest.mjs --fresh
docs-check: ## CI parity: current images/generated docs plus video integrity (no capture/render)
	node scripts/docs-check.mjs

cert: ## Provision/renew the mkcert TLS certificate (scripts/setup-cert.sh)
	./scripts/setup-cert.sh

install: build ## Copy bin/picode to ~/.local/bin and enable systemd --user
	./bin/picode install

deploy: build ## Rebuild UI+binary and restart the installed service
	./bin/picode deploy

build: web ## Build UI + bin/picode (embeds the UI — ADR-0023)
	go build -tags embedui -o bin/picode ./cmd/picode

desktop: ## Cross-compile the Windows tray + console native host (ADR-0020 / 0043)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
		go build -ldflags "-H=windowsgui -s -w" -o bin/picode-desktop.exe ./cmd/picode-desktop
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
		go build -ldflags "-s -w" -o bin/picode-nmh.exe ./cmd/picode-desktop

desktop-restart: desktop ## Swap the Windows exes and relaunch the tray via the logon task (NEVER `&` from WSL — scripts/desktop-swap.sh)
	./scripts/desktop-swap.sh

restart: deploy ## Rebuild and restart the systemd service (`picode deploy`)

test: ## Run all Go tests
	go test ./...

test-js: $(NODE_STAMP) ## Run the frontend unit tests and the pi package suites
	cd web && npm test
	node --test scripts/*.test.mjs
	node --test packages/pi-roles/test/*.test.ts
	node --test packages/pi-inbox/test/*.test.ts
	node --test packages/pi-checklist/test/*.test.ts
	node --test packages/pi-sysadmin/test/*.test.ts
	npm install --prefix packages/pi-compact --no-audit --no-fund
	npm test --prefix packages/pi-compact

# Both targets walk the package directories `go list` reports, not the tree.
# `.` reaches into .worktrees/, where a sibling agent has its own checkout: fmt
# would rewrite their uncommitted files and fmt-check would fail this gate on
# their code. A worktree carries its own go.mod, so ./... already excludes it —
# vet and test were always safe; these two were not.
fmt: ## Format all Go code
	@dirs=$$(go list -f '{{.Dir}}' ./...) || exit 1; \
	gofmt -w $$dirs

fmt-check: ## Fail if any file is unformatted
	@dirs=$$(go list -f '{{.Dir}}' ./...) || exit 1; \
	out=$$(gofmt -l $$dirs); \
	if [ -n "$$out" ]; then echo "gofmt needed on:"; echo "$$out"; exit 1; fi

vet: ## Static analysis
	go vet ./...

# Keep parity ahead of generation: `docs` rewrites OpenAPI/llms.txt, so checking
# afterward would accidentally bless stale committed artifacts.
ci-docs: ## Verify committed docs parity, then build the public site
	$(MAKE) docs-check
	$(MAKE) docs

ci: hooks-check fmt-check vet test test-js build ci-docs vale ## Everything CI runs (includes UI + public docs + image + prose parity)

clean: ## Remove build artifacts
	rm -rf bin/ web/node_modules/ www/node_modules/ www/.vitepress/dist
