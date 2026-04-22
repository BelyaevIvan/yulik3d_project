# Спецификация API бэкенда yulik3d

Полный перечень HTTP-методов основного бэкенд-сервиса. Это **канонический источник истины** для реализации хэндлеров и документации. Любые изменения функциональности сначала отражаются здесь, затем в коде.

Связанные документы:
- [db_description.md](db_description.md) — структура БД (таблицы, поля, связи).
- [project_structure.md](project_structure.md) — структура репозитория и архитектурные правила.

---

## 1. Общие соглашения

### 1.1. Base URL и версионирование

Все методы имеют префикс `/api`. Версионирование URL (`/api/v1/...`) **не вводим** — проект небольшой, контракт ломаем через явный discontinue/release.

### 1.2. Роли доступа

| Сокращение | Значение |
|---|---|
| `guest` | Без сессии. Методы с этим доступом отвечают и авторизованным пользователям тоже |
| `user` | Авторизованный пользователь любой роли (`user` или `admin`). Middleware `RequireAuth` |
| `admin` | Только `role = 'admin'`. Middleware `RequireAuth` + `RequireRole(admin)` |

Middleware описаны в `backend/internal/middleware/auth.go`.

### 1.3. Аутентификация

Cookie `session` (httpOnly, Secure, SameSite=Lax, Max-Age=2592000). Браузер отправляет её автоматически. Backend → Redis → `user_id` + `role` + метаданные в ctx запроса. Postgres в middleware не трогается. Подробности — в `feedback_yulik3d_auth.md` (память).

### 1.4. Формат ошибок

**Единый формат ответа при любой ошибке** (валидация, 401, 403, 404, 409, 500 и т.д.):

```json
{
  "statusCode": 400,
  "url": "/api/items/42",
  "message": "Item not found",
  "date": "2026-04-22T18:42:00Z"
}
```

| Поле | Описание |
|---|---|
| `statusCode` | HTTP-код ответа. Дублирует статус, для удобства клиента |
| `url` | Полный путь запроса, где возникла ошибка (без query-параметров). Пример: `/api/orders/15` |
| `message` | Информативное сообщение. Для ожидаемых ошибок — понятный текст (`"Item not found"`, `"Invalid credentials"`). Для непредвиденных (500, паника) — универсальное `"Internal server error"` без раскрытия стека |
| `date` | Время запроса в ISO 8601 с таймзоной (RFC 3339). Пример: `2026-04-22T18:42:00Z` |

Реализация — в `middleware/error.go` через `errors.Is/As` над доменными ошибками из `model/errors.go`. Перед отправкой middleware стирает внутренние детали из `message` для кодов 5xx.

### 1.5. Пагинация

Все listing-методы (каталог, заказы, избранное, админские списки) поддерживают **единый стиль пагинации**:

**Query:**
- `limit` (int, optional) — размер страницы. Default: `20`. Max: `100`. Значения вне диапазона → clamp, не ошибка.
- `offset` (int, optional) — сдвиг от начала. Default: `0`.

**Ответ — единая обёртка:**
```json
{
  "items": [...],
  "total": 152,
  "limit": 20,
  "offset": 40
}
```

Где `items` — имя массива совпадает с семантикой ресурса (`items`, `orders`, `favorites` и т.д. — в каждом методе указано).

### 1.6. Формат даты

Все `*_at`-поля и поле `date` в ошибках — RFC 3339 / ISO 8601 с таймзоной: `2026-04-22T18:42:00Z`.

### 1.7. Цены

Все `price`, `sale`, `total_price` и т.п. — **целые в рублях** (см. `db_description.md`). `sale` — процент (0–100). `final_price` вычисляется как `round(price * (100 - sale) / 100)` и отдаётся клиенту уже посчитанным, чтобы фронт не дублировал логику.

### 1.8. Content-Type

Все запросы с телом — `application/json; charset=utf-8`, кроме загрузки картинок — `multipart/form-data`.
Все ответы — `application/json; charset=utf-8`.

### 1.9. Валидация входа

Хэндлер делает формальную проверку (JSON parse, типы). Бизнес-валидация (диапазоны, уникальность, ссылочная целостность) — в сервисе. Ошибки валидации → `400 Bad Request` с понятным `message`.

### 1.10. Health check

- `GET /api/health` — `guest`. Возвращает `200 { "status": "ok" }`. Проверяет доступность Postgres и Redis (пинг). При недоступности любой из зависимостей → `503 Service Unavailable` с единым error-форматом.

---

## 2. Аутентификация и сессии

### 2.1. `POST /api/auth/register`

**Описание:** Регистрация нового пользователя. Автоматически логинит (создаёт сессию и ставит cookie).

**Доступ:** `guest` (авторизованным отдаём 409 Conflict «Already authenticated»).

**Тело запроса:**
```json
{
  "email": "user@example.com",
  "password": "strongpass123",
  "full_name": "Иван Петров",
  "phone": "+79991234567"
}
```

**Валидация:**
- `email` — формат RFC 5322, нормализация (lowercase). Обязательно.
- `password` — min 8 символов, max 128. Обязательно.
- `full_name` — не пусто, max 200 символов. Обязательно.
- `phone` — optional, разумный формат (E.164 предпочтительно).

**Ответ (201 Created):**
```json
{
  "id": 42,
  "email": "user@example.com",
  "full_name": "Иван Петров",
  "phone": "+79991234567",
  "role": "user",
  "created_at": "2026-04-22T18:42:00Z"
}
```

**Заголовки ответа:**
```
Set-Cookie: session=<id>; HttpOnly; Secure; SameSite=Lax; Path=/; Max-Age=2592000
```

