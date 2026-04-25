// Package repository — сырой SQL через pgx. Репы используют DB, который
// прозрачно переключается на *pgx.Tx, если он есть в ctx (через TxManager).
package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------- ctx helpers ----------

type ctxTxKey struct{}

// WithTx кладёт tx в context.
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, ctxTxKey{}, tx)
}

// TxFromCtx достаёт tx из context, если есть.
func TxFromCtx(ctx context.Context) pgx.Tx {
	if v, ok := ctx.Value(ctxTxKey{}).(pgx.Tx); ok {
		return v
	}
	return nil
}

// ---------- TxManager ----------

// TxManager — обёртка над пулом для атомарных операций.
type TxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

// Run выполняет fn внутри транзакции, подкладывая tx в ctx. Коммитит на
// nil-ошибке, откатывает на любой другой.
func (m *TxManager) Run(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	// Уже внутри другой транзакции? Выполнить как есть (savepoint-уровень нам не нужен).
	if TxFromCtx(ctx) != nil {
		return fn(ctx)
	}
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}
		err = tx.Commit(ctx)
	}()
	return fn(WithTx(ctx, tx))
}

// ---------- DB helper ----------

// DB — обёртка над пулом, учитывающая ctx-tx. Все репы его используют.
type DB struct {
	pool *pgxpool.Pool
}

func NewDB(pool *pgxpool.Pool) *DB {
	return &DB{pool: pool}
}

// Exec возвращает CommandTag, чтобы репы могли проверить RowsAffected.
func (d *DB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if tx := TxFromCtx(ctx); tx != nil {
		return tx.Exec(ctx, sql, args...)
	}
	return d.pool.Exec(ctx, sql, args...)
}

func (d *DB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if tx := TxFromCtx(ctx); tx != nil {
		return tx.QueryRow(ctx, sql, args...)
	}
	return d.pool.QueryRow(ctx, sql, args...)
}

func (d *DB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if tx := TxFromCtx(ctx); tx != nil {
		return tx.Query(ctx, sql, args...)
	}
	return d.pool.Query(ctx, sql, args...)
}
