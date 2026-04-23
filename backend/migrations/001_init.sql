-- +goose Up
-- +goose StatementBegin

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

-- ============================================================
-- INDEXES (критичные под текущие запросы)
-- ============================================================

CREATE INDEX idx_item_hidden           ON item(hidden);
CREATE INDEX idx_subcategory_category  ON subcategory(category_id);
CREATE INDEX idx_item_picture_item     ON item_picture(item_id, position);
CREATE INDEX idx_item_option_item      ON item_option(item_id, type_id, position);
CREATE INDEX idx_item_subcategory_sub  ON item_subcategory(subcategory_id);
CREATE INDEX idx_favorite_user_created ON favorite(user_id, created_at DESC);
CREATE INDEX idx_order_user_created    ON "order"(user_id, created_at DESC);
CREATE INDEX idx_order_status_created  ON "order"(status, created_at DESC);
CREATE INDEX idx_order_item_order      ON order_item(order_id);
CREATE INDEX idx_order_item_option_oi  ON order_item_option(order_item_id);

-- ============================================================
-- ARTICUL GENERATION (для item.articul)
-- ============================================================

CREATE SEQUENCE item_articul_seq START 1;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP SEQUENCE IF EXISTS item_articul_seq;

DROP TABLE IF EXISTS order_item_option;
DROP TABLE IF EXISTS order_item;
DROP TABLE IF EXISTS "order";
DROP TABLE IF EXISTS favorite;
DROP TABLE IF EXISTS "user";
DROP TABLE IF EXISTS item_subcategory;
DROP TABLE IF EXISTS subcategory;
DROP TABLE IF EXISTS category;
DROP TABLE IF EXISTS item_option;
DROP TABLE IF EXISTS option_type;
DROP TABLE IF EXISTS item_picture;
DROP TABLE IF EXISTS picture;
DROP TABLE IF EXISTS item;

DROP TYPE IF EXISTS order_status;
DROP TYPE IF EXISTS user_role;
DROP TYPE IF EXISTS category_type;

-- +goose StatementEnd
