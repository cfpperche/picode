# PiCode — make targets
# Quality gates are the contract (AGENTS.md); `make ci` mirrors GitHub Actions.

.PHONY: help dev ui web build restart deploy install test fmt fmt-check vet ci clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "} {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

dev: ## Run the Go server (HTTPS, port 8445+; serves last `make web` build)
	go run ./cmd/picode

ui: ## Vite HMR on :5173 (proxies /api and /ws to https://localhost:8445)
	cd web && npm run dev

# npm ci wipes and reinstalls, so it is gated on the lockfile rather than run
# by every target that needs node_modules. Without this, `make ci` paid for two
# full installs: one to test the frontend, one to build it.
web/node_modules: web/package-lock.json
	cd web && npm ci
	@touch web/node_modules

web: web/node_modules ## Build the React UI into internal/web/public (ADR-0008)
	cd web && npm run build

cert: ## Provision/renew the mkcert TLS certificate (scripts/setup-cert.sh)
	./scripts/setup-cert.sh

install: build ## Copy bin/picode to ~/.local/bin and enable systemd --user
	./bin/picode install

deploy: build ## Rebuild UI+binary and restart the installed service
	./bin/picode deploy

build: web ## Build UI + bin/picode (embeds the UI — ADR-0023)
	go build -tags embedui -o bin/picode ./cmd/picode

desktop: ## Cross-compile the Windows tray binary (ADR-0020) — no C compiler needed
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
		go build -ldflags "-H=windowsgui -s -w" -o bin/picode-desktop.exe ./cmd/picode-desktop

restart: deploy ## Rebuild and restart the systemd service (`picode deploy`)

test: ## Run all Go tests
	go test ./...

test-js: web/node_modules ## Run the frontend unit tests
	cd web && npm test

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

ci: fmt-check vet test test-js build ## Everything CI runs (includes UI build)

clean: ## Remove build artifacts
	rm -rf bin/ web/node_modules/