**Алгоритм:**
1. Валидация формата тела.
2. Нормализация email (lowercase, trim).
3. Проверить, что email ещё не занят: `SELECT id FROM "user" WHERE email=$1` → если найден, вернуть `409 Conflict` (`"Email already registered"`).
4. Захэшировать пароль через argon2id (`pkg/passwordhash`).
5. `INSERT INTO "user" (email, password_hash, full_name, phone, role) VALUES (...,'user') RETURNING id, created_at`.
6. Сгенерировать `session_id` (32 байта `crypto/rand`, base64url).
7. `SET session:<id>` в Redis с JSON `{user_id, role='user', full_name, created_at, expires_at, user_agent, ip}`, TTL 2592000.
8. Установить cookie.
9. Вернуть DTO пользователя без `password_hash`.

**Ошибки:** 400 (валидация), 409 (email занят / уже авторизован), 429 (rate limit), 500.

---

### 2.2. `POST /api/auth/login`

**Описание:** Вход по email/паролю. Создаёт сессию, ставит cookie.

**Доступ:** `guest` (авторизованным — 409 «Already authenticated»).

**Тело запроса:**
```json
{
  "email": "user@example.com",
  "password": "strongpass123"
}
```

**Ответ (200 OK):** тот же DTO, что и `register`.
**Заголовки ответа:** `Set-Cookie: session=...` (аналогично register).

**Алгоритм:**
1. Валидация формата.
2. Rate limit: `INCR rl:login:<ip>` и `INCR rl:login:<email>` в Redis с TTL 900. Если > 5 — `429 Too Many Requests`.
3. `SELECT id, password_hash, role, full_name FROM "user" WHERE email=$1`. Если не найден — `401 Unauthenticated` (`"Invalid credentials"` — не раскрывать, что именно не так).
4. Проверить пароль через argon2id.Verify. Если не совпал — `401` (та же формулировка).
5. На успехе: сбросить rate-limit счётчики (`DEL rl:login:<email>`).
6. Сгенерировать `session_id`, записать в Redis, поставить cookie (см. register шаг 6–8).
7. Вернуть DTO.

**Ошибки:** 400, 401, 409, 429, 500.

---

### 2.3. `POST /api/auth/logout`

**Описание:** Выход из текущей сессии.

**Доступ:** `user`.

**Тело запроса:** отсутствует.

**Ответ (204 No Content):** пустое тело.

**Заголовки ответа:**
```
Set-Cookie: session=; Max-Age=0; Path=/
```

**Алгоритм:**
1. Middleware уже положил в ctx `session_id`.
2. `DEL session:<id>` в Redis.
3. Установить cookie с `Max-Age=0` (очистка на стороне браузера).

**Ошибки:** 401, 500.

---

## 3. Профиль пользователя

### 3.1. `GET /api/me`

**Описание:** Возвращает полный профиль текущего пользователя (из Postgres, не из сессии — нужны все поля, включая phone).

**Доступ:** `user`.

**Ответ (200):**
```json
{
  "id": 42,
  "email": "user@example.com",
  "full_name": "Иван Петров",
  "phone": "+79991234567",
  "role": "user",
  "created_at": "2026-04-22T18:42:00Z"
}
```

**Алгоритм:**
1. `user_id` из ctx.
2. `SELECT id, email, full_name, phone, role, created_at FROM "user" WHERE id=$1`.
3. Вернуть DTO.

**Ошибки:** 401, 500.

---

### 3.2. `PATCH /api/me`

**Описание:** Обновление своих полей профиля (full_name, phone). Email и пароль не редактируются (в MVP).

**Доступ:** `user`.

**Тело запроса (все поля optional, обновляются те, что переданы):**
```json
{
  "full_name": "Иван Петров",
  "phone": "+79991234567"
}
```

**Ответ (200):** обновлённый DTO профиля (как в `GET /api/me`).

**Алгоритм:**
1. Валидация (full_name не пусто если передано; phone формат).
2. `UPDATE "user" SET full_name=COALESCE($2, full_name), phone=COALESCE($3, phone), updated_at=NOW() WHERE id=$1 RETURNING ...`.
3. Если `full_name` изменился — **опционально** обновить сессию в Redis (перезаписать `full_name` в значении), чтобы фронт всегда видел свежее. В MVP можно не делать — фронт дёргает `/api/me` после PATCH и живёт этим.
4. Вернуть DTO.

**Ошибки:** 400, 401, 500.

---

## 4. Каталог (публичный)

### 4.1. `GET /api/items`

**Описание:** Список товаров с фильтрацией, поиском, сортировкой, пагинацией. Публичный — возвращает только `hidden=false`.

**Доступ:** `guest`.

**Query-параметры:**

| Параметр | Тип | Default | Описание |
|---|---|---|---|
| `category_type` | `figure` \| `other` | — | Фильтр по типу категории |
| `category_id` | int | — | Фильтр по категории (через `subcategory.category_id`) |
| `subcategory_id` | int | — | Фильтр по подкатегории |
| `q` | string | — | Полнотекстовый поиск по `name` (ILIKE `%q%`, регистронезависимо) |
| `has_sale` | bool | — | `true` → только со скидкой (`sale > 0`) |
| `sort` | enum | `created_desc` | `created_desc`, `created_asc`, `price_asc`, `price_desc`, `name_asc`, `name_desc` |
| `limit` | int | 20 | Макс 100 |
| `offset` | int | 0 | |

**Ответ (200):**
```json
{
  "items": [
    {
      "id": 1,
      "name": "Ам Ням",
      "articul": "CAT-001",
      "price": 2500,
      "sale": 10,
      "final_price": 2250,
      "primary_picture_url": "https://.../items/1/main.jpg",
      "category": { "id": 1, "name": "Игры", "type": "figure" },
      "subcategories": [ { "id": 5, "name": "Cut The Rope" } ]
    }
  ],
  "total": 42,
  "limit": 20,
  "offset": 0
}
```

