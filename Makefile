# ==============================================================================
# Herotech Makefile
# ==============================================================================
# Load environment variables from .env file
ifneq (,$(wildcard .env))
	include .env
	export $(shell sed 's/=.*//' .env)
endif

# ------------------------------------------------------------------------------
# Configuration
# ------------------------------------------------------------------------------
APP_NAME          := herotech
POSTGRES_CONTAINER:= postgres_herotech
POSTGRES_IMAGE    := postgres:18.4-alpine3.24
MIGRATIONS_DIR    := internal/db/migrations
SQLC_DIR          := internal/db/sqlc
BINARY_DIR        := bin
BINARY_NAME       := $(BINARY_DIR)/$(APP_NAME)

# Colors for prettier output
COLOR_RESET   := \033[0m
COLOR_INFO    := \033[36m
COLOR_SUCCESS := \033[32m
COLOR_WARNING := \033[33m
COLOR_ERROR   := \033[31m

# ------------------------------------------------------------------------------
# Default target: show help
# ------------------------------------------------------------------------------
.DEFAULT_GOAL := help

# ------------------------------------------------------------------------------
# Development targets
# ------------------------------------------------------------------------------
.PHONY: run
run: ## Run the application
	@echo "$(COLOR_INFO)▶ Starting $(APP_NAME)...$(COLOR_RESET)"
	go run cmd/api/main.go

.PHONY: air
air: ## Run the application
	@echo "$(COLOR_INFO)▶ Starting And Watching $(APP_NAME)...$(COLOR_RESET)"
	air

.PHONY: test
test: ## Run all tests
	@echo "$(COLOR_INFO)▶ Running tests...$(COLOR_RESET)"
	go test -v -race -cover ./...

.PHONY: fmt
fmt: ## Format Go source code
	@echo "$(COLOR_INFO)▶ Formatting code...$(COLOR_RESET)"
	go fmt ./...
	@echo "$(COLOR_SUCCESS)✓ Formatting complete$(COLOR_RESET)"

.PHONY: vet
vet: ## Run go vet
	@echo "$(COLOR_INFO)▶ Running go vet...$(COLOR_RESET)"
	go vet ./...
	@echo "$(COLOR_SUCCESS)✓ Vet checks passed$(COLOR_RESET)"

.PHONY: build
build: ## Build the binary
	@echo "$(COLOR_INFO)▶ Building $(APP_NAME)...$(COLOR_RESET)"
	@mkdir -p $(BINARY_DIR)
	go build -ldflags="-s -w" -o $(BINARY_NAME) cmd/api/main.go
	@echo "$(COLOR_SUCCESS)✓ Binary built at $(BINARY_NAME)$(COLOR_RESET)"

.PHONY: clean
clean: ## Remove build artifacts
	@echo "$(COLOR_INFO)▶ Cleaning up...$(COLOR_RESET)"
	@rm -rf $(BINARY_DIR)
	@echo "$(COLOR_SUCCESS)✓ Cleaned$(COLOR_RESET)"

.PHONY: swagger
swagger:
	swag init -g cmd/api/main.go -o docs --dir . --parseInternal --parseDependency

# ------------------------------------------------------------------------------
# Database (PostgreSQL via Docker)
# ------------------------------------------------------------------------------
.PHONY: postgres
postgres: ## Start PostgreSQL container
	@if docker ps -a --format '{{.Names}}' | grep -q "^$(POSTGRES_CONTAINER)$$"; then \
		echo "$(COLOR_WARNING)⚠ Container $(POSTGRES_CONTAINER) already exists. Starting it...$(COLOR_RESET)"; \
		docker start $(POSTGRES_CONTAINER); \
	else \
		echo "$(COLOR_INFO)▶ Creating and starting PostgreSQL container...$(COLOR_RESET)"; \
		docker run --name $(POSTGRES_CONTAINER) \
			-p $(DB_PORT):5432 \
			-e POSTGRES_USER=$(DB_USER) \
			-e POSTGRES_PASSWORD=$(DB_PASSWORD) \
			-e POSTGRES_DB=$(DB_NAME) \
			-d $(POSTGRES_IMAGE); \
		echo "$(COLOR_SUCCESS)✓ PostgreSQL started$(COLOR_RESET)"; \
	fi

.PHONY: postgres_rm
postgres_rm: ## Stop and remove PostgreSQL container
	@echo "$(COLOR_WARNING)▶ Stopping and removing $(POSTGRES_CONTAINER)...$(COLOR_RESET)"
	@docker stop $(POSTGRES_CONTAINER) 2>/dev/null || true
	@docker rm $(POSTGRES_CONTAINER) 2>/dev/null || true
	@echo "$(COLOR_SUCCESS)✓ Container removed$(COLOR_RESET)"

.PHONY: wait_for_postgres
wait_for_postgres: ## Wait until PostgreSQL is fully ready for connections
	@echo "$(COLOR_INFO)▶ Waiting for PostgreSQL...$(COLOR_RESET)"
	@timeout=30; \
	while [ $$timeout -gt 0 ]; do \
		if docker exec $(POSTGRES_CONTAINER) pg_isready -U $(DB_USER) >/dev/null 2>&1; then \
			if command -v psql >/dev/null 2>&1; then \
				if PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -U $(DB_USER) -d $(DB_NAME) -c "SELECT 1" >/dev/null 2>&1; then \
					echo "$(COLOR_SUCCESS)✓ PostgreSQL is fully ready$(COLOR_RESET)"; \
					break; \
				fi; \
			else \
				sleep 2; \
				echo "$(COLOR_SUCCESS)✓ PostgreSQL is ready$(COLOR_RESET)"; \
				break; \
			fi; \
		fi; \
		echo "$(COLOR_WARNING)⏳ PostgreSQL not ready, retrying... ($$timeout sec left)$(COLOR_RESET)"; \
		sleep 2; \
		timeout=$$((timeout-2)); \
	done; \
	if [ $$timeout -le 0 ]; then \
		echo "$(COLOR_ERROR)✗ PostgreSQL did not become ready in time$(COLOR_RESET)"; \
		exit 1; \
	fi

.PHONY: create_db
create_db: ## Create the application database
	@echo "$(COLOR_INFO)▶ Creating database '$(APP_NAME)'...$(COLOR_RESET)"
	@docker exec $(POSTGRES_CONTAINER) createdb --username=$(DB_USER) --owner=$(DB_USER) $(DB_NAME) 2>/dev/null || \
		echo "$(COLOR_WARNING)⚠ Database may already exist$(COLOR_RESET)"
	@echo "$(COLOR_SUCCESS)✓ Database ready$(COLOR_RESET)"

.PHONY: drop_db
drop_db: ## Drop the application database
	@echo "$(COLOR_WARNING)▶ Dropping database '$(APP_NAME)'...$(COLOR_RESET)"
	@docker exec $(POSTGRES_CONTAINER) dropdb --username=$(DB_USER) --if-exists $(APP_NAME)
	@echo "$(COLOR_SUCCESS)✓ Database dropped$(COLOR_RESET)"

# ------------------------------------------------------------------------------
# Database Migrations (golang-migrate)
# ------------------------------------------------------------------------------
MIGRATE := $(shell command -v migrate 2>/dev/null)

.PHONY: ensure_migrate
ensure_migrate:
ifndef MIGRATE
	$(error "$(COLOR_ERROR)✗ 'migrate' CLI not found. Please install golang-migrate: https://github.com/golang-migrate/migrate/tree/master/cmd/migrate$(COLOR_RESET)")
endif

.PHONY: create_migration
create_migration: ensure_migrate ## Create a new migration file (usage: make create_migration name=add_users_table)
	@read -p "Enter migration name (default: $(name)): " input; \
	name=$${input:-$(name)}; \
	if [ -z "$$name" ]; then \
		echo "$(COLOR_ERROR)✗ Migration name is required$(COLOR_RESET)"; \
		exit 1; \
	fi; \
	migrate create -ext sql -dir $(MIGRATIONS_DIR) $$name
	@echo "$(COLOR_SUCCESS)✓ Migration files created for '$$name'$(COLOR_RESET)"

DB_URL := postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

.PHONY: migrate_up
migrate_up: ensure_migrate ## Apply all pending migrations
	@echo "$(COLOR_INFO)▶ Applying migrations...$(COLOR_RESET)"
	@if [ -z "$(DB_USER)" ] || [ -z "$(DB_PASSWORD)" ] || [ -z "$(DB_HOST)" ] || [ -z "$(DB_PORT)" ] || [ -z "$(DB_NAME)" ]; then \
		echo "$(COLOR_ERROR)✗ Database environment variables are not all set$(COLOR_RESET)"; \
		exit 1; \
	fi
	@migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" -verbose up
	@echo "$(COLOR_SUCCESS)✓ Migrations applied$(COLOR_RESET)"

.PHONY: migrate_down
migrate_down: ensure_migrate ## Rollback the last migration
	@echo "$(COLOR_WARNING)▶ Rolling back last migration...$(COLOR_RESET)"
	@if [ -z "$(DB_USER)" ] || [ -z "$(DB_PASSWORD)" ] || [ -z "$(DB_HOST)" ] || [ -z "$(DB_PORT)" ] || [ -z "$(DB_NAME)" ]; then \
		echo "$(COLOR_ERROR)✗ Database environment variables are not all set$(COLOR_RESET)"; \
		exit 1; \
	fi
	@migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" -verbose down 1
	@echo "$(COLOR_SUCCESS)✓ Migration rolled back$(COLOR_RESET)"

.PHONY: migrate_force
migrate_force: ensure_migrate ## Force set migration version (usage: make migrate_force VERSION=3)
	@if [ -z "$(VERSION)" ]; then \
		echo "$(COLOR_ERROR)✗ VERSION argument is required (e.g., make migrate_force VERSION=3)$(COLOR_RESET)"; \
		exit 1; \
	fi
	@migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" force $(VERSION)
	@echo "$(COLOR_SUCCESS)✓ Migration version forced to $(VERSION)$(COLOR_RESET)"

# ------------------------------------------------------------------------------
# Database Backup
# ------------------------------------------------------------------------------
.PHONY: db_schema_dump
db_schema_dump: ## Dump only database schema to file
	@echo "$(COLOR_INFO)▶ Dumping database schema...$(COLOR_RESET)"
	@mkdir -p internal/db/dump
	docker exec -i $(POSTGRES_CONTAINER) pg_dump -U $(DB_USER) -d $(DB_NAME) --schema-only > internal/db/dump/$(DB_NAME)_schema.sql
	@echo "$(COLOR_SUCCESS)✓ Schema dumped successfully to internal/db/dump/$(DB_NAME)_schema.sql$(COLOR_RESET)"

# ------------------------------------------------------------------------------
# SQLC
# ------------------------------------------------------------------------------
.PHONY: sqlc
sqlc: ## Generate type-safe Go code from SQL (requires sqlc)
	@command -v sqlc >/dev/null 2>&1 || { \
		echo "$(COLOR_ERROR)✗ sqlc not found. Install: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest$(COLOR_RESET)"; \
		exit 1; \
	}
	@echo "$(COLOR_INFO)▶ Generating SQLC code...$(COLOR_RESET)"
	@find $(SQLC_DIR) -type f -name "*.go" ! -name "*_test.go" -delete
	@sqlc generate
	@echo "$(COLOR_SUCCESS)✓ SQLC generation complete$(COLOR_RESET)"

# ------------------------------------------------------------------------------
# Composite workflows
# ------------------------------------------------------------------------------
.PHONY: init_db
init_db: postgres_rm postgres wait_for_postgres create_db migrate_up ## Full database initialization (fresh start)
	@echo "$(COLOR_SUCCESS)✓ Database initialization complete$(COLOR_RESET)"

.PHONY: dev
dev: postgres wait_for_postgres swagger air ## Start development environment with fresh database and run app

.PHONY: compose-up
compose-up: ## Start services using Docker Compose
	@echo "$(COLOR_INFO)▶ Starting services with docker-compose...$(COLOR_RESET)"
	docker compose -p $(APP_NAME) -f docker/docker-compose.yaml up -d

.PHONY: compose-build
compose-build: ## Build and start services using Docker Compose
	@echo "$(COLOR_INFO)▶ Building and starting services with docker-compose...$(COLOR_RESET)"
	docker compose -p $(APP_NAME) -f docker/docker-compose.yaml up --build -d

.PHONY: compose-down
compose-down: ## Stop Docker Compose services
	@echo "$(COLOR_WARNING)▶ Stopping docker-compose services...$(COLOR_RESET)"
	docker compose -p $(APP_NAME) -f docker/docker-compose.yaml down -v
	@echo "$(COLOR_SUCCESS)✓ Services stopped$(COLOR_RESET)"

# ------------------------------------------------------------------------------
# Project Structure Generator
# ------------------------------------------------------------------------------

.PHONY: project_tree
project_tree: ## Generate project structure tree (excluding unnecessary folders)
	@echo "$(COLOR_INFO)▶ Generating project structure...$(COLOR_RESET)"
	@tree -I "bin|vendor|.git|.idea|node_modules" > project_structure.txt
	@echo "$(COLOR_SUCCESS)✓ project_structure.txt created$(COLOR_RESET)"
	
# ------------------------------------------------------------------------------
# Mocks Generation (Mockery)
# ------------------------------------------------------------------------------
.PHONY: mocks
mocks: ## Generate mocks for interfaces using mockery
	@command -v mockery >/dev/null 2>&1 || { \
	    echo "$(COLOR_ERROR)✗ mockery not found. Install: go install github.com/vektra/mockery/v2@latest$(COLOR_RESET)"; \
	    exit 1; \
	}
	@echo "$(COLOR_INFO)▶ Generating mocks...$(COLOR_RESET)"
	@rm -rf internal/mocks/* 2>/dev/null || true
	@mockery
	@echo "$(COLOR_SUCCESS)✓ Mocks generated successfully in internal/mocks$(COLOR_RESET)"
# ------------------------------------------------------------------------------
# Help
# ------------------------------------------------------------------------------
.PHONY: help
help: ## Show this help message
	@echo "$(COLOR_INFO)Available targets:$(COLOR_RESET)"
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(COLOR_SUCCESS)%-20s$(COLOR_RESET) %s\n", $$1, $$2}'
	@echo "\n$(COLOR_INFO)Environment variables needed (from .env):$(COLOR_RESET)"
	@echo "  DB_URL   PostgreSQL connection string (e.g., postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME))"
