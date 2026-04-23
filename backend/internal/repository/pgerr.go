package repository

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Вспомогательные помощники для классификации ошибок Postgres.

const (
	PgCodeUniqueViolation     = "23505"
	PgCodeForeignKeyViolation = "23503"
	PgCodeCheckViolation      = "23514"
	PgCodeNotNullViolation    = "23502"
)

// IsNoRows — true, если rows не нашлись.
func IsNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// PgErrCode возвращает SQLSTATE или пустую строку.
func PgErrCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}