**Алгоритм:**
1. Парсинг query, clamp `limit`/`offset`.
2. Собрать `WHERE`: `hidden = false` + условия фильтров.
3. `SELECT COUNT(*)` для `total`.
4. `SELECT ... FROM item JOIN item_subcategory JOIN subcategory JOIN category ...` с `DISTINCT item.id`.
5. Для каждого товара получить primary picture (`ORDER BY position LIMIT 1` из `item_picture JOIN picture`) и собрать URL через helper.
6. Для каждого товара — список подкатегорий (агрегируем в один запрос с `JSON_AGG` или постпроцессингом).
7. Посчитать `final_price` для каждого.
8. Вернуть обёртку с пагинацией.

**Примечание про категорию товара:** товар может относиться к нескольким подкатегориям из разных категорий. В списке каталога показываем «первую» категорию (упорядоченно по `category.id`). Полный список — в детальном методе.

**Ошибки:** 400 (невалидный sort/тип), 500.

---

### 4.2. `GET /api/items/:id`

**Описание:** Детальная карточка товара — всё: описания, цена, все картинки, опции сгруппированные по типу, подкатегории/категории. Работает **и для скрытых товаров** — флаг `hidden` в ответе, фронт рендерит «Не доступно к заказу».

**Доступ:** `guest`.

**Путь-параметр:** `id` — int.

**Ответ (200):**
```json
{
  "id": 1,
  "name": "Ам Ням",
  "articul": "CAT-001",
  "description_info": "**Технология:** Ручная работа + 3D принтер\n**Материал:** Полимерная смола\n...",
  "description_other": "- Подсвечивается в темноте\n- ...",
  "price": 2500,
  "sale": 10,
  "final_price": 2250,
  "hidden": false,
  "pictures": [
    { "id": 10, "url": "https://.../items/1/main.jpg", "position": 1 },
    { "id": 11, "url": "https://.../items/1/g2.jpg", "position": 2 }
  ],
  "options": [
    {
      "type": { "id": 1, "code": "size", "label": "Размер" },
      "values": [
        { "id": 1, "value": "S", "price": 0,    "position": 0 },
        { "id": 2, "value": "M", "price": 1100, "position": 1 },
        { "id": 3, "value": "L", "price": 4300, "position": 2 }
      ]
    },
    {
      "type": { "id": 2, "code": "engraving", "label": "Гравировка" },
      "values": [
        { "id": 4, "value": "Нет", "price": 0,    "position": 0 },
        { "id": 5, "value": "Да",  "price": 1000, "position": 1 }
      ]
    }
  ],
  "subcategories": [
    { "id": 5, "name": "Cut The Rope", "category": { "id": 1, "name": "Игры", "type": "figure" } }
  ],
  "created_at": "...",
  "updated_at": "..."
}
```

**Алгоритм:**
1. `SELECT * FROM item WHERE id=$1`. Если нет — `404 Not Found` (`"Item not found"`).
2. `SELECT p.* FROM picture p JOIN item_picture ip ON ... WHERE ip.item_id=$1 ORDER BY ip.position`.
3. `SELECT io.*, ot.* FROM item_option io JOIN option_type ot ON ... WHERE io.item_id=$1 ORDER BY ot.id, io.position`. В сервисе сгруппировать по type.
4. `SELECT s.*, c.* FROM item_subcategory is JOIN subcategory s JOIN category c WHERE is.item_id=$1`.
5. Собрать URL картинок через helper.
6. Посчитать `final_price`.
7. Отдать DTO.

**Ошибки:** 404, 500.

---

### 4.3. `GET /api/categories`

**Описание:** Список всех категорий с опциональным разворачиванием подкатегорий. Для фронта — построить меню/фильтры.

**Доступ:** `guest`.

**Query-параметры:**

| Параметр | Тип | Описание |
|---|---|---|
| `type` | `figure` \| `other` | Опциональный фильтр по типу |
| `with_subcategories` | bool, default `false` | Если `true` — вложить массив подкатегорий в каждую категорию |

**Ответ (200):**
```json
{
  "categories": [
    {
      "id": 1,
      "name": "Игры",
      "type": "figure",
      "subcategories": [
        { "id": 5, "name": "Cut The Rope" },
        { "id": 6, "name": "Dota 2" }
      ]
    }
  ]
}
```

Если `with_subcategories=false` — ключ `subcategories` отсутствует.

Без пагинации — категорий заведомо мало.

**Алгоритм:**
1. Фильтр по `type`.
2. `SELECT * FROM category ORDER BY name`.
3. Если `with_subcategories` — `SELECT * FROM subcategory WHERE category_id = ANY($1) ORDER BY name`, сгруппировать.
4. Отдать.

**Ошибки:** 400 (невалидный type), 500.

---

### 4.4. `GET /api/categories/:id/subcategories`

**Описание:** Подкатегории указанной категории (короткий эндпоинт, если `with_subcategories` не нужно на всех сразу).

**Доступ:** `guest`.

**Путь:** `id` — category id.

**Ответ (200):**
```json
{
  "subcategories": [
    { "id": 5, "name": "Cut The Rope" }
  ]
}
```

**Алгоритм:**
1. Проверить существование категории → `404` если нет.
2. `SELECT * FROM subcategory WHERE category_id=$1 ORDER BY name`.
3. Отдать.

**Ошибки:** 404, 500.

---

## 5. Избранное

### 5.1. `GET /api/favorites`

**Описание:** Мои избранные товары. Возвращает те же карточки, что и каталог.

**Доступ:** `user`.

**Query:** `limit`, `offset` (как в 1.5).

**Ответ (200):**
```json
{
  "items": [ /* та же форма, что в GET /api/items */ ],
  "total": 12,
  "limit": 20,
  "offset": 0
}
```

