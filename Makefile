SHELL := /bin/sh

.PHONY: start stop restart dev install generate migrate configure-local-storage test test-race lint typecheck build verify compose-config

ENV_FILE ?= .env.local
COMPOSE_FILE ?= infra/compose/dev.yml
COMPOSE = docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE)
APP_SERVICES = api worker renderer web

start:
	test -f "$(ENV_FILE)"
	$(COMPOSE) up -d --wait $(APP_SERVICES)

stop:
	$(COMPOSE) down --remove-orphans

restart:
	test -f "$(ENV_FILE)"
	$(COMPOSE) down --remove-orphans
	$(COMPOSE) build api worker renderer web river-migrate
	$(COMPOSE) up -d --wait $(APP_SERVICES)

dev:
	$(COMPOSE) up -d postgres minio
	bun run dev

install:
	bun install --frozen-lockfile

generate:
	cd services/api && GOCACHE="$$PWD/.gocache" go generate ./...
	bun run openapi:check

migrate:
	$(COMPOSE) run --rm migrate

configure-local-storage:
	test -n "$(CLIENT_ID)"
	$(COMPOSE) exec -T api /app/configure-local-storage "$(CLIENT_ID)"

test:
	cd services/api && GOCACHE="$$PWD/.gocache" go test ./...
	bun run test

test-race:
	cd services/api && GOCACHE="$$PWD/.gocache" go test -race ./...

lint:
	gofmt -l services/api | tee /tmp/studio-gofmt-files
	test ! -s /tmp/studio-gofmt-files
	bun run lint

typecheck:
	bun run typecheck

build:
	bun run build
	cd services/api && GOCACHE="$$PWD/.gocache" go build ./cmd/...

compose-config:
	docker compose -f infra/compose/dev.yml config --quiet
	docker compose -f infra/compose/prod.yml config --quiet

verify: lint typecheck test build compose-config
