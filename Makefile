SHELL := /bin/sh

.PHONY: dev install generate migrate test test-race lint typecheck build verify compose-config

dev:
	docker compose -f infra/compose/dev.yml up -d postgres minio
	bun run dev

install:
	bun install --frozen-lockfile

generate:
	cd services/api && GOCACHE="$$PWD/.gocache" go generate ./...
	bun run openapi:check

migrate:
	docker compose -f infra/compose/dev.yml run --rm migrate

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