**Алгоритм:**
1. `user_id` из ctx.
2. `SELECT COUNT(*) FROM favorite WHERE user_id=$1` → total.
3. `SELECT i.* FROM favorite f JOIN item i ON ... WHERE f.user_id=$1 ORDER BY f.created_at DESC LIMIT ... OFFSET ...`.
4. Для каждого — primary picture, subcategories, category (как в 4.1).
5. **Скрытые товары в избранном оставляем** — юзер увидит `hidden: true` и сможет понять, что товар пока недоступен. Не фильтруем.
6. Отдать.

**Ошибки:** 401, 500.

---

### 5.2. `POST /api/favorites/:item_id`

**Описание:** Добавить товар в избранное. Идемпотентно: повторный вызов — не ошибка.

**Доступ:** `user`.

**Путь:** `item_id` — int.

**Тело:** нет.

**Ответ (200):**
```json
{ "item_id": 42, "created_at": "2026-04-22T18:42:00Z" }
```

**Алгоритм:**
1. Проверить существование товара → `404` если нет. (Даже скрытые можно добавлять — юзер сохранил ссылку.)
2. `INSERT INTO favorite (user_id, item_id) VALUES ($1, $2) ON CONFLICT DO NOTHING RETURNING created_at`.
3. Если `ON CONFLICT` сработал — сделать `SELECT` для `created_at` (или просто вернуть `NOW()`).
4. Отдать.

**Ошибки:** 401, 404, 500.

---

### 5.3. `DELETE /api/favorites/:item_id`

**Описание:** Убрать товар из избранного. Идемпотентно.

**Доступ:** `user`.

**Ответ (204 No Content):** пустое тело.

**Алгоритм:**
1. `DELETE FROM favorite WHERE user_id=$1 AND item_id=$2`.
2. Вернуть 204 независимо от того, была запись или нет (идемпотентность).

**Ошибки:** 401, 500.

---

## 6. Заказы (пользователь)

### 6.1. `POST /api/orders`

**Описание:** Создание заказа из содержимого корзины. **Вся ценовая информация проверяется на бэке** — фронт только указывает, какой товар и какие опции.

**Доступ:** `user`.

**Тело запроса:**
```json
{
  "items": [
    {
      "item_id": 1,
      "quantity": 1,
      "option_ids": [2, 5]
    },
    {
      "item_id": 7,
      "quantity": 2,
      "option_ids": []
    }
  ],
  "customer_comment": "Можно в подарочной упаковке",
  "contact_phone": "+79991234567",
  "contact_full_name": "Иван Петров"
}
```

**Валидация:**
- `items` — не пусто, max 100 позиций.
- Каждая `quantity` — целое, 1..99.
- `option_ids` — массив уникальных int (могут быть нули).
- `contact_phone`, `contact_full_name` — обязательны.

**Ответ (201 Created):** полный DTO заказа (как в 6.3).

**Алгоритм:**
```
НАЧАТЬ TX (TxManager)
  для каждой позиции items[i]:
    1. SELECT id, name, price, sale, articul, hidden FROM item WHERE id=$1
       если нет → 400 (Item {id} not found)
       если hidden=true → 409 Conflict (Item {id} is not available)
    2. base_price = round(price * (100-sale)/100)
    3. Для каждого option_id в option_ids:
         SELECT io.*, ot.code, ot.label
         FROM item_option io JOIN option_type ot ON io.type_id=ot.id
         WHERE io.id=$1 AND io.item_id=<текущий item_id>
         если нет → 400 (Option {id} does not belong to item {item_id})
    4. unit_total_price = base_price + sum(option.price)
  
  5. total_price = sum(unit_total_price * quantity) по всем позициям
  
  6. INSERT INTO "order" (user_id, status='created', total_price,
                          customer_comment, contact_phone, contact_full_name)
     RETURNING id
  
  7. Для каждой позиции:
       INSERT INTO order_item (order_id, item_id, quantity,
                               unit_base_price, unit_total_price,
                               item_name_snapshot, item_articul_snapshot)
       RETURNING id
     
     для каждой выбранной опции:
       INSERT INTO order_item_option (order_item_id,
                                      type_code_snapshot, type_label_snapshot,
                                      value_snapshot, price_snapshot)
COMMIT

8. (Опционально) отправить email админу — сейчас не реализуем, см. memory/project_yulik3d.md
9. Вернуть DTO заказа с полным раскрытием.
```

Вся валидация — в сервисе. Репозиторий атомарен через `TxManager.Run(ctx, fn)`.

**Ошибки:** 400 (валидация/неизвестный item/option), 401, 409 (item hidden), 500.

---

### 6.2. `GET /api/orders`

**Описание:** История моих заказов (краткий вид для списка).

**Доступ:** `user`.

**Query:** `limit`, `offset`, опционально `status` (фильтр).

**Ответ (200):**
```json
{
  "orders": [
    {
      "id": 15,
      "status": "created",
      "total_price": 5500,
      "items_count": 2,
      "created_at": "...",
      "updated_at": "..."
    }
  ],
  "total": 4,
  "limit": 20,
  "offset": 0
}
```

**Алгоритм:**
1. `user_id` из ctx.
2. `SELECT COUNT(*)` + `SELECT ... FROM "order" WHERE user_id=$1 [AND status=$2] ORDER BY created_at DESC LIMIT ... OFFSET ...`.
3. Для каждого заказа — `(SELECT COUNT(*) FROM order_item WHERE order_id=o.id)` как `items_count` (подзапрос или отдельным агрегатом).
4. Отдать.

**Ошибки:** 401, 500.

---

### 6.3. `GET /api/orders/:id`

**Описание:** Детали моего заказа. Ownership — проверка в сервисе.

**Доступ:** `user`.

**Путь:** `id` — int.

