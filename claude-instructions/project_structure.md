# Структура репозитория

Корень проекта: `yulik3d/yulik3d_project/`. Всё разрабатываем только в ней. Черновой фронт в соседней папке `yulik3d_frontend/` — вне этого проекта, используется только как визуальный/структурный референс при разработке фронта.

## Общее дерево

```
yulik3d_project/
├── CLAUDE.md                       # Главный файл для Claude: описание проекта + ссылки
├── README.md                       # Пользовательский README (минимальный)
├── claude-instructions/            # Инструкции и документация для Claude
│   ├── db_description.md           #   Полная схема БД (PostgreSQL)
│   └── project_structure.md        #   Этот файл
│
├── backend/                        # Go-бэкенд, чистая архитектура, без фреймворков
│   ├── cmd/
│   │   ├── main.go                 # Точка входа: DI + регистрация роутов
│   │   └── swagger.go              # Swagger UI + встроенная OpenAPI-спека
│   ├── config/
│   │   └── config.go               # Чтение env-переменных в структуру
│   ├── internal/                   # Прикладной код (недоступен извне Go-модуля)
│   │   ├── handler/                #   Слой транспорта: HTTP-хэндлеры
│   │   │                           #   (файлы добавляются по доменам, по мере реализации)
│   │   ├── service/                #   Бизнес-логика, валидация
│   │   │                           #   (файлы добавляются по доменам)
│   │   ├── repository/             #   SQL-запросы к PostgreSQL
│   │   │   ├── tx.go               #     TxManager: атомарные операции внутри транзакции
│   │   │   └── ...                 #     остальные файлы по доменам
│   │   ├── model/                  #   Структуры данных (entities, DTO), доменные ошибки
│   │   │   ├── errors.go           #     доменные ошибки: ErrNotFound, ErrConflict, …
│   │   │   └── ...                 #     <domain>.go (entity) + <domain>_dto.go (wire-format)
│   │   └── middleware/             #   Сквозная логика HTTP
│   │       ├── auth.go             #     проверка сессии (Redis), user_id и role в ctx
│   │       ├── cors.go             #     CORS-заголовки
│   │       ├── error.go            #     универсальный обработчик ошибок + JSON-хелпер
│   │       ├── logging.go          #     request-id + structured access-log (slog)
│   │       └── recover.go          #     ловит паники, отдаёт 500, не роняет процесс
│   ├── pkg/                        # Утилиты без бизнес-логики (потенциально переносимые)
│   │   ├── logger/                 #   slog-обёртка, JSON-формат, конструктор логгера
│   │   ├── passwordhash/           #   argon2id: Hash + Verify
│   │   └── cookie/                 #   построение/чтение сессионного cookie
│   ├── docs/                       # API-документация (AsciiDoc)
│   │   ├── index.adoc              #   корневой список всех методов
│   │   ├── _template/              #   шаблон для нового метода (копируется и переименовывается)
│   │   │   ├── _template.adoc
│   │   │   ├── request.adoc
│   │   │   ├── response-success.adoc
│   │   │   ├── response-errors.adoc
│   │   │   └── diagram.puml
│   │   └── {endpoint-name}/        #   одна папка на один метод (создаются по ходу)
│   │       ├── {endpoint-name}.adoc      # главный файл (разделы 1–4)
│   │       ├── request.adoc              # пример запроса
│   │       ├── response-success.adoc     # пример успешного ответа
│   │       ├── response-errors.adoc      # все возможные ошибки
│   │       └── diagram.puml              # PlantUML-диаграмма алгоритма
│   ├── migrations/                 # SQL-миграции БД, нумерация 001_*.sql, 002_*.sql, …
│   ├── Dockerfile                  # multi-stage build + slim runtime
│   ├── Makefile                    # шорткаты: run, build, test, lint, migrate-*
│   ├── go.mod                      # (создаётся `go mod init` на этапе бэка)
│   └── go.sum
│
├── frontend/                       # Фронтенд (SPA). Структуру детализируем позже.
│   └── Dockerfile                  # (добавится позже)
│
├── docker-compose.yml              # Оркестрация всех сервисов разом
├── .env                            # Реальные значения (в .gitignore)
├── .env.example                    # Шаблон переменных, коммитится
├── .gitignore
└── .dockerignore
```

