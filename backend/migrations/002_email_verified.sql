-- +goose Up
-- +goose StatementBegin

-- Добавляем колонку email_verified. Существующие пользователи получают FALSE
-- и должны подтвердить email через ссылку, которую можно запросить из UI.
-- Их прошлые заказы остаются нетронутыми — колонка про сам аккаунт.
ALTER TABLE "user" ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE "user" DROP COLUMN email_verified;

-- +goose StatementEnd
