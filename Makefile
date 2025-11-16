# ============================================================================
# CONFIGURATION
# ============================================================================

# Binary name for compilation
SERVER_BINARY=laba
CLIENT_BINARY=client
TEST_AUDIO_BINARY=test-audio

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

build: build-server build-client build-test-audio

build-server: ## Build the binary
	@echo "Building server..."
	go build -o ./bin/$(SERVER_BINARY) ./cmd/laba/main.go

build-client:
	@echo "Building client..."
	go build -o ./bin/$(CLIENT_BINARY) ./cmd/client/main.go

build-test-audio:
	@echo "Building test audio generator..."
	go build -o ./bin/$(TEST_AUDIO_BINARY) ./cmd/test-audio/main.go

run: build-server ## Build and run the server
	@echo "Running server..."
	./bin/$(SERVER_BINARY)

run-client: build-client ## Build and run the client (requires JWT token)
	@echo "Usage: make run-client TOKEN=your_jwt_token"
	@if [ -z "$(TOKEN)" ]; then \
		echo "Error: TOKEN variable is required"; \
		echo "Example: make run-client TOKEN=eyJhbGc..."; \
		exit 1; \
	fi
	./bin/$(CLIENT_BINARY) -token $(TOKEN)

generate-test-audio: build-test-audio ## Generate a test audio file
	@echo "Generating test audio file..."
	./bin/$(TEST_AUDIO_BINARY) -size 10240 -output test_audio.opus
	@echo "Test file created: test_audio.opus"

clean: ## Clean all binaries
	@echo "Cleaning..."
	go clean
	rm -f ./bin/$(SERVER_BINARY)
	rm -f ./bin/$(CLIENT_BINARY)
	rm -f ./bin/$(TEST_AUDIO_BINARY)
	rm -f test_audio.opus

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