**Ответ (200):**
```json
{
  "id": 15,
  "status": "manufacturing",
  "total_price": 5500,
  "customer_comment": "...",
  "contact_phone": "...",
  "contact_full_name": "...",
  "items": [
    {
      "id": 101,
      "item_id": 1,
      "item_name_snapshot": "Ам Ням",
      "item_articul_snapshot": "CAT-001",
      "quantity": 1,
      "unit_base_price": 2250,
      "unit_total_price": 3350,
      "options": [
        { "type_code_snapshot": "size",      "type_label_snapshot": "Размер",    "value_snapshot": "M", "price_snapshot": 1100 },
        { "type_code_snapshot": "engraving", "type_label_snapshot": "Гравировка","value_snapshot": "Да","price_snapshot": 1000 }
      ]
    }
  ],
  "created_at": "...",
  "updated_at": "..."
}
```

Без `admin_note` — это внутреннее поле админа.

**Алгоритм:**
1. `SELECT * FROM "order" WHERE id=$1` → если нет → `404`.
2. **Если `order.user_id != ctx.user_id`** → `404` (не 403, чтобы не светить факт существования чужого заказа).
3. `SELECT * FROM order_item WHERE order_id=$1 ORDER BY id`.
4. Для каждой позиции — `SELECT * FROM order_item_option WHERE order_item_id=$1`.
5. Собрать DTO.

**Ошибки:** 401, 404, 500.

---

## 7. Админ — Товары

### 7.1. `GET /api/admin/items`

**Описание:** Админский список товаров. Отличие от публичного — возвращает **все** товары (в т.ч. `hidden=true`) и дополнительные поля для управления.

**Доступ:** `admin`.

**Query:** все те же, что в 4.1, плюс `hidden` (`any` | `true` | `false`, default `any`).

**Ответ (200):** как 4.1, но в каждом элементе дополнительно `hidden: true/false`, `created_at`, `updated_at`.

**Алгоритм:** аналогично 4.1, без ограничения `hidden=false`.

**Ошибки:** 401, 403, 500.

---

### 7.2. `POST /api/admin/items`

**Описание:** Создать товар. Атомарно: item + subcategory-связи + опции. **Картинки загружаются отдельно** (см. §8) — нужен файл, отдельный multipart.

**Доступ:** `admin`.

**Тело запроса:**
```json
{
  "name": "Ам Ням",
  "description_info": "**Технология:** ...\n**Материал:** ...",
  "description_other": "- Особенность 1\n- Особенность 2",
  "price": 2500,
  "sale": 10,
  "hidden": false,
  "subcategory_ids": [5, 7],
  "options": [
    { "type_id": 1, "value": "S", "price": 0,    "position": 0 },
    { "type_id": 1, "value": "M", "price": 1100, "position": 1 },
    { "type_id": 2, "value": "Нет", "price": 0, "position": 0 }
  ]
}
```

**Валидация:**
- `name`, `description_info`, `description_other` — не пусто.
- `price` ≥ 0, `sale` 0..100.
- `subcategory_ids` — все существуют.
- `options[].type_id` — существуют в `option_type`.
- `(type_id, value)` внутри массива `options` уникальны для каждого type_id (проверка на дубликаты).

**Ответ (201 Created):** полный DTO товара (как в 4.2, без картинок — их ещё нет).

**Заголовки ответа:** `Location: /api/items/:id`.

**Алгоритм:**
```
НАЧАТЬ TX
  1. Сгенерировать articul (например, "CAT-" + zero-padded серийник из SEQUENCE или из MAX(id)+1)
  2. INSERT INTO item (name, description_info, description_other,
                       price, sale, articul, hidden) RETURNING id
  3. Для каждого sid из subcategory_ids:
       INSERT INTO item_subcategory (item_id, subcategory_id) VALUES ($1, $sid)
  4. Для каждой опции:
       INSERT INTO item_option (item_id, type_id, value, price, position)
COMMIT

5. Загрузить свежесозданный товар (reuse GET /api/items/:id логики) и вернуть.
```

**Ошибки:** 400 (валидация), 401, 403, 404 (subcategory/type не найдены — как 400 с пояснением), 409 (violation уникальности), 500.

---

### 7.3. `GET /api/admin/items/:id`

**Описание:** Админская карточка товара. Отличие от 4.2 — не фильтрует по `hidden`, добавляет `created_at/updated_at`.

**Доступ:** `admin`.

**Ответ (200):** как 4.2 + все админские поля.

**Ошибки:** 401, 403, 404, 500.

---

### 7.4. `PUT /api/admin/items/:id`

**Описание:** Полная замена товара. Опции и subcategory-связи **перезаписываются целиком** тем, что в теле. Картинки не трогаются — для них отдельные эндпоинты.

**Доступ:** `admin`.

**Тело:** как в 7.2, без `articul` (неизменяемое). Все поля обязательны.

**Ответ (200):** обновлённый DTO товара (как в 7.3).

**Алгоритм:**
```
НАЧАТЬ TX
  1. UPDATE item SET name=$, description_info=$, description_other=$,
                     price=$, sale=$, hidden=$, updated_at=NOW()
     WHERE id=$id RETURNING id → если нет → 404
  
  2. DELETE FROM item_subcategory WHERE item_id=$id
     Для каждого sid из subcategory_ids: INSERT ...
  
  3. DELETE FROM item_option WHERE item_id=$id
     Для каждой опции: INSERT ...
COMMIT
```

Удаление опций физическое. История в заказах сохранится через snapshot (не ссылается на `item_option`).

**Ошибки:** 400, 401, 403, 404, 409, 500.

---

### 7.5. `PATCH /api/admin/items/:id`

**Описание:** Частичное обновление. В MVP основной use-case — быстрое переключение `hidden`. Другие поля можно редактировать через `PUT`.

**Доступ:** `admin`.

**Тело (любое подмножество):**
```json
{
  "hidden": true
}
```

Разрешённые поля: `hidden`, `name`, `description_info`, `description_other`, `price`, `sale`.

**Ответ (200):** обновлённый DTO (как 7.3).

**Алгоритм:**
1. Собрать `UPDATE item SET ... WHERE id=$id` только из переданных полей.
2. Если ничего не передано — `400 Bad Request` (`"No fields to update"`).
3. Вернуть свежий DTO.

