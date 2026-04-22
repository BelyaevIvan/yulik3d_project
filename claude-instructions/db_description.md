# Структура БД (PostgreSQL)

Документ описывает схему БД проекта **yulik3d**. Всё DDL ниже — канонический источник истины для моделей бэкенда и миграций.

## Общие соглашения

- **СУБД:** PostgreSQL.
- **Все первичные и внешние ключи** — `UUID` (тип `uuid` в PostgreSQL). Генерация — на стороне Go: **UUIDv7** через `github.com/google/uuid` (`uuid.NewV7()`). На уровне БД `DEFAULT` не ставим, приложение всегда передаёт ID явно. UUIDv7 time-ordered, что даёт хорошую B-tree locality при вставках, плюс отсутствие enumeration-атак на публичные URL (`/api/v1/items/:id`, `/api/v1/orders/:id`).
- **Все временны́е поля** — `TIMESTAMPTZ` с `DEFAULT NOW()`. Поля `created_at` / `updated_at` заполняются бэком / триггерами.
- **Цены** хранятся в **рублях** (целые числа, `INT`). Копейки не используются.
- **`sale`** — скидка в **процентах** (целое число `0..100`). `0` означает отсутствие скидки. Итоговая цена считается на бэке / фронте как `round(price * (100 - sale) / 100)`.
- **Junction-таблицы** имеют составной PRIMARY KEY по паре FK (если не указано иное).
- **Каскады:**
  - `ON DELETE CASCADE` — для связующих записей (junction) и для подчинённых сущностей, которые теряют смысл без родителя.
  - `ON DELETE RESTRICT` — для защиты исторических данных (заказы и их позиции, используемые типы опций).
  - Товары физически **не удаляются** — скрытие через `item.hidden`. `RESTRICT` на `order_item.item_id` — страховка.
- **Снапшоты в заказах** — все данные заказа (цены, названия, артикулы, опции и их значения) хранятся **копиями текстом/числом** на момент создания заказа. Это гарантирует, что последующие изменения товаров и опций не сломают историю.
- **Картинки:** в `picture.object_key` хранится **относительный путь внутри бакета MinIO** (например, `items/018f7d3e-.../main.jpg`). Полный URL собирает бэк при отдаче API (`PUBLIC_MINIO_URL + bucket + object_key`). Для MVP бакет с публичным read; позже можно перейти на presigned URL без миграций данных.

---

## ENUM-типы

```sql
CREATE TYPE category_type AS ENUM ('figure', 'other');

CREATE TYPE user_role AS ENUM ('user', 'admin');

CREATE TYPE order_status AS ENUM (
  'created',        -- Создан (ставится автоматически при создании)
  'confirmed',      -- Подтверждён
  'manufacturing',  -- На изготовлении
  'delivering',     -- В доставке
  'completed',      -- Завершён
  'cancelled'       -- Отменён (может быть выставлен из любого статуса)
);
```

Переходы между статусами идут только вперёд по списку (`created → confirmed → manufacturing → delivering → completed`). `cancelled` — выставляется вручную из любого предыдущего. Валидация переходов — на бэке. Никакой автоматической логики на статусах не завязываем: админ двигает вручную в админке, это просто инфо для покупателя.

> **`option_type`** специально сделан **таблицей**, а не ENUM — чтобы админ мог добавлять новые типы опций из UI обычным `INSERT`, без `ALTER TYPE`.

---

## Товары и картинки

### `item` — товары

| Поле | Тип | Ограничения | Примечание |
|---|---|---|---|
| `id` | UUID | PK | |
| `name` | TEXT | NOT NULL | Название (редактируется админом) |
| `description_info` | TEXT | NOT NULL | Markdown. Блок «Информация о товаре». Подзаголовки («Технология:», «Материал:», …) — жирным, значения после них пишет админ |
| `description_other` | TEXT | NOT NULL | Markdown (маркированный список). Блок «Особенности». Формируется из динамических инпутов «добавить особенность» в админке |
| `price` | INT | NOT NULL | Базовая цена в рублях |
| `sale` | INT | NOT NULL, DEFAULT 0, CHECK (`sale BETWEEN 0 AND 100`) | Скидка в процентах (0 = нет скидки, 100 = бесплатно) |
| `articul` | TEXT | NOT NULL, UNIQUE | Генерируется на бэке, неизменяемое поле |
| `hidden` | BOOLEAN | NOT NULL, DEFAULT FALSE | `TRUE` — товар скрыт из общего каталога, но доступен по прямой ссылке с пометкой «Не доступно к заказу» |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |
| `updated_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |

### `picture` — картинки

| Поле | Тип | Ограничения | Примечание |
|---|---|---|---|
| `id` | UUID | PK | |
| `object_key` | TEXT | NOT NULL | Относительный путь в MinIO-бакете (например, `items/018f7d3e-.../main.jpg`) |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |
| `updated_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |

