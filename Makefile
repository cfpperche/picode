# PiCode — make targets
# Quality gates are the contract (AGENTS.md); `make ci` mirrors GitHub Actions.

.PHONY: help dev build test fmt fmt-check vet ci clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "} {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

dev: ## Run the dev server (HTTPS, port 8445+; PICODE_INSECURE=1 for http)
	go run ./cmd/picode

cert: ## Provision/renew the mkcert TLS certificate (scripts/setup-cert.sh)
	./scripts/setup-cert.sh

install: ## Install as systemd user service + cert renewal timer
	./scripts/install-systemd.sh

build: ## Build bin/picode
	go build -o bin/picode ./cmd/picode

test: ## Run all tests
	go test ./...

fmt: ## Format all Go code
	gofmt -w .

fmt-check: ## Fail if any file is unformatted
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt needed on:"; echo "$$out"; exit 1; fi

vet: ## Static analysis
	go vet ./...

ci: fmt-check vet test build ## Everything CI runs

clean: ## Remove build artifacts
	rm -rf bin/
