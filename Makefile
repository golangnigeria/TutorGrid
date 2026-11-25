.PHONY: db-up db-down migrate-up migrate-down test-unit test-integration vet build web-lint web-typecheck web-test web-build

COMPOSE_FILE := deployments/docker-compose.dev.yml

db-up:
	docker compose -f $(COMPOSE_FILE) up -d

db-down:
	docker compose -f $(COMPOSE_FILE) down

migrate-up:
	go run ./cmd/migrate -direction=up

migrate-down:
	go run ./cmd/migrate -direction=down

test-unit:
	go test -race -short ./...

test-integration:
	go test -race -v ./internal/platform/database/...

vet:
	go vet ./...

build:
	go build ./cmd/api ./cmd/migrate

web-lint:
	cd web && npm run lint

web-typecheck:
	cd web && npm run typecheck

web-test:
	cd web && npm test

web-build:
	cd web && npm run build
