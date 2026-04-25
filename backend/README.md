# yulik3d backend

Go-бэкенд интернет-магазина yulik3d. Чистая архитектура, `net/http` stdlib, сырой SQL через pgx, сессии в Redis, картинки в MinIO, миграции goose (встроенные в бинарник), Swagger UI.

## Быстрый старт (docker-compose)

Из корня проекта:

```bash
cp .env.example .env   # при необходимости подправить пароли
docker compose up -d --build
```

После старта:

- **API:** http://localhost:8080/api/v1
- **Swagger UI:** http://localhost:8080/swagger/
- **Health:** http://localhost:8080/api/v1/health
- **MinIO Console:** http://localhost:9001 (логин/пароль из `.env`)
- **Postgres:** localhost:5432 (юзер/пароль/БД из `.env`)
- **Redis:** localhost:6379 (пароль из `.env`)

Миграции накатываются автоматически при старте бэка (goose из встроенных `migrations/*.sql`).

## Сделать пользователя админом

Роль `admin` выдаётся вручную из БД:

```sql
UPDATE "user" SET role = 'admin' WHERE email = 'you@example.com';
```

После этого пользователь должен перелогиниться, чтобы роль попала в сессию.

## Локальная разработка (без docker)

Убедись, что Postgres / Redis / MinIO запущены локально (можно через `docker compose up -d postgres redis minio minio-init`), затем:

```bash
cd backend
go mod tidy            # скачать зависимости
make swagger           # сгенерировать реальную Swagger-спеку (требует swag CLI)
make run               # запустить бэкенд
```

### Установить swag CLI

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

Бинарник кладётся в `$GOPATH/bin` (обычно `~/go/bin`). Добавь в PATH.

До первого запуска `make swagger` Swagger UI покажет пустую спеку из stub. После `make swagger` — все 41 эндпоинт с примерами.

## Архитектура

- `cmd/main.go` — точка входа, DI, graceful shutdown.
- `cmd/routes.go` — регистрация маршрутов.
- `cmd/swagger.go` — Swagger UI.
- `config/config.go` — загрузка env.
- `internal/model/` — entities, DTO, доменные ошибки.
- `internal/repository/` — сырой SQL, TxManager, session-store на Redis.
- `internal/service/` — бизнес-логика.
- `internal/middleware/` — auth (через сессии), CORS, recover, logging, error-формат, RequireAuth/RequireRole/RejectAuthed.
- `internal/handler/` — HTTP-хэндлеры со swagger-аннотациями.
- `internal/generated/docs/` — сгенерированный swag'ом spec (stub закоммичен).
- `pkg/logger` — slog JSON.
- `pkg/passwordhash` — argon2id.
- `pkg/cookie` — установка/чтение session cookie.
- `migrations/` — SQL-миграции goose, встраиваются в бинарник через `//go:embed`.

## Auth-flow в Swagger UI

1. Открой Swagger UI.
2. Выполни `/auth/register` или `/auth/login` — в ответе сервер пришлёт `Set-Cookie: session=...`. Браузер сохранит cookie автоматически (same-origin).
3. Все последующие запросы из Swagger UI браузер будет слать с этой cookie. Отдельно «Authorize» нажимать не надо.
4. Выйти — `/auth/logout`.

## Формат ошибок

Все ошибки — единый JSON:

```json
{
  "statusCode": 400,
  "url": "/api/v1/items/018f...",
  "message": "Item not found",
  "date": "2026-04-23T18:42:00Z"
}
```

Для 5xx `message` всегда `"Internal server error"` — детали только в логах.

## Полезные команды

```bash
make run         # запустить локально
make build       # собрать бинарник bin/yulik3d
make test        # тесты (позже добавятся)
make lint        # go vet
make tidy        # go mod tidy
make swagger     # сгенерировать swagger-спеку
make migrate-up  # накатить миграции вручную (goose)
make migrate-down
```
