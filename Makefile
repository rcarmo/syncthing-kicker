# Project automation

GO ?= go
GOFLAGS ?=


.PHONY: help clean \
	deps test fmt vet build run check

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_\-]+:.*##/ {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

clean: ## Remove cache artifacts
	rm -rf __pycache__ .pytest_cache .ruff_cache
	$(GO) clean -testcache
	rm -f syncthing-kicker
	rm -rf dist/ build/

deps: ## Download Go module dependencies
	$(GO) mod download
	$(GO) mod tidy
	$(GO) mod verify

test: ## Run Go tests
	$(GO) test ./...

fmt: ## Format Go code
	$(GO) fmt ./...

vet: ## Vet Go code
	$(GO) vet ./...

build: ## Build Go binary
	$(GO) build $(GOFLAGS) ./cmd/syncthing-kicker

run: ## Run Go service
	$(GO) run ./cmd/syncthing-kicker

check: ## Run Go fmt/vet/test
	$(MAKE) fmt
	$(MAKE) vet
	$(MAKE) test