В слоях `handler/ service/ repository/ model/` файлы именуются по доменам (например, `auth.go`, `item.go`, `order.go`) и создаются по мере реализации — в дереве выше они не перечислены, чтобы структура не устаревала. Доменный срез вытекает из таблиц БД (см. [db_description.md](db_description.md)): auth/user, item, picture, option, category, favorite, order, admin.

---

## Бэкенд: принципы организации

### Слои чистой архитектуры

Зависимости идут **внутрь**: `handler → service → repository → model`. Обратных импортов нет. `model` не зависит ни от кого.

| Слой | Ответственность | НЕ делает |
|---|---|---|
| `handler/` | Парсинг HTTP-запроса, валидация формата, вызов сервиса, формирование HTTP-ответа | Не содержит бизнес-логики, не лезет в БД |
| `service/` | Бизнес-логика: бизнес-правила, согласованность между сущностями, оркестрация нескольких репозиториев, валидация | Ничего не знает про HTTP, ничего не знает про конкретный SQL |
| `repository/` | Только SQL-запросы к PostgreSQL, маппинг строк на модели | Ничего не знает про HTTP, не содержит бизнес-правил |
| `model/` | Структуры (entities, DTO, value objects), доменные ошибки | Без зависимостей от других слоёв проекта |
| `middleware/` | Сквозная логика HTTP: auth, CORS, recover, logging, JSON-хелперы, обработчик ошибок | Без бизнес-логики |
| `config/` | Чтение env, валидация, структура `Config` | — |
| `pkg/` | Переиспользуемые утилиты без бизнес-логики (logger, passwordhash, cookie) | Без зависимостей на `internal/` |

### Обязательные практики (production quality)

Эти правила применяются при разработке — структура их поддерживает, но не навязывает.

1. **Dependency Inversion: интерфейсы на стороне потребителя.** Сервис объявляет интерфейс репозитория, который ему нужен; репозиторий — конкретный тип, ему удовлетворяющий. Хэндлер объявляет интерфейс сервиса. Это обеспечивает тестируемость (моки) и изоляцию слоёв.
2. **`context.Context` — первым аргументом** во всех публичных методах `service/` и `repository/`.
3. **Транзакции через `TxManager`.** Интерфейс в `repository/tx.go` с методом `Run(ctx, fn)`. Сервис оркестрирует, репозитории получают `*pgx.Tx` из контекста через helper. Никогда не прокидывать `*pgx.Tx` явно через сигнатуры.
4. **Политика ошибок.** Доменные ошибки — в `model/errors.go`. Репозиторий транслирует ошибки PostgreSQL (`pgconn.PgError`) в доменные через `%w`. Middleware `error.go` мапит домен в HTTP-код + унифицированный JSON.
5. **Entity vs DTO раздельно.** `model/<domain>.go` — entity из БД. `model/<domain>_dto.go` — wire-format для API. Entity никогда не сериализуется в JSON напрямую (утечки `PasswordHash` и т.п.).
6. **Структурированные логи** — `log/slog` (stdlib), JSON, уровни. Логгер инжектируется через конструктор, не глобальный.
7. **Пароли** — argon2id (`pkg/passwordhash/`).
8. **SQL — сырой через `pgx`**, без ORM.
9. **Роутер** — только `net/http` из stdlib (с Go 1.22+ `ServeMux` достаточно). Не chi/gorilla/mux/Echo/Gin/Fiber.

### Стек

