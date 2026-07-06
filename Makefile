-include .env

DB_HOST ?= 127.0.0.1
DB_PORT ?= 3306
DB_USER ?= app
DB_PASSWORD ?= app_password
DB_NAME ?= team_task
DB_DSN = $(DB_USER):$(DB_PASSWORD)@tcp($(DB_HOST):$(DB_PORT))/$(DB_NAME)?parseTime=true

.PHONY: run down build migrate-up migrate-down migrate-status db-conn test test-unit test-integration lint swagger

run: ## bring up mysql/redis, apply migrations, then build and start the app
	docker compose up -d --wait mysql redis
	$(MAKE) migrate-up
	docker compose up --build -d app

down: ## stop all services and remove volumes
	docker compose down -v

build: ## go build -o bin/api ./cmd/api
	go build -o bin/api ./cmd/api

migrate-up: ## apply all pending migrations
	goose -dir migrations mysql "$(DB_DSN)" up

migrate-down: ## roll back the last migration
	goose -dir migrations mysql "$(DB_DSN)" down

migrate-status: ## show applied/pending migrations
	goose -dir migrations mysql "$(DB_DSN)" status

db-conn: ## connect to the local DB via the mysql CLI
	mysql -h $(DB_HOST) -P $(DB_PORT) -u $(DB_USER) -p$(DB_PASSWORD) $(DB_NAME)

test-unit: ## run unit tests only
	go test ./... -race -count=1

test-integration: ## run integration tests (requires local Docker for testcontainers-go)
	go test ./internal/repository/... -tags=integration -race -count=1

test: test-unit test-integration ## run unit AND integration tests

lint: ## golangci-lint run ./...
	golangci-lint run ./...

swagger: ## regenerate the Swagger spec from handler annotations
	swag init -g cmd/api/main.go -o docs