### `item_picture` — связь товаров и картинок (M-M)

| Поле | Тип | Ограничения |
|---|---|---|
| `item_id` | UUID | NOT NULL, FK → `item(id)` ON DELETE CASCADE |
| `picture_id` | UUID | NOT NULL, FK → `picture(id)` ON DELETE CASCADE |
| `position` | INT | NOT NULL. `1` = титульная, далее по порядку отображения в галерее |
| **PK** | | `(item_id, picture_id)` |

---

## Опции

### `option_type` — справочник типов опций

Админ может добавлять новые типы через UI (обычный `INSERT`).

| Поле | Тип | Ограничения | Примечание |
|---|---|---|---|
| `id` | UUID | PK | |
| `code` | TEXT | NOT NULL, UNIQUE | Внутренний ключ: `size`, `paint`, `engraving`, … |
| `label` | TEXT | NOT NULL | Отображаемое имя для UI: `Размер`, `Покраска`, `Гравировка` |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |
| `updated_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |

### `item_option` — опции конкретного товара

Опция привязана к товару — поэтому размер `M` у разных товаров может иметь разную цену. Цена хранится здесь, а не в `option_type`.

| Поле | Тип | Ограничения | Примечание |
|---|---|---|---|
| `id` | UUID | PK | |
| `item_id` | UUID | NOT NULL, FK → `item(id)` ON DELETE CASCADE | |
| `type_id` | UUID | NOT NULL, FK → `option_type(id)` ON DELETE RESTRICT | Тип нельзя удалить, пока он используется |
| `value` | TEXT | NOT NULL | `S` / `M` / `L`; `Да` / `Нет`; и т.п. |
| `price` | INT | NOT NULL, DEFAULT 0 | Доплата в рублях. `0` — для дефолтных/«Нет» |
| `position` | INT | NOT NULL, DEFAULT 0 | Порядок в UI внутри одного типа |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |
| `updated_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |
| **UNIQUE** | | `(item_id, type_id, value)` — одна и та же комбинация значений не дублируется |

Группировка значений по типу для UI (размер: S/M/L вместе) — делается на бэке через `GROUP BY type_id`.

---

## Категории

### `category` — категории

| Поле | Тип | Ограничения | Примечание |
|---|---|---|---|
| `id` | UUID | PK | |
| `name` | TEXT | NOT NULL | «Игры», «Фильмы», «Декор», «Кастомизация ПК», … |
| `type` | `category_type` | NOT NULL | `figure` (фигурки) или `other` (макеты) |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |
| `updated_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |

### `subcategory` — подкатегории

| Поле | Тип | Ограничения | Примечание |
|---|---|---|---|
| `id` | UUID | PK | |
| `name` | TEXT | NOT NULL | «Dota 2», «Вазы», «Горшки» |
| `category_id` | UUID | NOT NULL, FK → `category(id)` ON DELETE CASCADE | Удаление категории удаляет её подкатегории |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |
| `updated_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |

### `item_subcategory` — связь товаров и подкатегорий (M-M)

| Поле | Тип | Ограничения |
|---|---|---|
| `item_id` | UUID | NOT NULL, FK → `item(id)` ON DELETE CASCADE |
| `subcategory_id` | UUID | NOT NULL, FK → `subcategory(id)` ON DELETE CASCADE |
| **PK** | | `(item_id, subcategory_id)` |

Товар может относиться к нескольким подкатегориям, в том числе из разных категорий (например, одновременно «Фильмы» и «Коллекционирование»). Принадлежность товара к категории выводится через `JOIN subcategory`.

---

## Пользователи и избранное

### `"user"` — пользователи

> Имя таблицы заключено в двойные кавычки, т.к. `user` — зарезервированное слово PostgreSQL.

