# ============================================================================
# CONFIGURATION
# ============================================================================

BINARY_NAME=laba
MIGRATIONS_PATH=internal/db/migrations
DB_URL=postgres://rx3lixir:password@localhost:5432/laba-storer?sslmode=disable

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
	@echo "🔨 Building..."
	go build -o ./bin/$(BINARY_NAME) ./cmd/laba/main.go

run: build ## Build and run the app
	@echo "🚀 Running..."
	./bin/$(BINARY_NAME)

clean: ## Clean binary
	@echo "🧹 Cleaning..."
	go clean
	rm -f ./bin/$(BINARY_NAME)

# ============================================================================
# TESTING
# ============================================================================

test: ## Run all tests with coverage
	@echo "🧪 Running tests..."
	go test -cover ./...

# ============================================================================
# MIGRATIONS (using migrate/migrate docker image)
# ============================================================================

migrate-new: ## Create a new migration: make migrate-new name=create_users
ifndef name
	$(error ❌ name is not set. Usage: make migrate-new name=your_migration_name)
endif
	@echo "🧬 Creating new migration '$(name)'..."
	docker run --rm -v $(shell pwd)/$(MIGRATIONS_PATH):/migrations \
		migrate/migrate create -ext sql -dir /migrations $(name)

migrate-up: ## Apply all migrations
	@echo "📥 Applying all migrations..."
	docker run --rm -v $(shell pwd)/$(MIGRATIONS_PATH):/migrations \
		--network host migrate/migrate \
		-path=/migrations -database "$(DB_URL)" up

migrate-down: ## Rollback one migration
	@echo "📤 Rolling back one migration..."
	docker run --rm -v $(shell pwd)/$(MIGRATIONS_PATH):/migrations \
		--network host migrate/migrate \
		-path=/migrations -database "$(DB_URL)" down 1

migrate-force-down-all: ## Rollback all migrations (dangerous!)
	@echo "⚠️ Force dropping all migrations!"
	docker run --rm -v $(shell pwd)/$(MIGRATIONS_PATH):/migrations \
		--network host migrate/migrate \
		-path=/migrations -database "$(DB_URL)" drop -f

migrate-status: ## Get current DB version
	@echo "🔍 Current migration version:"
	docker run --rm --network host \
		migrate/migrate -path=$(shell pwd)/$(MIGRATIONS_PATH) -database "$(DB_URL)" version || true

# ============================================================================
# UTILITIES
# ============================================================================

help: ## Show help
	@echo "📖 Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2}'
