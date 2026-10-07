.PHONY: help build run test lint verify generate clean docker-build docker-run k8s-diff

# Variables
BINARY_NAME := lumidive
BIN_DIR := bin
PORT ?= 8080
CACHE_TTL ?= 60s

help: ## Show available make targets
	@echo "lumidive - Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

build: ## Build binary into bin/lumidive
	mkdir -p $(BIN_DIR)
	go build -v -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/lumidive

run: build ## Start local API & Web UI server
	./$(BIN_DIR)/$(BINARY_NAME) server --port $(PORT) --cache-ttl $(CACHE_TTL)

test: ## Run unit and integration tests with -race
	go test -race -v -cover ./...

lint: ## Run golangci-lint
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	elif [ -f "$$(go env GOPATH)/bin/golangci-lint" ]; then \
		$$(go env GOPATH)/bin/golangci-lint run ./...; \
	else \
		echo "golangci-lint not found in PATH or GOPATH"; exit 1; \
	fi

generate: ## Regenerate OpenAPI code and sync spec
	go generate ./...

verify: ## Run strict full verification (drift, lint, tests, build)
	./scripts/verify-all.sh

clean: ## Clean build artifacts and temporary files
	rm -rf $(BIN_DIR) coverage.out

docker-build: ## Build container image locally
	docker build -t lumidive:local .

docker-run: ## Run container image locally on port 8080
	docker run --rm -p 8080:8080 lumidive:local

status: ## Check health and status of local server (default: port 8080 or PORT)
	@echo "Checking local lumidive server status on port $(PORT)..."
	@RESP=$$(curl -sS --max-time 2 http://localhost:$(PORT)/healthz 2>/dev/null || true); \
	if echo "$$RESP" | grep -q '"service":"lumidive"'; then \
		echo "  \033[32m● Running\033[0m: http://localhost:$(PORT)"; \
		echo "$$RESP" | jq . 2>/dev/null || echo "$$RESP"; \
		echo ""; \
	elif [ -n "$$RESP" ]; then \
		echo "  \033[33m! Port Conflict\033[0m: Another service is running on port $(PORT)"; \
	else \
		echo "  \033[31m○ Stopped\033[0m: No server responding on port $(PORT)"; \
	fi

k8s-status: ## Check Kubernetes pod and ingress status
	@echo "--- Lumidive Pods ---"
	@kubectl get pods -n lumidive -o wide
	@echo ""
	@echo "--- Ingress & Public Endpoint ---"
	@kubectl get ingress -n lumidive
	@echo ""
	@echo "--- Health Check (Production) ---"
	@curl -sS https://lumidive.aooba.net/healthz | jq . 2>/dev/null || curl -sS https://lumidive.aooba.net/healthz
	@echo ""

stop: ## Stop any local server running on port $(PORT)
	@PID=$$(lsof -ti :$(PORT) 2>/dev/null || true); \
	if [ -n "$$PID" ]; then \
		echo "Stopping process $$PID on port $(PORT)..."; \
		kill $$PID 2>/dev/null || kill -9 $$PID 2>/dev/null; \
		echo "Stopped."; \
	else \
		echo "No process running on port $(PORT)."; \
	fi

k8s-diff: ## Dry-run rendered Kubernetes manifests via kustomize
	kubectl kustomize k8s/manifests
