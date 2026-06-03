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

## agent-build-linux-amd64: build agent for Linux amd64
agent-build-linux-amd64:
	@GOOS=linux GOARCH=amd64 go build -o bin/agent-linux-amd64 ./cmd/agent

## agent-build-linux-arm64: build agent for Linux arm64 (鲲鹏/飞腾)
agent-build-linux-arm64:
	@GOOS=linux GOARCH=arm64 go build -o bin/agent-linux-arm64 ./cmd/agent

## agent-build-linux-loong64: build agent for LoongArch (龙芯)
agent-build-linux-loong64:
	@GOOS=linux GOARCH=loong64 go build -o bin/agent-linux-loong64 ./cmd/agent

## agent-build-all: build agent for all supported platforms
agent-build-all: agent-build-linux-amd64 agent-build-linux-arm64 agent-build-darwin-amd64 agent-build-darwin-arm64

## agent-build-darwin-amd64: build agent for Darwin amd64
agent-build-darwin-amd64:
	@GOOS=darwin GOARCH=amd64 go build -o bin/agent-darwin-amd64 ./cmd/agent

## agent-build-darwin-arm64: build agent for Darwin arm64 (Apple Silicon)
agent-build-darwin-arm64:
	@GOOS=darwin GOARCH=arm64 go build -o bin/agent-darwin-arm64 ./cmd/agent
	@echo "Agent binaries built:"
	@ls -lh bin/agent-*

## docker-build: build all Docker images
docker-build:
	@docker compose -f deploy/docker-compose/docker-compose.yml build

## docker-up: start all services
docker-up:
	@docker compose -f deploy/docker-compose/docker-compose.yml up -d

## docker-down: stop all services
docker-down:
	@docker compose -f deploy/docker-compose/docker-compose.yml down

## docker-restart: restart all services
docker-restart: docker-down docker-up

## tls-setup-domain: setup TLS with Let's Encrypt (usage: make tls-setup-domain DOMAIN=aiops.example.com)
tls-setup-domain:
	@bash deploy/tls/setup-tls.sh --domain $(DOMAIN)

## tls-setup-ip: setup TLS with self-signed cert for IP (usage: make tls-setup-ip IP=192.168.1.100)
tls-setup-ip:
	@bash deploy/tls/setup-tls.sh --ip $(IP)

## tls-setup-self-signed: setup TLS with self-signed cert (localhost)
tls-setup-self-signed:
	@bash deploy/tls/setup-tls.sh --self-signed

## backup: run full backup (SQLite + ClickHouse + VictoriaMetrics)
backup:
	@bash deploy/backup/backup.sh

## backup-sqlite: backup SQLite only
backup-sqlite:
	@bash deploy/backup/backup.sh --sqlite

## backup-clickhouse: backup ClickHouse only
backup-clickhouse:
	@bash deploy/backup/backup.sh --clickhouse

## backup-vm: backup VictoriaMetrics only
backup-vm:
	@bash deploy/backup/backup.sh --vm

## backup-list: list available backups
backup-list:
	@bash deploy/backup/restore.sh --list

## backup-restore: restore from backup (usage: make backup-restore TYPE=sqlite FILE=/path/to/backup.db)
backup-restore:
	@bash deploy/backup/restore.sh --$(TYPE) $(FILE)

## docker-logs: show service logs
docker-logs:
	@docker compose -f deploy/docker-compose/docker-compose.yml logs -f

## docker-ps: show running services
docker-ps:
	@docker compose -f deploy/docker-compose/docker-compose.yml ps

## offline-save-images: save Docker images for offline deployment
offline-save-images:
	@bash deploy/offline/save-images.sh

## offline-build: build offline installer package
offline-build:
	@bash deploy/offline/build-offline.sh

## deploy-agent: deploy agent to remote host (usage: make deploy-agent HOST=user@host)
deploy-agent:
	@if [ -z "$(HOST)" ]; then echo "Usage: make deploy-agent HOST=user@host [COLLECTOR=http://host:8084]"; exit 1; fi
	@ssh $(HOST) "bash -s" -- < deploy/agent/install.sh --collector $(COLLECTOR)

## deploy-init-db: initialize databases
init-db:
	@echo "Initializing ClickHouse..."
	@for f in deploy/sql/*.sql; do \
		echo "Running $$f..."; \
		clickhouse-client --multiquery < "$$f" 2>/dev/null || true; \
	done
	@echo "Done."

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
