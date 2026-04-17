.PHONY: help build test lint clean install \
	build-all build-reignxctl build-reignxd build-agent build-api \
	run-dev dev-up dev-down migrate proto-gen \
	docker-build k8s-deploy \
	build-admtools test-admtools

# Variables
GO := go
GOFLAGS := -v
LDFLAGS := -ldflags "-s -w"
BIN_DIR := bin
PROTO_DIR := api/proto
PROTO_GEN_DIR := api/proto/gen

# Build info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS_VERSION := -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)

help: ## Display this help message
	@echo "ReignX - Hybrid Infrastructure Management System"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build-all: build-reignxctl build-reignxd build-agent build-api ## Build all binaries

build-reignxctl: ## Build reignxctl CLI
	@echo "Building reignxctl..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -ldflags "$(LDFLAGS_VERSION)" -o $(BIN_DIR)/reignxctl ./reignxctl
	@echo "reignxctl built successfully"

build-reignxd: ## Build reignxd daemon
	@echo "Building reignxd..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -ldflags "$(LDFLAGS_VERSION)" -o $(BIN_DIR)/reignxd ./reignxd/cmd
	@echo "reignxd built successfully"

build-agent: ## Build reignx-agent
	@echo "Building reignx-agent..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -ldflags "$(LDFLAGS_VERSION)" -o $(BIN_DIR)/reignx-agent ./reignx-agent/cmd
	@echo "reignx-agent built successfully"

build-api: ## Build reignx-api
	@echo "Building reignx-api..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -ldflags "$(LDFLAGS_VERSION)" -o $(BIN_DIR)/reignx-api ./cmd/apiserver
	@echo "reignx-api built successfully"

install: build-all ## Install binaries to /usr/local/bin
	@echo "Installing binaries..."
	sudo cp $(BIN_DIR)/reignxctl /usr/local/bin/
	sudo cp $(BIN_DIR)/reignxd /usr/local/bin/
	sudo cp $(BIN_DIR)/reignx-agent /usr/local/bin/
	sudo cp $(BIN_DIR)/reignx-api /usr/local/bin/
	@echo "Binaries installed successfully"

test: ## Run tests
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	@echo "Tests completed"

lint: ## Run linters
	@echo "Running linters..."
	golangci-lint run --timeout 5m ./...
	@echo "Linting completed"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf $(BIN_DIR)
	rm -rf $(PROTO_GEN_DIR)
	rm -f coverage.txt coverage.html
	@echo "Clean completed"

dev-up: ## Start development environment
	@echo "Starting development environment..."
	docker-compose -f deployments/docker/docker-compose.yaml up -d
	@echo "Development environment started"

dev-down: ## Stop development environment
	@echo "Stopping development environment..."
	docker-compose -f deployments/docker/docker-compose.yaml down
	@echo "Development environment stopped"

migrate: ## Run database migrations
	@echo "Running migrations..."
	$(GO) run migrations/migrate.go up
	@echo "Migrations completed"

proto-gen: ## Generate protobuf code
	@echo "Generating protobuf code..."
	@mkdir -p $(PROTO_GEN_DIR)
	protoc --go_out=$(PROTO_GEN_DIR) --go-grpc_out=$(PROTO_GEN_DIR) $(PROTO_DIR)/*.proto
	@echo "Protobuf code generated"

run-dev: ## Run all services locally
	@echo "Starting ReignX services..."
	@echo "Starting reignxd..."
	./$(BIN_DIR)/reignxd &
	@echo "Starting reignx-api..."
	./$(BIN_DIR)/reignx-api &
	@echo "Services started. Press Ctrl+C to stop."

docker-build: ## Build Docker images
	@echo "Building Docker images..."
	docker build -t reignx/reignxd:$(VERSION) -f deployments/docker/Dockerfile.reignxd .
	docker build -t reignx/reignx-api:$(VERSION) -f deployments/docker/Dockerfile.api .
	docker build -t reignx/reignx-agent:$(VERSION) -f deployments/docker/Dockerfile.agent .
	@echo "Docker images built successfully"

k8s-deploy: ## Deploy to Kubernetes
	@echo "Deploying to Kubernetes..."
	kubectl apply -f deployments/kubernetes/
	@echo "Deployed to Kubernetes"

setup-dev: ## Complete development setup
	@echo "Setting up development environment..."
	go mod download
	make dev-up
	sleep 5
	make migrate
	@echo "Development environment setup completed"

fmt: ## Format code
	@echo "Formatting code..."
	gofmt -s -w .
	$(GO) mod tidy
	@echo "Code formatted"

.DEFAULT_GOAL := help

# Admtools image targets
build-admtools: ## Build admtools multi-arch image
	@echo "Building ReignX admtools image..."
	@./build-admtools.sh

test-admtools: ## Test admtools image
	@echo "Testing ReignX admtools image..."
	@./test-admtools.sh

build-agent-linux: ## Build agent for Linux (arm64 and amd64)
	@echo "Building agent for linux/arm64..."
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/agent-linux-arm64 ./cmd/agent
	@echo "Building agent for linux/amd64..."
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/agent-linux-amd64 ./cmd/agent
	@echo "Linux agents built successfully"

