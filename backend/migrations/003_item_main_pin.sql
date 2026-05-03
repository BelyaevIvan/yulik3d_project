-- +goose Up
-- +goose StatementBegin

-- Закрепление товаров на главной странице.
-- Один товар может быть закреплён ОТДЕЛЬНО в каждом типе (figure / other).
-- В рамках одного типа позиции 1..5 уникальны.
CREATE TABLE item_main_pin (
    item_id  UUID NOT NULL REFERENCES item(id) ON DELETE CASCADE,
    type     category_type NOT NULL,
    position INT NOT NULL CHECK (position BETWEEN 1 AND 5),
    PRIMARY KEY (item_id, type),
    UNIQUE (type, position)
);

-- Быстрый SELECT для главной (по типу с сортировкой по position).
CREATE INDEX idx_item_main_pin_type_pos ON item_main_pin(type, position);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS item_main_pin;

-- +goose StatementEnd