| Поле | Тип | Ограничения | Примечание |
|---|---|---|---|
| `id` | UUID | PK | |
| `email` | TEXT | NOT NULL, UNIQUE | |
| `password_hash` | TEXT | NOT NULL | Хэш (bcrypt / argon2 — решим на этапе бэка) |
| `full_name` | TEXT | NOT NULL | |
| `phone` | TEXT | nullable | Можно заполнить позже в профиле |
| `role` | `user_role` | NOT NULL, DEFAULT `'user'` | `admin` выдаётся вручную `UPDATE`-ом в БД |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |
| `updated_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |

### `favorite` — избранное (M-M)

| Поле | Тип | Ограничения |
|---|---|---|
| `user_id` | UUID | NOT NULL, FK → `"user"(id)` ON DELETE CASCADE |
| `item_id` | UUID | NOT NULL, FK → `item(id)` ON DELETE CASCADE |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |
| **PK** | | `(user_id, item_id)` |

---

## Заказы

### Правила

- Заказ оформить может **только авторизованный пользователь** (`order.user_id NOT NULL`). Добавление в корзину — тоже только для авторизованных (логика фронта).
- Корзина хранится **во фронте** (localStorage), в БД не попадает. При оформлении заказа фронт отправляет её содержимое в бэк.
- **Все данные заказа — снапшоты.** Цены, названия, артикулы, опции, значения опций, метки типов — копии на момент создания. Никаких связей с `item_option` / `option_type` нет (только `order_item.item_id` для справки, с `ON DELETE RESTRICT`).

### `"order"` — заказы

> Имя таблицы заключено в двойные кавычки, т.к. `order` — зарезервированное слово PostgreSQL.

| Поле | Тип | Ограничения | Примечание |
|---|---|---|---|
| `id` | UUID | PK | |
| `user_id` | UUID | NOT NULL, FK → `"user"(id)` ON DELETE RESTRICT | Нельзя удалить пользователя с заказами |
| `status` | `order_status` | NOT NULL, DEFAULT `'created'` | |
| `total_price` | INT | NOT NULL | Итоговая сумма на момент создания, в рублях |
| `customer_comment` | TEXT | nullable | Комментарий от покупателя (если предусмотрим поле) |
| `admin_note` | TEXT | nullable | Внутренняя пометка админа |
| `contact_phone` | TEXT | NOT NULL | **Снапшот** — юзер мог поменять свой phone после |
| `contact_full_name` | TEXT | NOT NULL | **Снапшот** — аналогично |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |
| `updated_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |

### `order_item` — позиции заказа

| Поле | Тип | Ограничения | Примечание |
|---|---|---|---|
| `id` | UUID | PK | |
| `order_id` | UUID | NOT NULL, FK → `"order"(id)` ON DELETE CASCADE | |
| `item_id` | UUID | NOT NULL, FK → `item(id)` ON DELETE RESTRICT | Товары не удаляются; `RESTRICT` — страховка |
| `quantity` | INT | NOT NULL | |
| `unit_base_price` | INT | NOT NULL | **Снапшот** базовой цены товара на момент заказа |
| `unit_total_price` | INT | NOT NULL | **Снапшот** цены с учётом опций (= `unit_base_price` + сумма `price_snapshot` опций) |
| `item_name_snapshot` | TEXT | NOT NULL | |
| `item_articul_snapshot` | TEXT | NOT NULL | |

### `order_item_option` — выбранные опции позиции

**Важно:** связей с `item_option` и `option_type` **нет**. Всё — текст/число. Так история заказа не зависит ни от правок опций, ни от удаления типов.

| Поле | Тип | Ограничения | Примечание |
|---|---|---|---|
| `id` | UUID | PK | |
| `order_item_id` | UUID | NOT NULL, FK → `order_item(id)` ON DELETE CASCADE | |
| `type_code_snapshot` | TEXT | NOT NULL | Снапшот `option_type.code`: `size`, `engraving`, … |
| `type_label_snapshot` | TEXT | NOT NULL | Снапшот `option_type.label`: `Размер`, `Гравировка`, … |
| `value_snapshot` | TEXT | NOT NULL | Снапшот `item_option.value`: `M`, `Да`, … |
| `price_snapshot` | INT | NOT NULL | Снапшот доплаты за опцию |

---

## Полный DDL

