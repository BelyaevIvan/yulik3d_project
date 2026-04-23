.PHONY: help run build test lint tidy swagger migrate-up migrate-down

help:
	@echo "run           - запустить бэкенд локально (без docker)"
	@echo "build         - собрать бинарник в bin/yulik3d"
	@echo "test          - запустить тесты"
	@echo "lint          - go vet"
	@echo "tidy          - go mod tidy"
	@echo "swagger       - сгенерировать swagger-спеку (swag init)"
	@echo "migrate-up    - накатить миграции (goose)"
	@echo "migrate-down  - откатить последнюю миграцию"

run:
	go run ./cmd

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/yulik3d ./cmd

test:
	go test ./...

lint:
	go vet ./...

tidy:
	go mod tidy

swagger:
	swag init -g cmd/main.go --output internal/generated/docs --parseInternal --parseDependency

MIGRATE_DSN=postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=$(POSTGRES_SSLMODE)

migrate-up:
	goose -dir migrations postgres "$(MIGRATE_DSN)" up

migrate-down:
	goose -dir migrations postgres "$(MIGRATE_DSN)" down
