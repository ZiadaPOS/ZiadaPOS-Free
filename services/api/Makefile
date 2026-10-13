# ZiadaPOS Go API
#
# The whole project tree mirrors the old Django layout: every domain lives in
# apps/ as its own package (models, repository, service, handlers, routes).
# Shared infrastructure lives in internal/.

# ── Build ────────────────────────────────────────────────────────────────────
.PHONY: build run dev test migrate tidy docker-up docker-down

build:
	go build -o bin/api ./cmd/api

run:
	go run ./cmd/api

dev:
	go run -tags dev ./cmd/api

test:
	go test ./...

tidy:
	go mod tidy

migrate:
	go run ./cmd/api -migrate

# ── Docker (local infra: postgres + redis + minio) ──────────────────────────
docker-up:
	docker compose up -d

docker-down:
	docker compose down

# ── Lint (install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest) ──
lint:
	golangci-lint run ./...