**Ошибки:** 400, 401, 403, 404, 500.

---

## 8. Админ — Картинки товара

### 8.1. `POST /api/admin/items/:id/pictures`

**Описание:** Загрузка картинки к товару. MinIO + запись в БД. Позиция в галерее — опциональный параметр; если не указана, назначается `MAX(position) + 1`.

**Доступ:** `admin`.

**Content-Type:** `multipart/form-data`.

**Form fields:**
- `file` (file, обязательно) — изображение (png/jpg/webp).
- `position` (int, optional) — позиция в галерее (1 = титульная).

**Валидация файла:**
- MIME: `image/png`, `image/jpeg`, `image/webp`.
- Размер ≤ 10 MB (настраивается через env).
- Расширение согласовано с MIME.

**Ответ (201):**
```json
{
  "id": 101,
  "url": "https://.../items/42/<uuid>.jpg",
  "position": 3
}
```

**Алгоритм:**
```
1. Проверить существование item → 404 если нет.
2. Валидация файла (MIME, размер).
3. Сгенерировать object_key = "items/<item_id>/<uuid>.<ext>"
4. Загрузить в MinIO (bucket=yulik3d, putObject). При ошибке → 500.
5. НАЧАТЬ TX:
     - INSERT INTO picture (object_key) RETURNING id
     - Если position не передан: pos = (SELECT COALESCE(MAX(position),0)+1 FROM item_picture WHERE item_id=$id)
     - INSERT INTO item_picture (item_id, picture_id, position) VALUES ...
   COMMIT
6. Собрать полный URL (через helper) и вернуть DTO.
```

Если TX не удалась — откатить + удалить файл из MinIO (компенсирующая операция).

**Ошибки:** 400 (валидация файла), 401, 403, 404, 413 (Payload Too Large), 415 (Unsupported Media Type), 500.

---

### 8.2. `DELETE /api/admin/items/:item_id/pictures/:picture_id`

**Описание:** Удалить картинку из товара (и из MinIO, если она больше нигде не используется).

**Доступ:** `admin`.

**Ответ (204 No Content).**

**Алгоритм:**
```
1. НАЧАТЬ TX:
     - DELETE FROM item_picture WHERE item_id=$1 AND picture_id=$2
     - Если аффект 0 → 404 (связь не найдена)
     - SELECT COUNT(*) FROM item_picture WHERE picture_id=$2
       если 0 → (a) DELETE FROM picture WHERE id=$2
                (b) флаг "нужно удалить из MinIO"
   COMMIT
2. Если флаг взведён → удалить object из MinIO (best-effort, ошибку логируем, но 204 отдаём).
```

**Ошибки:** 401, 403, 404, 500.

---

### 8.3. `PATCH /api/admin/items/:id/pictures/reorder`

**Описание:** Переупорядочить картинки товара.

**Доступ:** `admin`.

**Тело:**
```json
{
  "order": [
    { "picture_id": 11, "position": 1 },
    { "picture_id": 10, "position": 2 },
    { "picture_id": 12, "position": 3 }
  ]
}
```

Должен содержать **все** картинки товара. Позиции — положительные уникальные.

**Ответ (200):**
```json
{
  "pictures": [
    { "id": 11, "url": "...", "position": 1 },
    ...
  ]
}
```

**Алгоритм:**
1. Получить список существующих картинок товара.
2. Сверить: множество в теле === множество в БД. Иначе → `400 Bad Request`.
3. В TX: `UPDATE item_picture SET position=$1 WHERE item_id=$2 AND picture_id=$3` для каждой.
4. Вернуть свежий список.

**Ошибки:** 400, 401, 403, 404, 500.

---

## 9. Админ — Опции товаров и типы опций

### 9.1. `GET /api/admin/option-types`

**Описание:** Список типов опций. Для выпадашки в UI админа (чтобы выбирать при создании опции товара).

**Доступ:** `admin`.

**Ответ (200):**
```json
{
  "option_types": [
    { "id": 1, "code": "size",      "label": "Размер",    "created_at": "..." },
    { "id": 2, "code": "engraving", "label": "Гравировка","created_at": "..." }
  ]
}
```

Без пагинации — типов заведомо мало.

**Алгоритм:** `SELECT * FROM option_type ORDER BY label`.

**Ошибки:** 401, 403, 500.

---

### 9.2. `POST /api/admin/option-types`

**Описание:** Создать новый тип опции.

**Доступ:** `admin`.

**Тело:**
```json
{
  "code": "gift_wrap",
  "label": "Подарочная упаковка"
}
```

**Валидация:**
- `code` — slug: lowercase, `[a-z0-9_]`, 2..50 символов. Уникальный.
- `label` — не пусто, max 100 символов.

**Ответ (201):** созданный DTO.

**Алгоритм:** `INSERT INTO option_type (code, label) VALUES (...) RETURNING ...`. На `unique_violation` → `409 Conflict` (`"Option type code already exists"`).

**Ошибки:** 400, 401, 403, 409, 500.

---

### 9.3. `PATCH /api/admin/option-types/:id`

**Описание:** Редактирование типа. `code` **не меняется** (ломает ссылки в логике). Меняется только `label`.

**Доступ:** `admin`.

**Тело:**
```json
{
  "label": "Подарочная упаковка (новая)"
}
```

**Ответ (200):** обновлённый DTO.

**Алгоритм:** `UPDATE option_type SET label=$1, updated_at=NOW() WHERE id=$2 RETURNING ...`.

**Ошибки:** 400, 401, 403, 404, 500.

---

### 9.4. `DELETE /api/admin/option-types/:id`

**Описание:** Удалить тип. FK на `item_option` — `ON DELETE RESTRICT`, поэтому если тип используется → БД вернёт ошибку → отдаём `409 Conflict`.

