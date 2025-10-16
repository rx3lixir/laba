# ============================================================================
# CONFIGURATION
# ============================================================================

# Binary name for compilation
BINARY_NAME=laba

# Database connection parameters for migrations tool
MIGRATIONS_HOST=localhost
MIGRATIONS_PORT=5432
MIGRATIONS_USER=laba_admin
MIGRATIONS_PASSWORD=12345
MIGRATIONS_DBNAME=laba_main_db
MIGRATIONS_DRIVER=postgres

# Path where migrations lay
MIGRATIONS_PATH=internal/db/migrations

# Auto-load .env file if it exists
ifneq (,$(wildcard .env))
	include .env
	export
endif

# ============================================================================
# DEFAULT TARGET
# ============================================================================

all: build ## Default build command

# ============================================================================
# BUILD & RUN
# ============================================================================

build: ## Build the binary
	@echo "Building..."
	go build -o ./bin/$(BINARY_NAME) ./cmd/laba/main.go

run: build ## Build and run the app
	@echo "Running..."
	./bin/$(BINARY_NAME)

clean: ## Clean binary
	@echo "Cleaning..."
	go clean
	rm -f ./bin/$(BINARY_NAME)

# ============================================================================
# TESTING
# ============================================================================

test: ## Run all tests with coverage
	@echo "Running tests..."
	go test -cover ./...

# ============================================================================
# MIGRATIONS (using golang goose tool)
# ============================================================================

migrate-status: ## Checking migrations status
	goose -dir $(MIGRATIONS_PATH) $(MIGRATIONS_DRIVER) \
		"host=$(MIGRATIONS_HOST) port=$(MIGRATIONS_PORT) \
		user=$(MIGRATIONS_USER) password=$(MIGRATIONS_PASSWORD) \
		dbname=$(MIGRATIONS_DBNAME) sslmode=disable" \
		status	

migrate-up: ## Applying freshly written migrations
	goose -dir $(MIGRATIONS_PATH) $(MIGRATIONS_DRIVER) \
		"host=$(MIGRATIONS_HOST) port=$(MIGRATIONS_PORT) \
		user=$(MIGRATIONS_USER) password=$(MIGRATIONS_PASSWORD) \
		dbname=$(MIGRATIONS_DBNAME) sslmode=disable" \
		up
			
migrate-down: ## Denying freshly written migrations
	goose -dir $(MIGRATIONS_PATH) $(MIGRATIONS_DRIVER) \
		"host=$(MIGRATIONS_HOST) port=$(MIGRATIONS_PORT) \
		user=$(MIGRATIONS_USER) password=$(MIGRATIONS_PASSWORD) \
		dbname=$(MIGRATIONS_DBNAME) sslmode=disable" \
		down	

migrate-create: ## Creating a new pair of migrations 
	@read -p "Enter migration name: " name; \
	goose -dir $(MIGRATIONS_PATH) $(MIGRATIONS_DRIVER) \
		"host=$(MIGRATIONS_HOST) port=$(MIGRATIONS_PORT) \
		user=$(MIGRATIONS_USER) password=$(MIGRATIONS_PASSWORD) \
		dbname=$(MIGRATIONS_DBNAME) sslmode=disable" \
		create $$name sql	

# ============================================================================
# UTILITIES
# ============================================================================

help: ## Show help
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2}'
