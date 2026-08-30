# PiCode — make targets
# Quality gates are the contract (AGENTS.md); `make ci` mirrors GitHub Actions.

.PHONY: help dev ui web build restart deploy install test fmt fmt-check vet ci clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "} {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

dev: ## Run the Go server (HTTPS, port 8445+; serves last `make web` build)
	go run ./cmd/picode

ui: ## Vite HMR on :5173 (proxies /api and /ws to https://localhost:8445)
	cd web && npm run dev

web: ## Build the React UI into internal/web/public (ADR-0008)
	cd web && npm ci && npm run build

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

fmt: ## Format all Go code
	gofmt -w .

fmt-check: ## Fail if any file is unformatted
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt needed on:"; echo "$$out"; exit 1; fi

vet: ## Static analysis
	go vet ./...

ci: fmt-check vet test build ## Everything CI runs (includes UI build)

clean: ## Remove build artifacts
	rm -rf bin/ web/node_modules/