**Доступ:** `admin`.

**Ответ (204).**

**Алгоритм:**
1. `DELETE FROM option_type WHERE id=$1`.
2. Ловим `foreign_key_violation` → `409` (`"Option type is in use; remove its options from items first"`).
3. Если affected=0 → `404`.

**Ошибки:** 401, 403, 404, 409, 500.

---

### 9.5. `POST /api/admin/items/:id/options`

**Описание:** Добавить опцию к существующему товару. Альтернатива перезаписи товара через `PUT`.

**Доступ:** `admin`.

**Тело:**
```json
{
  "type_id": 1,
  "value": "XL",
  "price": 5500,
  "position": 3
}
```

**Ответ (201):**
```json
{
  "id": 20,
  "item_id": 42,
  "type": { "id": 1, "code": "size", "label": "Размер" },
  "value": "XL",
  "price": 5500,
  "position": 3
}
```

**Алгоритм:**
1. Проверить item → `404` если нет.
2. `INSERT INTO item_option (item_id, type_id, value, price, position) VALUES (...)`.
3. На `unique_violation (item_id, type_id, value)` → `409`.
4. Собрать DTO с joined `option_type`.

**Ошибки:** 400, 401, 403, 404, 409, 500.

---

### 9.6. `PATCH /api/admin/item-options/:id`

**Описание:** Редактирование опции товара (`value`, `price`, `position`). `type_id` и `item_id` не меняются.

**Доступ:** `admin`.

**Тело:**
```json
{
  "value": "XL",
  "price": 5800,
  "position": 4
}
```

**Ответ (200):** обновлённый DTO.

**Алгоритм:** `UPDATE item_option SET ... WHERE id=$1 RETURNING ...`.

**Ошибки:** 400, 401, 403, 404, 409 (уникальность), 500.

---

### 9.7. `DELETE /api/admin/item-options/:id`

**Описание:** Удалить опцию товара. История заказов не ломается (snapshot).

**Доступ:** `admin`.

**Ответ (204).**

**Алгоритм:** `DELETE FROM item_option WHERE id=$1`. Affected=0 → `404`.

**Ошибки:** 401, 403, 404, 500.

---

## 10. Админ — Категории и подкатегории

### 10.1. `POST /api/admin/categories`

**Описание:** Создать категорию.

**Доступ:** `admin`.

**Тело:**
```json
{ "name": "Декор", "type": "other" }
```

**Ответ (201):** созданная категория.

**Алгоритм:** INSERT + RETURNING.

**Ошибки:** 400, 401, 403, 500.

---

### 10.2. `PATCH /api/admin/categories/:id`

**Описание:** Обновить поля категории (`name`, `type`).

**Доступ:** `admin`.

**Тело:** любое подмножество разрешённых полей.

**Ответ (200):** обновлённый DTO.

**Ошибки:** 400, 401, 403, 404, 500.

---

### 10.3. `DELETE /api/admin/categories/:id`

**Описание:** Удалить категорию. `ON DELETE CASCADE` на `subcategory` — подкатегории уходят вместе. Связи `item_subcategory` — тоже через CASCADE.

**Доступ:** `admin`.

**Ответ (204).**

**Алгоритм:** `DELETE FROM category WHERE id=$1`. Affected=0 → `404`.

**Ошибки:** 401, 403, 404, 500.

---

### 10.4. `POST /api/admin/categories/:id/subcategories`

**Описание:** Создать подкатегорию в категории.

**Доступ:** `admin`.

**Тело:**
```json
{ "name": "Вазы" }
```

**Ответ (201):**
```json
{ "id": 20, "name": "Вазы", "category_id": 5, "created_at": "..." }
```

**Алгоритм:**
1. Проверить существование category.
2. `INSERT INTO subcategory (name, category_id) VALUES (...)`.

**Ошибки:** 400, 401, 403, 404, 500.

---

### 10.5. `PATCH /api/admin/subcategories/:id`

**Описание:** Обновить поля подкатегории (`name`, `category_id` — можно перенести в другую категорию).

**Доступ:** `admin`.

**Тело:** подмножество разрешённых полей.

**Ответ (200):** обновлённый DTO.

**Ошибки:** 400, 401, 403, 404, 500.

---

### 10.6. `DELETE /api/admin/subcategories/:id`

**Описание:** Удалить подкатегорию. Связи с товарами (`item_subcategory`) уходят CASCADE.

**Доступ:** `admin`.

**Ответ (204).**

**Ошибки:** 401, 403, 404, 500.

---

## 11. Админ — Заказы

### 11.1. `GET /api/admin/orders`

**Описание:** Список всех заказов для админки (очередь работ). Сортировка по дате создания (новые сверху).

**Доступ:** `admin`.

**Query:**

| Параметр | Тип | Default | Описание |
|---|---|---|---|
| `status` | enum | — | Фильтр по статусу (`created`, `confirmed`, `manufacturing`, `delivering`, `completed`, `cancelled`) |
| `user_id` | int | — | Фильтр по пользователю |
| `q` | string | — | Поиск по `contact_full_name` или `contact_phone` (ILIKE) |
| `limit` | int | 20 | |
| `offset` | int | 0 | |

**Ответ (200):**
```json
{
  "orders": [
    {
      "id": 15,
      "user": { "id": 42, "email": "user@example.com", "full_name": "Иван Петров" },
      "status": "created",
      "total_price": 5500,
      "items_count": 2,
      "contact_phone": "+79991234567",
      "contact_full_name": "Иван Петров",
      "customer_comment": "...",
      "admin_note": "Позвонить после 18:00",
      "created_at": "...",
      "updated_at": "..."
    }
  ],
  "total": 47,
  "limit": 20,
  "offset": 0
}
```

