package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"yulik3d/internal/model"
)

type CategoryRepo struct{ db *DB }

func NewCategoryRepo(db *DB) *CategoryRepo { return &CategoryRepo{db: db} }

func scanCategory(row pgx.Row) (model.Category, error) {
	var c model.Category
	err := row.Scan(&c.ID, &c.Name, &c.Type, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (r *CategoryRepo) Create(ctx context.Context, c *model.Category) error {
	const q = `INSERT INTO category (id, name, type) VALUES ($1, $2, $3) RETURNING created_at, updated_at`
	return r.db.QueryRow(ctx, q, c.ID, c.Name, c.Type).Scan(&c.CreatedAt, &c.UpdatedAt)
}

func (r *CategoryRepo) GetByID(ctx context.Context, id uuid.UUID) (model.Category, error) {
	const q = `SELECT id, name, type, created_at, updated_at FROM category WHERE id = $1`
	return scanCategory(r.db.QueryRow(ctx, q, id))
}

func (r *CategoryRepo) List(ctx context.Context, filterType *model.CategoryType) ([]model.Category, error) {
	q := `SELECT id, name, type, created_at, updated_at FROM category`
	var args []any
	if filterType != nil {
		q += ` WHERE type = $1`
		args = append(args, *filterType)
	}
	q += ` ORDER BY name`
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Category
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CategoryRepo) Patch(ctx context.Context, id uuid.UUID, name *string, typ *model.CategoryType) (model.Category, error) {
	const q = `
		UPDATE category
		SET name = COALESCE($2, name),
		    type = COALESCE($3, type),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, type, created_at, updated_at`
	return scanCategory(r.db.QueryRow(ctx, q, id, name, typ))
}

func (r *CategoryRepo) Delete(ctx context.Context, id uuid.UUID) (bool, error) {
	ct, err := r.db.Exec(ctx, `DELETE FROM category WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}

// CountItems — сколько товаров (включая скрытые) числится хотя бы в одной
// подкатегории этой категории. Используется при удалении категории.
func (r *CategoryRepo) CountItems(ctx context.Context, id uuid.UUID) (int, error) {
	const q = `
		SELECT COUNT(DISTINCT isc.item_id)
		FROM item_subcategory isc
		JOIN subcategory sc ON sc.id = isc.subcategory_id
		WHERE sc.category_id = $1`
	var n int
	err := r.db.QueryRow(ctx, q, id).Scan(&n)
	return n, err
}