```sql
-- ============================================================
-- ENUM TYPES
-- ============================================================

CREATE TYPE category_type AS ENUM ('figure', 'other');

CREATE TYPE user_role AS ENUM ('user', 'admin');

CREATE TYPE order_status AS ENUM (
  'created', 'confirmed', 'manufacturing', 'delivering', 'completed', 'cancelled'
);

-- ============================================================
-- ITEMS & PICTURES
-- ============================================================

CREATE TABLE item (
  id                  UUID PRIMARY KEY,
  name                TEXT NOT NULL,
  description_info    TEXT NOT NULL,
  description_other   TEXT NOT NULL,
  price               INT NOT NULL,
  sale                INT NOT NULL DEFAULT 0 CHECK (sale BETWEEN 0 AND 100),
  articul             TEXT NOT NULL UNIQUE,
  hidden              BOOLEAN NOT NULL DEFAULT FALSE,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE picture (
  id                  UUID PRIMARY KEY,
  object_key          TEXT NOT NULL,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE item_picture (
  item_id             UUID NOT NULL REFERENCES item(id)    ON DELETE CASCADE,
  picture_id          UUID NOT NULL REFERENCES picture(id) ON DELETE CASCADE,
  position            INT NOT NULL,
  PRIMARY KEY (item_id, picture_id)
);

-- ============================================================
-- OPTIONS
-- ============================================================

CREATE TABLE option_type (
  id                  UUID PRIMARY KEY,
  code                TEXT NOT NULL UNIQUE,
  label               TEXT NOT NULL,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE item_option (
  id                  UUID PRIMARY KEY,
  item_id             UUID NOT NULL REFERENCES item(id)        ON DELETE CASCADE,
  type_id             UUID NOT NULL REFERENCES option_type(id) ON DELETE RESTRICT,
  value               TEXT NOT NULL,
  price               INT NOT NULL DEFAULT 0,
  position            INT NOT NULL DEFAULT 0,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (item_id, type_id, value)
);

-- ============================================================
-- CATEGORIES
-- ============================================================

CREATE TABLE category (
  id                  UUID PRIMARY KEY,
  name                TEXT NOT NULL,
  type                category_type NOT NULL,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE subcategory (
  id                  UUID PRIMARY KEY,
  name                TEXT NOT NULL,
  category_id         UUID NOT NULL REFERENCES category(id) ON DELETE CASCADE,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE item_subcategory (
  item_id             UUID NOT NULL REFERENCES item(id)        ON DELETE CASCADE,
  subcategory_id      UUID NOT NULL REFERENCES subcategory(id) ON DELETE CASCADE,
  PRIMARY KEY (item_id, subcategory_id)
);

-- ============================================================
-- USERS & FAVORITES
-- ============================================================

CREATE TABLE "user" (
  id                  UUID PRIMARY KEY,
  email               TEXT NOT NULL UNIQUE,
  password_hash       TEXT NOT NULL,
  full_name           TEXT NOT NULL,
  phone               TEXT,
  role                user_role NOT NULL DEFAULT 'user',
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE favorite (
  user_id             UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
  item_id             UUID NOT NULL REFERENCES item(id)   ON DELETE CASCADE,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, item_id)
);

-- ============================================================
-- ORDERS
-- ============================================================

CREATE TABLE "order" (
  id                    UUID PRIMARY KEY,
  user_id               UUID NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
  status                order_status NOT NULL DEFAULT 'created',
  total_price           INT NOT NULL,
  customer_comment      TEXT,
  admin_note            TEXT,
  contact_phone         TEXT NOT NULL,
  contact_full_name     TEXT NOT NULL,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE order_item (
  id                     UUID PRIMARY KEY,
  order_id               UUID NOT NULL REFERENCES "order"(id) ON DELETE CASCADE,
  item_id                UUID NOT NULL REFERENCES item(id)    ON DELETE RESTRICT,
  quantity               INT NOT NULL,
  unit_base_price        INT NOT NULL,
  unit_total_price       INT NOT NULL,
  item_name_snapshot     TEXT NOT NULL,
  item_articul_snapshot  TEXT NOT NULL
);

CREATE TABLE order_item_option (
  id                    UUID PRIMARY KEY,
  order_item_id         UUID NOT NULL REFERENCES order_item(id) ON DELETE CASCADE,
  type_code_snapshot    TEXT NOT NULL,
  type_label_snapshot   TEXT NOT NULL,
  value_snapshot        TEXT NOT NULL,
  price_snapshot        INT NOT NULL
);
```

---

## Диаграмма связей (обзор)

```
                    ┌──────────────┐
                    │   category   │
                    └──────┬───────┘
                           │ 1:N
                    ┌──────▼───────┐
                    │ subcategory  │
                    └──────┬───────┘
                           │ M:N  (item_subcategory)
┌──────────┐        ┌──────▼───────┐        ┌───────────────┐
│ picture  │◄──M:N──┤     item     ├──M:N──►│ option_type   │
└──────────┘ (item_ │              │ (item_ └───────────────┘
             picture)└──────┬───────┘  option)
                            │ 1:N
                            │          ┌───────────┐
                            └──M:N─────┤ "user"    │
                                       │ (favorite)│
                                       └─────┬─────┘
                                             │ 1:N
                                       ┌─────▼─────┐
                                       │  "order"  │
                                       └─────┬─────┘
                                             │ 1:N
                                       ┌─────▼──────┐
                                       │ order_item │
                                       └─────┬──────┘
                                             │ 1:N
                                       ┌─────▼─────────────┐
                                       │ order_item_option │ (чистый снапшот, без FK на опции)
                                       └───────────────────┘
```