**Алгоритм:**
1. Парсинг query.
2. `SELECT COUNT(*) FROM "order" WHERE ...`.
3. `SELECT o.*, u.email, u.full_name AS user_full_name, (SELECT COUNT(*) FROM order_item WHERE order_id=o.id) AS items_count FROM "order" o JOIN "user" u ON o.user_id=u.id WHERE ... ORDER BY o.created_at DESC LIMIT ... OFFSET ...`.
4. Отдать.

**Ошибки:** 401, 403, 500.

---

### 11.2. `GET /api/admin/orders/:id`

**Описание:** Детали заказа для админа — полный вид с `admin_note` и инфой о пользователе.

**Доступ:** `admin`.

**Ответ (200):**
```json
{
  "id": 15,
  "user": { "id": 42, "email": "...", "full_name": "...", "phone": "..." },
  "status": "manufacturing",
  "total_price": 5500,
  "customer_comment": "...",
  "admin_note": "...",
  "contact_phone": "...",
  "contact_full_name": "...",
  "items": [ /* как в 6.3 */ ],
  "created_at": "...",
  "updated_at": "..."
}
```

**Алгоритм:** как 6.3, без проверки ownership + с `admin_note` и данными пользователя.

**Ошибки:** 401, 403, 404, 500.

---

### 11.3. `PATCH /api/admin/orders/:id/status`

**Описание:** Смена статуса заказа. Разрешены только переходы:
- Вперёд по цепочке: `created → confirmed → manufacturing → delivering → completed`.
- В `cancelled` — из любого статуса, кроме `completed` и самого `cancelled`.

Никаких возвратов назад.

**Доступ:** `admin`.

**Тело:**
```json
{ "status": "confirmed" }
```

**Ответ (200):** обновлённый DTO заказа (как 11.2).

**Алгоритм:**
1. `SELECT status FROM "order" WHERE id=$1` → `404` если нет.
2. Валидировать переход по матрице:
   ```
   created       → confirmed, cancelled
   confirmed     → manufacturing, cancelled
   manufacturing → delivering, cancelled
   delivering    → completed, cancelled
   completed     → (ничего)
   cancelled     → (ничего)
   ```
   Недопустимый переход → `409 Conflict` (`"Invalid status transition: X -> Y"`).
3. `UPDATE "order" SET status=$1, updated_at=NOW() WHERE id=$2`.
4. Отдать полный DTO.

**Ошибки:** 400 (невалидный status), 401, 403, 404, 409, 500.

---

### 11.4. `PATCH /api/admin/orders/:id`

**Описание:** Обновить внутренние поля заказа (сейчас — только `admin_note`).

**Доступ:** `admin`.

**Тело:**
```json
{ "admin_note": "Позвонить после 18:00" }
```

**Ответ (200):** обновлённый DTO.

**Алгоритм:** `UPDATE "order" SET admin_note=$1, updated_at=NOW() WHERE id=$2`.

**Ошибки:** 400, 401, 403, 404, 500.

---

## 12. Сводная матрица эндпоинтов

| Метод | Путь | Доступ |
|---|---|---|
| GET | `/api/health` | guest |
| POST | `/api/auth/register` | guest |
| POST | `/api/auth/login` | guest |
| POST | `/api/auth/logout` | user |
| GET | `/api/me` | user |
| PATCH | `/api/me` | user |
| GET | `/api/items` | guest |
| GET | `/api/items/:id` | guest |
| GET | `/api/categories` | guest |
| GET | `/api/categories/:id/subcategories` | guest |
| GET | `/api/favorites` | user |
| POST | `/api/favorites/:item_id` | user |
| DELETE | `/api/favorites/:item_id` | user |
| POST | `/api/orders` | user |
| GET | `/api/orders` | user |
| GET | `/api/orders/:id` | user |
| GET | `/api/admin/items` | admin |
| POST | `/api/admin/items` | admin |
| GET | `/api/admin/items/:id` | admin |
| PUT | `/api/admin/items/:id` | admin |
| PATCH | `/api/admin/items/:id` | admin |
| POST | `/api/admin/items/:id/pictures` | admin |
| DELETE | `/api/admin/items/:item_id/pictures/:picture_id` | admin |
| PATCH | `/api/admin/items/:id/pictures/reorder` | admin |
| GET | `/api/admin/option-types` | admin |
| POST | `/api/admin/option-types` | admin |
| PATCH | `/api/admin/option-types/:id` | admin |
| DELETE | `/api/admin/option-types/:id` | admin |
| POST | `/api/admin/items/:id/options` | admin |
| PATCH | `/api/admin/item-options/:id` | admin |
| DELETE | `/api/admin/item-options/:id` | admin |
| POST | `/api/admin/categories` | admin |
| PATCH | `/api/admin/categories/:id` | admin |
| DELETE | `/api/admin/categories/:id` | admin |
| POST | `/api/admin/categories/:id/subcategories` | admin |
| PATCH | `/api/admin/subcategories/:id` | admin |
| DELETE | `/api/admin/subcategories/:id` | admin |
| GET | `/api/admin/orders` | admin |
| GET | `/api/admin/orders/:id` | admin |
| PATCH | `/api/admin/orders/:id/status` | admin |
| PATCH | `/api/admin/orders/:id` | admin |

**Итого: 41 эндпоинт.**

---

## 13. Что закрыть при реализации

- Единый error-handler в `middleware/error.go` с обязательной трансляцией доменных ошибок в HTTP + формирование тела `{statusCode, url, message, date}`. Сохранять исходную ошибку в лог (structured slog), в клиент не раскрывать.
- Helper для URL картинок в `repository/photo_url.go` или `pkg/` — превращает `object_key` в абсолютный URL.
- Утилита генерации `articul` (последовательность через PostgreSQL `SEQUENCE`, префикс `CAT-`).
- Rate-limiter middleware на Redis для `/api/auth/login` и `/api/auth/register`.
- Email-уведомление админу при `POST /api/orders` — **отложено**, см. `memory/project_yulik3d.md`.
