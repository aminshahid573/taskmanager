# Variables
APP_NAME := taskmanager
MAIN_PATH := ./cmd/api
BINARY_NAME := $(APP_NAME)
CONFIG_PATH := config/local.yaml
DOCKER_COMPOSE := docker-compose
DOCKER_IMAGE := $(APP_NAME):latest

DB_HOST ?= localhost
DB_PORT ?= 5432
DB_USER ?= taskmanager
DB_PASSWORD ?= taskmanager123
DB_NAME ?= taskmanager
DB_SSL_MODE ?= disable
DB_URL := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL_MODE)
MIGRATIONS_PATH := ./migrations

GOFLAGS := -v
LDFLAGS := -ldflags="-w -s"
COVERAGE_OUT := coverage.out
COVERAGE_HTML := coverage.html

# help

.PHONY: help
help: ## Display this help screen
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*##"; printf ""} /^[a-zA-Z_-]+:.*?##/ { printf "  %-25s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# build and run

.PHONY: all
all: clean deps fmt lint vet test build ## Run all build steps

.PHONY: build
build: ## Build the application
	@echo "Building $(APP_NAME)..."
	CGO_ENABLED=0 go build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BINARY_NAME)"

.PHONY: run
run: ## Run the application locally
	@echo "Running $(APP_NAME)..."
	go run $(MAIN_PATH) -config $(CONFIG_PATH)

.PHONY: dev
dev: ## Run the application in development mode
	@echo "Starting development server with hot reload..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Air not installed. Running with go run instead..."; \
		go run $(MAIN_PATH) -config $(CONFIG_PATH); \
	fi

.PHONY: install
install: build ## Install the application
	@echo "Installing $(APP_NAME)..."
	go install $(MAIN_PATH)
	@echo "Installation complete"

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY_NAME)
	@rm -f $(COVERAGE_OUT)
	@rm -f $(COVERAGE_HTML)
	@go clean -cache -testcache
	@echo "Clean complete"

# dependency

.PHONY: deps
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies downloaded"

# code quality

.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	@echo "Formatting complete"

.PHONY: lint
lint: ## Run linter
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run ./...; \
	fi

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...
	@echo "Vet complete"

# testing

.PHONY: test
test: ## Run tests
	@echo "Running tests..."
	go test -v -race ./...
	@echo "Tests complete"

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=$(COVERAGE_OUT) -covermode=atomic ./...
	go tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)
	go tool cover -func=$(COVERAGE_OUT)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

# ci/cd

.PHONY: ci
ci: deps fmt lint vet test ## Run CI pipeline steps
	@echo "CI pipeline complete"

# docker

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .
	@echo "Docker image built: $(DOCKER_IMAGE)"

.PHONY: docker-up
docker-up: ## Start Docker containers
	@echo "Starting Docker containers..."
	$(DOCKER_COMPOSE) up -d
	@echo "Docker containers started"

.PHONY: docker-down
docker-down: ## Stop Docker containers
	@echo "Stopping Docker containers..."
	$(DOCKER_COMPOSE) down
	@echo "Docker containers stopped"

.PHONY: docker-restart
docker-restart: docker-down docker-up ## Restart Docker containers

.PHONY: docker-rebuild
docker-rebuild: ## Rebuild and restart Docker containers
	$(DOCKER_COMPOSE) up -d --build

.PHONY: docker-logs
docker-logs: ## View Docker container logs
	$(DOCKER_COMPOSE) logs -f

.PHONY: docker-clean
docker-clean: ## Remove Docker containers and volumes
	@echo "Removing Docker containers and volumes..."
	$(DOCKER_COMPOSE) down -v --remove-orphans
	docker rmi $(DOCKER_IMAGE) 2>/dev/null || true
	@echo "Docker cleanup complete"

# dtabase migration

.PHONY: migrate
migrate: migrate-up ## Alias for migrate-up

.PHONY: migrate-up
migrate-up: ## Run database migrations (apply all pending migrations)
	@echo "Running database migrations..."
	@if command -v migrate > /dev/null; then \
		migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" up; \
	else \
		echo "golang-migrate not installed. Installing..."; \
		go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest; \
		migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" up; \
	fi
	@echo "Migrations applied"

.PHONY: migrate-down
migrate-down: ## Rollback last database migration
	@echo "Rolling back last migration..."
	@if command -v migrate > /dev/null; then \
		migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" down 1; \
	else \
		echo "golang-migrate not installed. Installing..."; \
		go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest; \
		migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" down 1; \
	fi
	@echo "Rollback complete"

.PHONY: migrate-down-all
migrate-down-all: ## Rollback all database migrations
	@echo "Rolling back all migrations..."
	@if command -v migrate > /dev/null; then \
		migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" down -all; \
	else \
		echo "golang-migrate not installed. Installing..."; \
		go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest; \
		migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" down -all; \
	fi
	@echo "All migrations rolled back"

.PHONY: migrate-force
migrate-force: ## Force migration version (usage: make migrate-force VERSION=1)
ifndef VERSION
	$(error VERSION is required. Usage: make migrate-force VERSION=1)
endif
	@echo "Forcing migration version to $(VERSION)..."
	@if command -v migrate > /dev/null; then \
		migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" force $(VERSION); \
	else \
		echo "golang-migrate not installed. Installing..."; \
		go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest; \
		migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" force $(VERSION); \
	fi
	@echo "Migration version forced to $(VERSION)"

.PHONY: migrate-status
migrate-status: ## Check migration status
	@echo "Checking migration status..."
	@if command -v migrate > /dev/null; then \
		migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" version; \
	else \
		echo "golang-migrate not installed. Installing..."; \
		go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest; \
		migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" version; \
	fi

.DEFAULT_GOAL := help

