# PiCode — make targets
# Quality gates are the contract (AGENTS.md); `make ci` mirrors GitHub Actions.

.PHONY: help hooks hooks-check dev ui web docs build restart deploy install test test-js fmt fmt-check vet ci clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "} {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

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
	cd web && npm ci
	@touch $(NODE_STAMP)

web: $(NODE_STAMP) ## Build the React UI into internal/web/public (ADR-0008)
	cd web && npm run build

WWW_STAMP := www/node_modules/.package-lock.json

$(WWW_STAMP): www/package-lock.json
	cd www && npm ci
	@touch $(WWW_STAMP)

docs: $(WWW_STAMP) ## Build the VitePress public site (GitHub Pages)
	cd www && npm run build

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
	node --test packages/pi-roles/test/*.test.ts
	node --test packages/pi-inbox/test/*.test.ts

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

ci: hooks-check fmt-check vet test test-js build docs ## Everything CI runs (includes UI + public docs)

clean: ## Remove build artifacts
	rm -rf bin/ web/node_modules/ www/node_modules/ www/.vitepress/dist
