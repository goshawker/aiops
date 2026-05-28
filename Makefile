# AIOps Platform Makefile

# Variables
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Go binaries
SERVICES := gateway query collector alert job admin

# Python services
AI_SERVICES := anomaly rca alert_agg llm

.PHONY: all build clean test lint fmt proto docker-build docker-up docker-down help

## help: show this help
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'

## all: build everything
all: proto build web-build

## build: build all Go services
build: $(SERVICES)

$(SERVICES):
	@echo "Building cmd/$@..."
	@go build $(LDFLAGS) -o bin/$@ ./cmd/$@

## test: run Go tests
test:
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out

## test-short: run tests without race detector
test-short:
	@go test -short ./...

## lint: run golangci-lint
lint:
	@golangci-lint run ./...

## fmt: format Go code
fmt:
	@gofmt -w .
	@goimports -w .

## proto: generate protobuf code
proto:
	@echo "Generating protobuf code..."
	@protoc --go_out=. --go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		api/proto/*.proto

## web-install: install frontend dependencies
web-install:
	@cd web && npm install

## web-dev: run frontend dev server
web-dev:
	@cd web && npm run dev

## web-build: build frontend for production
web-build:
	@cd web && npm run build

## ai-install: install Python dependencies
ai-install:
	@cd ai && pip install -r requirements.txt

## ai-test: run Python tests
ai-test:
	@cd ai && python -m pytest -v

## docker-build: build all Docker images
docker-build:
	@docker compose -f deploy/docker-compose/docker-compose.yml build

## docker-up: start all services
docker-up:
	@docker compose -f deploy/docker-compose/docker-compose.yml up -d

## docker-down: stop all services
docker-down:
	@docker compose -f deploy/docker-compose/docker-compose.yml down

## docker-logs: show service logs
docker-logs:
	@docker compose -f deploy/docker-compose/docker-compose.yml logs -f

## clean: remove build artifacts
clean:
	@rm -rf bin/ coverage.out
	@cd web && rm -rf dist/ node_modules/
	@cd ai && rm -rf __pycache__/ .pytest_cache/

## init-db: initialize databases
init-db:
	@echo "Initializing SQLite..."
	@sqlite3 aiops.db < deploy/sql/001_init.sql
	@echo "Initializing ClickHouse..."
	@cat deploy/clickhouse/*.sql | clickhouse-client
