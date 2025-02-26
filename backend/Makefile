# Pavilion Network Backend Makefile
# This file contains commands for development, testing, and deployment

# Variables
BACKEND_DIR = .

# Colors for terminal output
COLOR_RESET = \033[0m
COLOR_GREEN = \033[32m
COLOR_YELLOW = \033[33m
COLOR_BLUE = \033[34m

# Help command
.PHONY: help
help:
	@echo "$(COLOR_GREEN)Pavilion Network Backend Makefile$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Available commands:$(COLOR_RESET)"
	@echo "  $(COLOR_BLUE)make help$(COLOR_RESET)              - Show this help message"
	@echo ""
	@echo "$(COLOR_YELLOW)Development:$(COLOR_RESET)"
	@echo "  $(COLOR_BLUE)make dev$(COLOR_RESET)               - Run backend in development mode"
	@echo ""
	@echo "$(COLOR_YELLOW)Testing:$(COLOR_RESET)"
	@echo "  $(COLOR_BLUE)make test$(COLOR_RESET)              - Run all backend tests"
	@echo "  $(COLOR_BLUE)make test-unit$(COLOR_RESET)         - Run backend unit tests"
	@echo "  $(COLOR_BLUE)make test-integration$(COLOR_RESET)  - Run backend integration tests"
	@echo "  $(COLOR_BLUE)make test-e2e$(COLOR_RESET)          - Run backend end-to-end tests"
	@echo "  $(COLOR_BLUE)make test-video-e2e$(COLOR_RESET)    - Run video end-to-end tests specifically"
	@echo ""
	@echo "$(COLOR_YELLOW)Database:$(COLOR_RESET)"
	@echo "  $(COLOR_BLUE)make db-migrate$(COLOR_RESET)        - Run database migrations"
	@echo "  $(COLOR_BLUE)make db-rollback$(COLOR_RESET)       - Rollback the last database migration"
	@echo ""
	@echo "$(COLOR_YELLOW)Build:$(COLOR_RESET)"
	@echo "  $(COLOR_BLUE)make build$(COLOR_RESET)             - Build backend"
	@echo ""
	@echo "$(COLOR_YELLOW)Utilities:$(COLOR_RESET)"
	@echo "  $(COLOR_BLUE)make clean$(COLOR_RESET)             - Clean build artifacts"
	@echo "  $(COLOR_BLUE)make lint$(COLOR_RESET)              - Run linters"

# Development commands
.PHONY: dev
dev:
	@echo "$(COLOR_GREEN)Starting backend in development mode...$(COLOR_RESET)"
	@cd $(BACKEND_DIR) && go run cmd/server/main.go

# Testing commands
.PHONY: test test-unit test-integration test-e2e test-video-e2e
test:
	@echo "$(COLOR_GREEN)Running all backend tests...$(COLOR_RESET)"
	@cd $(BACKEND_DIR) && go test ./...

test-unit:
	@echo "$(COLOR_GREEN)Running backend unit tests...$(COLOR_RESET)"
	@cd $(BACKEND_DIR) && go test ./internal/*/tests/unit/...

test-integration:
	@echo "$(COLOR_GREEN)Running backend integration tests...$(COLOR_RESET)"
	@cd $(BACKEND_DIR) && ENV=test go test ./internal/*/tests/integration/...

test-e2e:
	@echo "$(COLOR_GREEN)Running backend end-to-end tests...$(COLOR_RESET)"
	@cd $(BACKEND_DIR) && ENV=test E2E_TEST=true go test ./internal/*/tests/e2e/...

test-video-e2e:
	@echo "$(COLOR_GREEN)Running video end-to-end tests...$(COLOR_RESET)"
	@cd $(BACKEND_DIR) && ENV=test E2E_TEST=true go test ./internal/video/tests/e2e/... -v

# Database commands
.PHONY: db-migrate db-rollback
db-migrate:
	@echo "$(COLOR_GREEN)Running database migrations...$(COLOR_RESET)"
	@cd $(BACKEND_DIR) && go run cmd/migrate/main.go up

db-rollback:
	@echo "$(COLOR_GREEN)Rolling back the last database migration...$(COLOR_RESET)"
	@cd $(BACKEND_DIR) && go run cmd/migrate/main.go down 1

# Build commands
.PHONY: build
build:
	@echo "$(COLOR_GREEN)Building backend...$(COLOR_RESET)"
	@cd $(BACKEND_DIR) && go build -o bin/server cmd/server/main.go

# Utility commands
.PHONY: clean lint
clean:
	@echo "$(COLOR_GREEN)Cleaning build artifacts...$(COLOR_RESET)"
	@rm -rf $(BACKEND_DIR)/bin

lint:
	@echo "$(COLOR_GREEN)Running linters...$(COLOR_RESET)"
	@cd $(BACKEND_DIR) && go vet ./...

# Default target
.DEFAULT_GOAL := help 