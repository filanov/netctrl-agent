.PHONY: all build clean test coverage run help deps lint fmt tidy test-race test-cover
.PHONY: docker-build-dev docker-build-prod docker-build-multiarch docker-push-multiarch docker-run docker-shell docker-clean
.PHONY: build-local lint-local test-local fmt-local install-tools

# Variables
BINARY_NAME=netctrl-agent
BIN_DIR=bin
CMD_DIR=cmd/agent

# Docker variables
DOCKER_IMAGE=netctrl-agent:dev
DOCKER_PROD_IMAGE?=netctrl-agent:latest
DOCKER_RUN=docker run --rm \
	-v $(PWD):/workspace \
	-v $(HOME)/.cache/go-build:/root/.cache/go-build \
	-v $(HOME)/go/pkg:/go/pkg \
	-w /workspace \
	$(DOCKER_IMAGE)

# Detect if Docker is available
DOCKER_AVAILABLE := $(shell command -v docker 2> /dev/null)

# Use Docker by default if available, otherwise fall back to local
ifdef DOCKER_AVAILABLE
USE_DOCKER=true
else
USE_DOCKER=false
$(warning Docker not found - falling back to local execution. Install Docker for containerized builds.)
endif

# Default target
all: clean deps build

# ============================================================================
# Docker Image Management
# ============================================================================

docker-build-dev: ## Build development Docker image (with all tools)
	@echo "Building development Docker image..."
	@docker build -f Dockerfile.dev -t $(DOCKER_IMAGE) .
	@echo "Development image built: $(DOCKER_IMAGE)"
	@echo "Image size:"
	@docker images $(DOCKER_IMAGE) --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}"

docker-build-prod: ## Build production Docker image (multi-stage, minimal)
	@echo "Building production Docker image..."
	@docker build -f Dockerfile.prod -t $(DOCKER_PROD_IMAGE) .
	@echo "Production image built: $(DOCKER_PROD_IMAGE)"
	@echo "Image size:"
	@docker images $(DOCKER_PROD_IMAGE) --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}"

docker-build-multiarch: ## Build and test multi-architecture image (linux/amd64,linux/arm64)
	@echo "Building multi-architecture Docker image..."
	@echo "Note: This requires a multi-platform builder. Creating if needed..."
	@docker buildx inspect multiarch-builder >/dev/null 2>&1 || \
		docker buildx create --name multiarch-builder --driver docker-container --use
	@docker buildx build \
		--builder multiarch-builder \
		--platform linux/amd64,linux/arm64 --push \
		-f Dockerfile.prod \
		-t $(DOCKER_PROD_IMAGE) \
		.

docker-run: ## Run agent in production container (requires NETCTRL_CLUSTER_ID and NETCTRL_SERVER_ADDRESS env vars)
	@echo "Running agent in Docker container..."
	@if [ -z "$$NETCTRL_CLUSTER_ID" ]; then \
		echo "Error: NETCTRL_CLUSTER_ID environment variable is required"; \
		exit 1; \
	fi
	@docker run --rm \
		-e NETCTRL_CLUSTER_ID \
		-e NETCTRL_SERVER_ADDRESS=$${NETCTRL_SERVER_ADDRESS:-host.docker.internal:9090} \
		$(DOCKER_PROD_IMAGE) \
		./netctrl-agent --cluster-id=$$NETCTRL_CLUSTER_ID --server-address=$$NETCTRL_SERVER_ADDRESS

docker-shell: ## Open interactive shell in dev container
	@echo "Opening shell in development container..."
	@docker run --rm -it \
		-v $(PWD):/workspace \
		-v $(HOME)/.cache/go-build:/root/.cache/go-build \
		-v $(HOME)/go/pkg:/go/pkg \
		-w /workspace \
		$(DOCKER_IMAGE) /bin/sh

docker-clean: ## Remove Docker images
	@echo "Removing Docker images..."
	@docker rmi -f $(DOCKER_IMAGE) $(DOCKER_PROD_IMAGE) 2>/dev/null || true
	@echo "Docker images removed"

# ============================================================================
# Building
# ============================================================================

