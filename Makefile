# Nebula PaaS Makefile

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags="-w -s -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

.PHONY: all build build-server build-cli clean test lint run dev docker help

all: build ## Build everything

build: build-server build-cli ## Build server and CLI

build-server: ## Build the server binary
	@echo "Building nebula-server..."
	@go build $(LDFLAGS) -o bin/nebula-server ./cmd/nebula-server

build-cli: ## Build the CLI binary
	@echo "Building nebula CLI..."
	@go build $(LDFLAGS) -o bin/nebula ./cmd/nebula

build-linux: ## Build for Linux (cross-compile)
	@echo "Building for Linux..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/nebula-server-linux-amd64 ./cmd/nebula-server
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/nebula-linux-amd64 ./cmd/nebula
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/nebula-server-linux-arm64 ./cmd/nebula-server
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/nebula-linux-arm64 ./cmd/nebula

clean: ## Clean build artifacts
	@rm -rf bin/
	@rm -rf web/dist/
	@rm -rf web/node_modules/

test: ## Run tests
	@go test -v ./...

lint: ## Run linter
	@golangci-lint run ./...

run: build-server ## Run the server locally
	@./bin/nebula-server

dev: ## Run server in development mode with hot reload
	@which air > /dev/null || go install github.com/cosmtrek/air@latest
	@air -c .air.toml

# Frontend
.PHONY: web web-dev web-build

web: web-build ## Build frontend

web-dev: ## Run frontend in dev mode
	@cd web && npm run dev

web-build: ## Build frontend for production
	@cd web && npm ci && npm run build

web-install: ## Install frontend dependencies
	@cd web && npm install

# Docker
.PHONY: docker docker-build docker-push docker-run

docker: docker-build ## Build Docker image

docker-build: ## Build Docker images
	@docker compose build

docker-push: ## Push Docker images to registry
	@docker compose push

docker-run: ## Run with Docker Compose
	@docker compose up -d

docker-stop: ## Stop Docker Compose
	@docker compose down

docker-logs: ## View Docker logs
	@docker compose logs -f

# Installation
.PHONY: install install-server install-cli

install: install-server install-cli ## Install binaries to system

install-server: build-server ## Install server to /usr/local/bin
	@sudo cp bin/nebula-server /usr/local/bin/
	@echo "Server installed to /usr/local/bin/nebula-server"

install-cli: build-cli ## Install CLI to /usr/local/bin
	@sudo cp bin/nebula /usr/local/bin/
	@echo "CLI installed to /usr/local/bin/nebula"

# Database
.PHONY: migrate

migrate: ## Run database migrations
	@./bin/nebula-server migrate

# Release
.PHONY: release release-dry

release: ## Create a new release (requires goreleaser)
	@goreleaser release --clean

release-dry: ## Test release without publishing
	@goreleaser release --snapshot --clean

# Help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
