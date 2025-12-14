# Makefile for rasp-service-backend
# Useful targets: build, run, test, fmt, vet, lint, docker, compose

# Configuration
GO ?= go
PKG := ./cmd/app
BINARY := rasp-service
OUT_DIR := bin

# On Windows produce .exe and make OUT include it so build/run match
ifeq ($(OS),Windows_NT)
	EXE_EXT := .exe
else
	EXE_EXT :=
endif

OUT := $(OUT_DIR)/$(BINARY)$(EXE_EXT)
LDFLAGS := -s -w
IMAGE ?= $(BINARY)
TAG ?= latest

# PostgreSQL client settings (used by migration targets)
PSQL ?= psql
PSQL_HOST ?= localhost
PSQL_PORT ?= 5432
PSQL_USER ?= postgres
PSQL_DB ?= postgres
PSQL_PASS ?=

.PHONY: all help build run clean test fmt vet lint deps modtidy docker-build docker-push compose-up compose-down migrate-up migrate-down install-tools start start-local migrate-all

help:

	@echo "  build         Собрать бинарник в ./bin"
	@echo "  run           Собрать и запустить локально"
	@echo "  test          Запустить unit-тесты"
	@echo "  modtidy       Выполнить 'go mod tidy'"
	@echo "  deps          Скачать модули"
	@echo "  clean         Удалить bin/ и временные файлы"
	@echo "  docker-build  Собрать docker-образ"
	@echo "  compose-up    docker-compose up -d --build"
	@echo "  compose-down  docker-compose down"
	@echo "  migrate-all   Запустить Postgres и применить все SQL-миграции"
	@echo "  start-local   Поднять сервисы, применить миграции и запустить локально"
	@echo
	@echo "Переменные (можно переопределить): IMAGE, TAG, PSQL_*"

ifeq ($(OS),Windows_NT)
$(OUT_DIR):
	@powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path '$(OUT_DIR)' | Out-Null"
else
$(OUT_DIR):
	@mkdir -p $(OUT_DIR)
endif

build: $(OUT_DIR)
	@echo "Building $(BINARY) -> $(OUT)"
	$(GO) build -ldflags "$(LDFLAGS)" -o $(OUT) $(PKG)

run: build
	@echo "Running $(OUT)"
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "& '$(OUT)'"
else
	@$(OUT)
endif

modtidy:
	$(GO) mod tidy

deps:
	$(GO) mod download

clean:
	@echo "Cleaning..."
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "Remove-Item -Recurse -Force -ErrorAction SilentlyContinue '$(OUT_DIR)'"
else
	@rm -rf $(OUT_DIR)
endif


compose-up:
	@echo "Starting services via docker-compose"
	docker-compose up -d --build

compose-down:
	@echo "Stopping services via docker-compose"
	docker-compose down

install-tools:
	@echo "Installing developer tools (golangci-lint)"
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONY: migrate-all

migrate-all:
	@echo "Starting postgres service..."
	docker-compose up -d --build
	@echo "Waiting for Postgres to become available (container: rasp-service-postgres)..."
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "& { $$i=0; while ($$i -lt 60) { docker exec rasp-service-postgres pg_isready -U admin -d rasp_db -p 5432 > $$null 2>$$null; if ($$LASTEXITCODE -eq 0) { break } ; Start-Sleep -Seconds 1; $$i++; Write-Host \"waiting... ($$i)\" } ; if ($$i -ge 60) { Write-Error 'Postgres did not become ready in time'; exit 1 } }"
	@echo "Applying migrations..."
	@powershell -NoProfile -Command "& { Get-Content -Raw 'internal/storage/migrations/1_init.up.sql' | docker exec -i rasp-service-postgres psql -U admin -d rasp_db -f - }"
	@echo "Migrations applied."


.PHONY: start start-local

start: deps modtidy migrate-all build
	@echo "Starting local binary after migrations"
	@$(MAKE) run
else
	@i=0; until docker exec rasp-service-postgres pg_isready -U admin -d rasp_db -p 5432 >/dev/null 2>&1 || [ $$i -ge 60 ]; do i=$$((i+1)); sleep 1; echo "waiting... ($$i)"; done; \
	if [ $$i -ge 60 ]; then echo "Postgres did not become ready in time"; exit 1; fi
	@echo "Applying migrations..."
	@docker exec -i rasp-service-postgres psql -U admin -d rasp_db -f - < internal/storage/migrations/1_init.up.sql
	@echo "Migrations applied."
endif

.DEFAULT_GOAL := help