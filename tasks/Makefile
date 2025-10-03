.PHONY: help build run dev test clean docker-up docker-down docker-restart logs install migrate

# Variables
APP_NAME=tasks-service
BINARY=bin/$(APP_NAME)
MAIN_PATH=cmd/api/main.go
DB_URL=postgresql://tasks_user:tasks_pass@localhost:5433/tasks_db?sslmode=disable

# Colors
GREEN=\033[0;32m
YELLOW=\033[1;33m
RED=\033[0;31m
NC=\033[0m # No Color

help: ## Show this help message
	@echo "$(GREEN)Tasks Service Makefile Commands:$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""

install: ## Install dependencies
	@echo "$(GREEN)Installing Go dependencies...$(NC)"
	@go mod download
	@go mod tidy
	@echo "$(GREEN)✅ Dependencies installed$(NC)"

build: ## Build the application
	@echo "$(GREEN)Building $(APP_NAME)...$(NC)"
	@mkdir -p bin
	@go build -o $(BINARY) $(MAIN_PATH)
	@echo "$(GREEN)✅ Build complete: $(BINARY)$(NC)"

run: ## Run the application directly
	@echo "$(GREEN)Running $(APP_NAME)...$(NC)"
	@go run $(MAIN_PATH)

dev: ## Run with hot reload (requires air)
	@if command -v air > /dev/null; then \
		echo "$(GREEN)Starting development server with hot reload...$(NC)"; \
		air; \
	else \
		echo "$(YELLOW)⚠️  'air' not found. Install it with: go install github.com/cosmtrek/air@latest$(NC)"; \
		echo "$(GREEN)Running without hot reload...$(NC)"; \
		go run $(MAIN_PATH); \
	fi

test: ## Run tests
	@echo "$(GREEN)Running tests...$(NC)"
	@go test -v -cover ./...
	@echo "$(GREEN)✅ Tests complete$(NC)"

test-coverage: ## Run tests with coverage report
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✅ Coverage report: coverage.html$(NC)"

clean: ## Clean build artifacts
	@echo "$(GREEN)Cleaning build artifacts...$(NC)"
	@rm -rf bin
	@rm -f coverage.out coverage.html
	@echo "$(GREEN)✅ Clean complete$(NC)"

docker-up: ## Start database and dependencies
	@echo "$(GREEN)Starting Docker services...$(NC)"
	@docker compose -f compose.dev.yml up -d
	@echo "$(GREEN)✅ Docker services started$(NC)"
	@echo "$(YELLOW)Waiting for database to be ready...$(NC)"
	@sleep 3

docker-down: ## Stop Docker services
	@echo "$(GREEN)Stopping Docker services...$(NC)"
	@docker compose -f compose.dev.yml down
	@echo "$(GREEN)✅ Docker services stopped$(NC)"

docker-restart: docker-down docker-up ## Restart Docker services

docker-logs: ## Show Docker logs
	@docker compose -f compose.dev.yml logs -f

docker-clean: ## Remove Docker volumes and clean up
	@echo "$(RED)Removing Docker volumes...$(NC)"
	@docker compose -f compose.dev.yml down -v
	@echo "$(GREEN)✅ Docker cleanup complete$(NC)"

logs: ## Show application logs
	@tail -f logs/*.log

db-connect: ## Connect to PostgreSQL database
	@echo "$(GREEN)Connecting to database...$(NC)"
	@docker exec -it tasks_postgres psql -U tasks_user -d tasks_db

db-reset: ## Reset database (drop and recreate)
	@echo "$(RED)Resetting database...$(NC)"
	@docker exec -it tasks_postgres psql -U tasks_user -d tasks_db -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	@docker exec -it tasks_postgres psql -U tasks_user -d tasks_db -f /docker-entrypoint-initdb.d/init.sql
	@echo "$(GREEN)✅ Database reset complete$(NC)"

lint: ## Run linter
	@if command -v golangci-lint > /dev/null; then \
		echo "$(GREEN)Running linter...$(NC)"; \
		golangci-lint run; \
	else \
		echo "$(YELLOW)⚠️  'golangci-lint' not found. Install it from: https://golangci-lint.run/usage/install/$(NC)"; \
	fi

fmt: ## Format code
	@echo "$(GREEN)Formatting code...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)✅ Code formatted$(NC)"

vet: ## Run go vet
	@echo "$(GREEN)Running go vet...$(NC)"
	@go vet ./...
	@echo "$(GREEN)✅ Vet complete$(NC)"

check: fmt vet lint test ## Run all checks (fmt, vet, lint, test)

compose-dev: docker-up run ## Start dependencies and run the service

start: docker-up ## Full start (docker + app)
	@echo "$(GREEN)Starting full application stack...$(NC)"
	@sleep 3
	@make run

stop: ## Stop everything
	@echo "$(GREEN)Stopping all services...$(NC)"
	@pkill -f "$(APP_NAME)" || true
	@make docker-down

restart: stop start ## Restart everything

status: ## Show service status
	@echo "$(GREEN)Service Status:$(NC)"
	@echo ""
	@if pgrep -f "$(APP_NAME)" > /dev/null; then \
		echo "  $(GREEN)✅ Tasks Service: Running$(NC)"; \
	else \
		echo "  $(RED)❌ Tasks Service: Not Running$(NC)"; \
	fi
	@echo ""
	@if docker ps | grep -q tasks_postgres; then \
		echo "  $(GREEN)✅ PostgreSQL: Running$(NC)"; \
	else \
		echo "  $(RED)❌ PostgreSQL: Not Running$(NC)"; \
	fi
	@echo ""

deps-install: ## Install development dependencies (air, golangci-lint)
	@echo "$(GREEN)Installing development tools...$(NC)"
	@go install github.com/cosmtrek/air@latest
	@echo "$(GREEN)✅ Air installed (hot reload)$(NC)"
	@echo "$(YELLOW)For golangci-lint, visit: https://golangci-lint.run/usage/install/$(NC)"

production-build: ## Build for production
	@echo "$(GREEN)Building for production...$(NC)"
	@mkdir -p bin
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o $(BINARY) $(MAIN_PATH)
	@echo "$(GREEN)✅ Production build complete: $(BINARY)$(NC)"

# Default target
.DEFAULT_GOAL := help
