APP_NAME := auth-service
SRC_DIR := ./cmd
BUILD_DIR := ./bin
MIGRATIONS_DIR := ./migrations

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

CONFIG_PATH ?= ./config.yml

.PHONY: all ci build run fmt vet lint test tidy clean help \
migrate-create migrate-up migrate-down \
docker-compose-up docker-compose-down

all: build ## Default target: Build the project.

ci: tidy fmt vet lint test build clean ## Run all CI checks.

build: ## Build the project binary.
	@echo "Building the binary for ${GOOS}/${GOARCH}..."
	@mkdir -p "${BUILD_DIR}"
	GOOS="${GOOS}" GOARCH="${GOARCH}" go build -o "${BUILD_DIR}/${APP_NAME}" "${SRC_DIR}"

run: build ## Build and run the application.
	@echo "Running the application..."
	"${BUILD_DIR}/${APP_NAME}" -configPath="${CONFIG_PATH}"

fmt: ## Format code using gofmt.
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet.
	@echo "Running go vet..."
	go vet ./...

lint: ## Lint the code.
	@echo "Running lint checks..."
	@command -v golangci-lint > /dev/null 2>&1 || { echo "golangci-lint not found"; exit 1; }
	golangci-lint run --config=./.golangci.yml

test: ## Run unit tests.
	@echo "Running unit tests..."
	go test -cover ./...

tidy: ## Ensure module dependencies are tidy.
	@echo "Tydying up go.mod and go.sum..."
	go mod tidy

clean: ## Remove build files and artifacts.
	@echo "Cleaning up..."
	@if test -d "${BUILD_DIR}"; then rm -rf "${BUILD_DIR}"; fi
	go clean -testcache

migrate-create: ## Create a new migration. Usage: MIGRATION_NAME=<name> make migrate-create
	@echo "Creating database migration..."
	@if [ -z "${MIGRATION_NAME}" ]; then \
		echo "Error: MIGRATION_NAME is required"; \
		exit 1; \
	fi
	migrate create -ext sql -dir "${MIGRATIONS_DIR}" -seq "${MIGRATION_NAME}"

migrate-up: ## Apply all migrations. Usage: DATABASE_DSN=<dsn> make migrate-up
	@echo "Applying database migrations..."
	@if [ -z "${DATABASE_DSN}" ]; then \
		echo "Error: DATABASE_DSN is required"; \
		exit 1; \
	fi
	migrate -database "${DATABASE_DSN}" -path "${MIGRATIONS_DIR}" up

migrate-down: ## Rollback all migrations. Usage: DATABASE_DSN=<dsn> make migrate-down
	@echo "Rollbacking all migrations..."
	@if [ -z "${DATABASE_DSN}" ]; then \
		echo "Error: DATABASE_DSN is required"; \
		exit 1; \
	fi
	migrate -database "${DATABASE_DSN}" -path "${MIGRATIONS_DIR}" down

docker-compose-up: ## Create and start containers. Usage: make docker-compose-up CONFIG_PATH=<path>
	@echo "Creating and starting containers..."
	CONFIG_PATH="${CONFIG_PATH}" docker-compose up -d

docker-compose-down: ## Stop and remove containers.
	@echo "Stopping and removing containers..."
	docker-compose down

help: ## Display help for each target.
	@echo "Usage: make [target]"
	@echo
	@echo "Available targets:"
	@grep -E '^[a-zA-Z0-9_/.-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'
	@echo