- **Язык:** Go, без веб-фреймворков. Роутер — **только `net/http` из stdlib** (с Go 1.22+ `ServeMux` поддерживает паттерны `GET /items/{id}` и `r.PathValue("id")`). Не chi, не gorilla/mux, не Echo / Gin / Fiber / Fx.
- **SQL:** сырой SQL (`database/sql` + `pgx`). ORM не используем.
- **Миграции:** инструмент выберем на этапе бэка (кандидаты: `goose`, `golang-migrate`, `atlas`). Файлы — в `backend/migrations/`, нумерация `001_*.sql`, `002_*.sql`, …
- **Логи / конфиг / auth** — реализуем ручками, с минимумом зависимостей.
- **OpenAPI / Swagger UI:** подключаем в `cmd/swagger.go`. Спека — встроенная в бинарник.

### Точка входа

`backend/cmd/main.go` — единственная точка входа на HTTP-сервер. Внутри: load config → open DB → open MinIO client → construct repositories → construct services → construct handlers → attach middleware → register routes → start server. Никакой глобальщины.

Если понадобится второй бинарник — добавим новую папку в `cmd/`.

### Папка `internal/`

Код в `internal/` недоступен для импорта извне модуля (правило Go). Защита от превращения прикладных структур в публичный API проекта по ошибке.

### Папка `config/` (вне `internal/`)

Расположена на уровне `cmd/` и `internal/`. Содержит структуру `Config` и функцию её загрузки из env. Вынесена из `internal/`, чтобы `cmd/main.go` мог подключать её без циркулярных зависимостей через прикладные слои.

---

## Документация API: `backend/docs/`

Формат AsciiDoc. Одна папка на метод. Шаблон для копирования — `docs/_template/`.

```
docs/
├── index.adoc                          # корневой список всех методов (навигация)
├── _template/                          # скелет для нового метода — копируется и переименовывается
│   ├── _template.adoc                  #   главный файл, разделы 1–4:
│   │                                   #     1. Описание
│   │                                   #     2. Контракт (URL, метод, заголовки)
│   │                                   #     3. Авторизация / роли
│   │                                   #     4. Ссылки на остальные файлы
│   ├── request.adoc                    #   пример запроса
│   ├── response-success.adoc           #   пример успешного ответа
│   ├── response-errors.adoc            #   все возможные ошибки и их коды
│   └── diagram.puml                    #   PlantUML-диаграмма алгоритма
└── <endpoint-name>/                    # имя метода в kebab-case
    ├── <endpoint-name>.adoc
    ├── request.adoc
    ├── response-success.adoc
    ├── response-errors.adoc
    └── diagram.puml
```

Точное содержимое и стиль файлов согласуем на этапе реализации — пользователь покажет примеры.

---

## Frontend

Структуру детализируем позже, когда перейдём к разработке фронта. За визуальный и логический референс берём черновой фронт в соседней папке `yulik3d_frontend/` (вне этого проекта).

---

## Docker

- **`docker-compose.yml`** в корне — оркестрирует все сервисы одной командой. Планируемые сервисы:
  - `postgres` — PostgreSQL.
  - `minio` — объектное хранилище картинок.
  - `backend` — Go-сервис (собирается из `backend/Dockerfile`).
  - `frontend` — SPA (собирается из `frontend/Dockerfile`), добавится позже.
  - Волюмы для данных postgres и minio.
- **Разные `Dockerfile`** для бэка и фронта, каждый в своей папке.
- **Один `.env`** в корне — docker-compose читает его и прокидывает нужные переменные в соответствующие сервисы.

---

## Окружение: `.env` и `.env.example`

- **`.env`** — в корне, с реальными значениями. **В `.gitignore`.** Никогда не коммитится.
- **`.env.example`** — в корне, тот же набор ключей со значениями-плейсхолдерами. Коммитится. При клонировании: `cp .env.example .env` и заполнить.
- Переменные делим по префиксам: `POSTGRES_*`, `MINIO_*`, `BACKEND_*`, `FRONTEND_*`, общие (например, `PUBLIC_URL`).
