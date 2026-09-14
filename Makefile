# PiCode — make targets
# Quality gates are the contract (AGENTS.md); `make ci` mirrors GitHub Actions.

.PHONY: help hooks hooks-check dev ui web docs docs-videos docs-videos-check docs-videos-fresh build restart deploy _deploy cert-timer changelog adr install test test-js fmt fmt-check vet ci-docs ci ci-gates ci-scoped close close-summary handoff worktree worktree-status worktree-gc clean desktop desktop-shell desktop-restart

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
	@printf 'dev: %s on %s\n' "$$(pwd)" "$$(git branch --show-current 2>/dev/null || echo '(no git)')"
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

# Rebuilt only when something under web/ changed (ADR-0105): the Vite build
# ran up to five times per branch — ci-scoped, close, docs-shots, make ci on
# main and deploy — for the same sources. The stamp lives in var/ (git-ignored)
# so the embedded UI never carries it; a missing build output resets it.
WEB_SRC := $(shell find web -type f -not -path '*/node_modules/*' -not -path '*/dist/*' 2>/dev/null)
WEB_STAMP := var/web.built

$(WEB_STAMP): $(NODE_STAMP) $(WEB_SRC)
	cd web && npm run build
	@mkdir -p var && touch $(WEB_STAMP)

web: ## Build launcher + desktop/mobile into internal/web/public when web/ changed (ADR-0072)
	@if [ ! -f internal/web/public/index.html ] && [ -f $(WEB_STAMP) ]; then touch -t 197001010000 $(WEB_STAMP); fi
	@$(MAKE) --no-print-directory $(WEB_STAMP)

DOCS_STAMP := docs-site/node_modules/.package-lock.json

$(DOCS_STAMP): docs-site/package-lock.json
	cd docs-site && npm ci $(NPM_CI_FLAGS)
	@touch $(DOCS_STAMP)

docs: openapi llms $(DOCS_STAMP) ## Build the VitePress public site (GitHub Pages)
	cd docs-site && npm run build

openapi: ## Generate the OpenAPI spec from the server's route registration
	mkdir -p docs-site/public/api
	go run ./cmd/picode-openapi > docs-site/public/api/openapi.json

llms: ## Generate llms.txt (machine-readable map of the docs site)
	node scripts/docs-llms.mjs

fixture: ## Run the docs fixture daemon (synthetic seeded UI, 127.0.0.1:18740)
	go run ./cmd/picode-docs-fixture

