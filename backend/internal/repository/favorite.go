package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"yulik3d/internal/model"
)

type FavoriteRepo struct{ db *DB }

func NewFavoriteRepo(db *DB) *FavoriteRepo { return &FavoriteRepo{db: db} }

// Add — идемпотентная вставка. Возвращает created_at (как фактический, так и из ON CONFLICT).
func (r *FavoriteRepo) Add(ctx context.Context, userID, itemID uuid.UUID) (time.Time, error) {
	const q = `
		INSERT INTO favorite (user_id, item_id) VALUES ($1, $2)
		ON CONFLICT (user_id, item_id) DO UPDATE SET user_id = EXCLUDED.user_id
		RETURNING created_at`
	var t time.Time
	err := r.db.QueryRow(ctx, q, userID, itemID).Scan(&t)
	return t, err
}

// Remove — идемпотентное удаление. Возвращает true, если запись была.
func (r *FavoriteRepo) Remove(ctx context.Context, userID, itemID uuid.UUID) (bool, error) {
	ct, err := r.db.Exec(ctx, `DELETE FROM favorite WHERE user_id = $1 AND item_id = $2`, userID, itemID)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}

// Count — общее число избранного пользователя (для пагинации).
func (r *FavoriteRepo) Count(ctx context.Context, userID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM favorite WHERE user_id = $1`, userID).Scan(&n)
	return n, err
}

// ListItems возвращает товары из избранного с пагинацией, новые сверху.
func (r *FavoriteRepo) ListItems(ctx context.Context, userID uuid.UUID, p model.Pagination) ([]model.Item, error) {
	const q = `
		SELECT ` + `i.id, i.name, i.description_info, i.description_other, i.price, i.sale, i.articul, i.hidden, i.created_at, i.updated_at` + `
		FROM favorite f JOIN item i ON i.id = f.item_id
		WHERE f.user_id = $1
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, q, userID, p.Limit, p.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Item
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
