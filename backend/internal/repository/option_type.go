package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"yulik3d/internal/model"
)

type OptionTypeRepo struct{ db *DB }

func NewOptionTypeRepo(db *DB) *OptionTypeRepo { return &OptionTypeRepo{db: db} }

func scanOptionType(row pgx.Row) (model.OptionType, error) {
	var o model.OptionType
	err := row.Scan(&o.ID, &o.Code, &o.Label, &o.CreatedAt, &o.UpdatedAt)
	return o, err
}

func (r *OptionTypeRepo) Create(ctx context.Context, o *model.OptionType) error {
	const q = `INSERT INTO option_type (id, code, label) VALUES ($1, $2, $3) RETURNING created_at, updated_at`
	return r.db.QueryRow(ctx, q, o.ID, o.Code, o.Label).Scan(&o.CreatedAt, &o.UpdatedAt)
}

func (r *OptionTypeRepo) GetByID(ctx context.Context, id uuid.UUID) (model.OptionType, error) {
	const q = `SELECT id, code, label, created_at, updated_at FROM option_type WHERE id = $1`
	return scanOptionType(r.db.QueryRow(ctx, q, id))
}

func (r *OptionTypeRepo) List(ctx context.Context) ([]model.OptionType, error) {
	const q = `SELECT id, code, label, created_at, updated_at FROM option_type ORDER BY label`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.OptionType
	for rows.Next() {
		o, err := scanOptionType(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// ListByIDs — для батч-загрузки типов в детальной карточке товара.
func (r *OptionTypeRepo) ListByIDs(ctx context.Context, ids []uuid.UUID) ([]model.OptionType, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	const q = `SELECT id, code, label, created_at, updated_at FROM option_type WHERE id = ANY($1)`
	rows, err := r.db.Query(ctx, q, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.OptionType
	for rows.Next() {
		o, err := scanOptionType(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (r *OptionTypeRepo) PatchLabel(ctx context.Context, id uuid.UUID, label string) (model.OptionType, error) {
	const q = `UPDATE option_type SET label = $2, updated_at = NOW() WHERE id = $1
	           RETURNING id, code, label, created_at, updated_at`
	return scanOptionType(r.db.QueryRow(ctx, q, id, label))
}

func (r *OptionTypeRepo) Delete(ctx context.Context, id uuid.UUID) (bool, error) {
	ct, err := r.db.Exec(ctx, `DELETE FROM option_type WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}
