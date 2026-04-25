package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"yulik3d/internal/model"
)

type ItemOptionRepo struct{ db *DB }

func NewItemOptionRepo(db *DB) *ItemOptionRepo { return &ItemOptionRepo{db: db} }

func scanItemOption(row pgx.Row) (model.ItemOption, error) {
	var o model.ItemOption
	err := row.Scan(&o.ID, &o.ItemID, &o.TypeID, &o.Value, &o.Price, &o.Position, &o.CreatedAt, &o.UpdatedAt)
	return o, err
}

const itemOptionCols = `id, item_id, type_id, value, price, position, created_at, updated_at`

func (r *ItemOptionRepo) Create(ctx context.Context, o *model.ItemOption) error {
	const q = `INSERT INTO item_option (id, item_id, type_id, value, price, position)
	           VALUES ($1, $2, $3, $4, $5, $6)
	           RETURNING created_at, updated_at`
	return r.db.QueryRow(ctx, q, o.ID, o.ItemID, o.TypeID, o.Value, o.Price, o.Position).
		Scan(&o.CreatedAt, &o.UpdatedAt)
}

func (r *ItemOptionRepo) GetByID(ctx context.Context, id uuid.UUID) (model.ItemOption, error) {
	const q = `SELECT ` + itemOptionCols + ` FROM item_option WHERE id = $1`
	return scanItemOption(r.db.QueryRow(ctx, q, id))
}

func (r *ItemOptionRepo) GetByIDForItem(ctx context.Context, itemID, id uuid.UUID) (model.ItemOption, error) {
	const q = `SELECT ` + itemOptionCols + ` FROM item_option WHERE id = $1 AND item_id = $2`
	return scanItemOption(r.db.QueryRow(ctx, q, id, itemID))
}

func (r *ItemOptionRepo) ListByItem(ctx context.Context, itemID uuid.UUID) ([]model.ItemOption, error) {
	const q = `SELECT ` + itemOptionCols + ` FROM item_option WHERE item_id = $1 ORDER BY type_id, position`
	rows, err := r.db.Query(ctx, q, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.ItemOption
	for rows.Next() {
		o, err := scanItemOption(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (r *ItemOptionRepo) DeleteByItem(ctx context.Context, itemID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM item_option WHERE item_id = $1`, itemID)
	return err
}

func (r *ItemOptionRepo) Patch(ctx context.Context, id uuid.UUID, value *string, price, position *int) (model.ItemOption, error) {
	const q = `
		UPDATE item_option
		SET value    = COALESCE($2, value),
		    price    = COALESCE($3, price),
		    position = COALESCE($4, position),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING ` + itemOptionCols
	return scanItemOption(r.db.QueryRow(ctx, q, id, value, price, position))
}

func (r *ItemOptionRepo) Delete(ctx context.Context, id uuid.UUID) (bool, error) {
	ct, err := r.db.Exec(ctx, `DELETE FROM item_option WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}