build: ## Build the application binary
ifeq ($(USE_DOCKER),true)
	@echo "Building $(BINARY_NAME) (Docker)..."
	@mkdir -p $(BIN_DIR)
	@docker images $(DOCKER_IMAGE) -q | grep -q . || $(MAKE) docker-build-dev
	@$(DOCKER_RUN) go build -o $(BIN_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
else
	@$(MAKE) build-local
endif
	@echo "Build complete: $(BIN_DIR)/$(BINARY_NAME)"

build-local: ## Build using local Go
	@echo "Building $(BINARY_NAME) (local)..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

# ============================================================================
# Testing
# ============================================================================

test: ## Run all tests
ifeq ($(USE_DOCKER),true)
	@echo "Running tests (Docker)..."
	@docker images $(DOCKER_IMAGE) -q | grep -q . || $(MAKE) docker-build-dev
	@$(DOCKER_RUN) go test ./...
else
	@$(MAKE) test-local
endif

test-local: ## Run tests using local Go
	@echo "Running tests (local)..."
	@go test ./...

test-race: ## Run tests with race detector
ifeq ($(USE_DOCKER),true)
	@echo "Running tests with race detector (Docker)..."
	@docker images $(DOCKER_IMAGE) -q | grep -q . || $(MAKE) docker-build-dev
	@$(DOCKER_RUN) go test -race ./...
else
	@echo "Running tests with race detector (local)..."
	@go test -race ./...
endif

test-cover: ## Run tests with coverage
ifeq ($(USE_DOCKER),true)
	@echo "Running tests with coverage (Docker)..."
	@docker images $(DOCKER_IMAGE) -q | grep -q . || $(MAKE) docker-build-dev
	@$(DOCKER_RUN) go test -cover ./...
else
	@echo "Running tests with coverage (local)..."
	@go test -cover ./...
endif

coverage: ## Generate and view test coverage report
ifeq ($(USE_DOCKER),true)
	@echo "Generating coverage report (Docker)..."
	@docker images $(DOCKER_IMAGE) -q | grep -q . || $(MAKE) docker-build-dev
	@$(DOCKER_RUN) go test -coverprofile=coverage.out ./...
	@$(DOCKER_RUN) go tool cover -html=coverage.out -o coverage.html
else
	@echo "Generating coverage report (local)..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
endif
	@echo "Coverage report generated: coverage.html"

# ============================================================================
# Code Quality
# ============================================================================

lint: ## Run golangci-lint and go vet
ifeq ($(USE_DOCKER),true)
	@echo "Running linters (Docker)..."
	@docker images $(DOCKER_IMAGE) -q | grep -q . || $(MAKE) docker-build-dev
	@$(DOCKER_RUN) golangci-lint run ./...
	@$(DOCKER_RUN) go vet ./...
else
	@$(MAKE) lint-local
endif

lint-local: ## Run linters using local tools
	@echo "Running linters (local)..."
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not found, skipping..."
	@go vet ./...

fmt: ## Format all Go code
ifeq ($(USE_DOCKER),true)
	@echo "Formatting code (Docker)..."
	@docker images $(DOCKER_IMAGE) -q | grep -q . || $(MAKE) docker-build-dev
	@$(DOCKER_RUN) go fmt ./...
else
	@$(MAKE) fmt-local
endif

fmt-local: ## Format using local Go
	@echo "Formatting code (local)..."
	@go fmt ./...

# ============================================================================
# Dependency Management
# ============================================================================

deps: ## Download and tidy dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies ready"

tidy: ## Tidy go.mod and go.sum
	@echo "Tidying go.mod..."
	@go mod tidy

install-tools: ## Install development tools locally
	@echo "Installing tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed"

# ============================================================================
# Running & Cleanup
# ============================================================================

run: build ## Build and run the application (requires NETCTRL_CLUSTER_ID env var)
	@echo "Running $(BINARY_NAME)..."
	@if [ -z "$$NETCTRL_CLUSTER_ID" ]; then \
		echo "Error: NETCTRL_CLUSTER_ID environment variable is required"; \
		echo "Usage: NETCTRL_CLUSTER_ID=my-cluster make run"; \
		echo "Optional: Set NETCTRL_SERVER_ADDRESS (default: localhost:9090)"; \
		exit 1; \
	fi
	@./$(BIN_DIR)/$(BINARY_NAME) \
		--cluster-id=$$NETCTRL_CLUSTER_ID \
		--server-address=$${NETCTRL_SERVER_ADDRESS:-localhost:9090}

clean: ## Remove build artifacts
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

# ============================================================================
# Help
# ============================================================================

help: ## Show this help message
	@echo "netctrl-agent Makefile"
	@echo ""
	@echo "Standard targets (use Docker if available):"
	@echo "  all          - Clean, download deps, and build"
	@echo "  build        - Build the application binary"
	@echo "  test         - Run all tests"
	@echo "  test-race    - Run tests with race detector"
	@echo "  test-cover   - Run tests with coverage"
	@echo "  coverage     - Generate and view test coverage report"
	@echo "  lint         - Run golangci-lint and go vet"
	@echo "  fmt          - Format all Go code"
	@echo "  run          - Build and run the application"
	@echo ""
	@echo "Docker-specific targets:"
	@echo "  docker-build-dev        - Build development Docker image"
	@echo "  docker-build-prod       - Build production Docker image (multi-stage)"
	@echo "  docker-build-multiarch  - Build multi-arch image (amd64, arm64) locally"
	@echo "  docker-push-multiarch   - Build and push multi-arch image to registry"
	@echo "  docker-run              - Run agent in production container"
	@echo "  docker-shell            - Open interactive shell in dev container"
	@echo "  docker-clean            - Remove Docker images"
	@echo ""
	@echo "Local execution targets (bypass Docker):"
	@echo "  build-local  - Build using local Go installation"
	@echo "  lint-local   - Lint using local tools"
	@echo "  test-local   - Test using local Go"
	@echo "  fmt-local    - Format using local Go"
	@echo ""
	@echo "Utility targets:"
	@echo "  clean        - Remove build artifacts"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  tidy         - Tidy go.mod and go.sum"
	@echo "  install-tools - Install golangci-lint locally"
	@echo "  help         - Show this help message"
	@echo ""
	@echo "Docker status: $(if $(DOCKER_AVAILABLE),available,not found - using local execution)"
	@echo ""
	@echo "Agent Configuration:"
	@echo "  NETCTRL_CLUSTER_ID      - Cluster ID (required for run/docker-run)"
	@echo "  NETCTRL_SERVER_ADDRESS  - Server address (default: localhost:9090)"
	@echo ""
	@echo "Example usage:"
	@echo "  NETCTRL_CLUSTER_ID=my-cluster make run"
	@echo "  NETCTRL_CLUSTER_ID=my-cluster NETCTRL_SERVER_ADDRESS=server:9090 make docker-run"
	@echo "  make docker-build-multiarch  # Build for amd64 and arm64"
	@echo "  DOCKER_REGISTRY=docker.io/myuser make docker-push-multiarch  # Push to registry"

.DEFAULT_GOAL := help
