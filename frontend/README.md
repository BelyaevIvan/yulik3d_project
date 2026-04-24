# yulik3d frontend

SPA на TypeScript + Vite + Handlebars + SCSS, History API роутер, без фреймворков.

## Перед первым билдом

Сгенерируй `package-lock.json` (Dockerfile использует `npm ci` для скорости/детерминизма):

```bash
cd frontend
npm install   # создаст package-lock.json
```

После этого `package-lock.json` нужно коммитить в репозиторий.

## Запуск через docker-compose (вместе с бэком)

Из корня проекта:

```bash
docker compose up -d --build
```

- Фронт: http://localhost:5173/
- Бэк: http://localhost:8080/api/v1
- Swagger: http://localhost:8080/swagger/
- MinIO: http://localhost:9001

Nginx внутри контейнера фронта проксирует `/api/*` на backend и `/minio/*` на minio.

## Локальная разработка (Vite dev server)

```bash
cd frontend
npm install
VITE_BACKEND_URL=http://localhost:8080 npm run dev
```

Открыть http://localhost:5173/.
В dev `vite.config.js` проксирует `/api/*` на `VITE_BACKEND_URL`.

## Логотип

`public/logo.svg` — placeholder. Замени на свой `logo.png`/`logo.svg`. Если меняешь имя — поправь:
- `index.html` (favicon)
- `src/components/Header/Header.template.ts`
- `src/components/Footer/Footer.template.ts`

## Структура

```
src/
├── api/          # Fetch-клиент + типы DTO + per-domain API (auth, catalog, favorites, orders, admin)
├── components/   # Header, Footer, ProductCard, Toast, Modal
├── pages/        # Home, Catalog, ProductDetail, Cart, Auth/{Login,Register}, Profile,
│                 # Favorites, Orders/{MyOrders,OrderDetail}, Admin/*, Errors/{NotFound,Forbidden}
├── router/       # History API роутер с поддержкой :params + query
├── store/        # auth (currentUser в памяти), cart (localStorage)
├── styles/       # _variables.scss, global.scss
├── utils/        # config, template (Handlebars + helpers), markdown (marked)
└── main.ts       # точка входа: грузит /me, рендерит шапку/футер, ставит роуты
```

## Auth-логика

- Сессия — http-only cookie, выставляется бэком при `/auth/login` или `/auth/register`. Браузер шлёт cookie автоматически.
- `authStore.init()` дёргает `/me` на старте — заполняет данные пользователя.
- Защищённые страницы (`/profile`, `/orders`, `/favorites`, `/admin/*`, `/cart` (для submit)) при отсутствии сессии редиректят на `/login?next=<цель>`.
- Админ-страницы доп. проверяют `role === 'admin'` → 403 если нет.

## Корзина

- Хранится во фронте, в localStorage (`yulik3d:cart:v1`).
- Гость может листать товары, но добавление в корзину и оформление — только после `/login`.
- При оформлении заказа фронт шлёт `item_id`, `quantity`, `option_ids` — бэк сам пересчитывает все цены из БД.

## Контактные данные

Лежат в [src/utils/config.ts](src/utils/config.ts) (с дефолтами на основе данных пользователя). Можно переопределить через env:
- `VITE_CONTACT_EMAIL`
- `VITE_CONTACT_VK`
- `VITE_CONTACT_TG`
- `VITE_CONTACT_INSTAGRAM`