# Parity principle (docs/benchmarks/2026-09-03-docs-harness.md): the site's
# images are generated from the current UI, never hand-placed. UI change ⇒
# re-run docs-shots, or docs-check fails.
docs-shots: web ## Capture the current UI into docs-site/img (needs agent-browser on PATH)
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
	$(VALE) --config=.vale.ini --minAlertLevel=error docs-site/*.md docs-site/guide/*.md
docs-videos: ## Capture stills + render the three docs tutorial videos into docs-site/public/video (needs agent-browser)
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

# Deploy is the owner's call (ADR-0105): a branch session never deploys; the
# owner runs `make deploy` from the root whenever they want main live. The
# ADR-0086 guard stays: `picode deploy` refuses while anyone is mid-turn
# (exit 2); PICODE_DEPLOY_FORCE=1 overrides for a deliberate one-off.
deploy: ## Rebuild UI+binary, refresh stale public captures, restart the service (owner's call; refuses while agents work)
	flock -x /tmp/picode-deploy.lock $(MAKE) --no-print-directory _deploy

# Body of deploy, held under the lock: parallel sessions deploying between
# one agent's gate and its restart have shipped the wrong tree (2026-09-05).
# Public captures follow the UI here, once per deploy, instead of once per
# branch in `make close` (85 capture commits in three days). The commit
# names its paths, so whatever the owner had staged stays staged.
_deploy: web
	@if ! DOCS_STRICT=1 node scripts/docs-check.mjs >/dev/null 2>&1 && node scripts/docs-check.mjs --strict 2>&1 | grep -q 'inputs changed'; then \
		echo "deploy: public captures are stale — recapturing"; \
		if $(MAKE) --no-print-directory docs-shots >/tmp/picode-deploy-shots.log 2>&1; then \
			if [ -n "$$(git status --porcelain -- docs-site/img)" ]; then \
				git add docs-site/img && git commit -q -m "docs: refresh public captures" -- docs-site/img && echo "deploy: committed refreshed captures"; \
			fi; \
		else echo "deploy: docs-shots failed (see /tmp/picode-deploy-shots.log); deploying without recapture"; fi; \
	fi
	go build -tags embedui -o bin/picode ./cmd/picode
	./bin/picode deploy

cert-timer: ## Install the weekly certificate check (systemd --user)
	mkdir -p ~/.config/systemd/user
	cp scripts/systemd/picode-cert.service scripts/systemd/picode-cert.timer ~/.config/systemd/user/
	systemctl --user daemon-reload
	systemctl --user enable --now picode-cert.timer
	systemctl --user list-timers picode-cert.timer --no-pager

build: web ## Build UI + bin/picode (embeds the UI — ADR-0023)
	go build -tags embedui -o bin/picode ./cmd/picode

desktop: ## Cross-compile the Windows tray + console native host (ADR-0020 / 0043)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
		go build -ldflags "-H=windowsgui -s -w" -o bin/picode-desktop.exe ./cmd/picode-desktop
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
		go build -ldflags "-s -w" -o bin/picode-nmh.exe ./cmd/picode-desktop

desktop-shell: ## Build the v2 Windows shell (Rust/Tauri — needs rustup, x86_64-pc-windows-msvc target, cargo-xwin; ADR-0120)
	cd desktop-shell && cargo xwin build --release --target x86_64-pc-windows-msvc

desktop-restart: desktop desktop-shell ## Build both exes, swap them, relaunch the tray (and the shell if it ran) — NEVER `&` from WSL (scripts/desktop-swap.sh)
	./scripts/desktop-swap.sh

restart: deploy ## Rebuild and restart the systemd service (`picode deploy`)

test: ## Run all Go tests (internal/server sharded across four processes — ADR-0105)
	./scripts/go-test.sh ./...

test-js: $(NODE_STAMP) ## Run the frontend unit tests and the pi package suites
	cd web && npm test
	node --test scripts/*.test.mjs
	node --test packages/pi-roles/test/*.test.ts
	node --test packages/pi-inbox/test/*.test.ts
	node --test packages/pi-checklist/test/*.test.ts
	node --test packages/pi-sysadmin/test/*.test.ts
	node --test packages/pi-diff/test/*.test.ts
	node --test packages/pi-browser-capture/test/*.test.ts
	node --test packages/pi-browser/test/*.test.ts
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

# Keep parity ahead of generation: `docs` rewrites the committed OpenAPI spec,
# so checking afterward would accidentally bless a stale artifact. (llms.txt is
# not committed — ADR-0125.)
ci-docs: ## Verify committed docs parity, then build the public site
	$(MAKE) docs-check
	$(MAKE) docs

# The gates are independent, so they run four at a time (ADR-0105): the Vite
# build, the docs site and the JS suites overlap the Go tests instead of
# queueing behind them. --output-sync keeps each gate's log in one piece.
ci: ## Everything CI runs — the gate for the merge on main (full output in var/ci-last.log)
	./scripts/ci.sh

ci-gates: hooks-check fmt-check vet test test-js build ci-docs vale

# A worktree iteration runs what its diff can break (ADR-0086); `make ci`
# stays the whole matrix for the merge on main.
ci-scoped: ## The gates this branch's diff can break (go: affected packages; web: test-js+build; docs: site+vale)
	./scripts/ci-scoped.sh

close: ## End a worktree session: scoped gates, regenerated artifacts, fast-forward check, closing summary
	./scripts/close.sh

close-summary: ## Print what the closing docs need (commits, diff, owed files) — write handoff/changelog from this
	./scripts/close-summary.sh

changelog: ## Fold docs/changelog.d/ fragments into CHANGELOG.md [Unreleased] and stage it (on main, before a release)
	node scripts/changelog-assemble.mjs

adr: ## Seed the next decision record with its index row: make adr NAME=<short-title> [TITLE="Words"]
	./scripts/adr-new.sh "$(NAME)" $(if $(TITLE),"$(TITLE)")

handoff: ## Render docs/handoff.md (generated view: git state + open topics + session notes, ADR-0123)
	node scripts/handoff-board.mjs

worktree: ## New isolated tree ready to build: make worktree NAME=<name> [BRANCH=feat/<name>]
	./scripts/worktree.sh "$(NAME)" $(BRANCH)

worktree-status: ## What is actually in flight: branch, ahead/behind, dirty files, last commit and green run
	node scripts/worktree-status.mjs

worktree-gc: ## Remove worktrees whose branch is merged, tree clean and idle for an hour (FORCE=1 skips the idle check)
	./scripts/worktree-gc.sh

clean: ## Remove build artifacts
	rm -rf bin/ web/node_modules/ docs-site/node_modules/ docs-site/.vitepress/dist
