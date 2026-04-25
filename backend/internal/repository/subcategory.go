package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"yulik3d/internal/model"
)

type SubcategoryRepo struct{ db *DB }

func NewSubcategoryRepo(db *DB) *SubcategoryRepo { return &SubcategoryRepo{db: db} }

func scanSub(row pgx.Row) (model.Subcategory, error) {
	var s model.Subcategory
	err := row.Scan(&s.ID, &s.Name, &s.CategoryID, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

func (r *SubcategoryRepo) Create(ctx context.Context, s *model.Subcategory) error {
	const q = `INSERT INTO subcategory (id, name, category_id) VALUES ($1, $2, $3) RETURNING created_at, updated_at`
	return r.db.QueryRow(ctx, q, s.ID, s.Name, s.CategoryID).Scan(&s.CreatedAt, &s.UpdatedAt)
}

func (r *SubcategoryRepo) GetByID(ctx context.Context, id uuid.UUID) (model.Subcategory, error) {
	const q = `SELECT id, name, category_id, created_at, updated_at FROM subcategory WHERE id = $1`
	return scanSub(r.db.QueryRow(ctx, q, id))
}

func (r *SubcategoryRepo) ListByCategory(ctx context.Context, categoryID uuid.UUID) ([]model.Subcategory, error) {
	const q = `SELECT id, name, category_id, created_at, updated_at FROM subcategory WHERE category_id = $1 ORDER BY name`
	rows, err := r.db.Query(ctx, q, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Subcategory
	for rows.Next() {
		s, err := scanSub(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListByCategoryIDs — батч для GET /categories?with_subcategories=true.
func (r *SubcategoryRepo) ListByCategoryIDs(ctx context.Context, ids []uuid.UUID) ([]model.Subcategory, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	const q = `SELECT id, name, category_id, created_at, updated_at FROM subcategory WHERE category_id = ANY($1) ORDER BY name`
	rows, err := r.db.Query(ctx, q, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Subcategory
	for rows.Next() {
		s, err := scanSub(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ExistIDs — проверяет, что все переданные ID существуют. Используется при
// создании/обновлении item c subcategory_ids.
func (r *SubcategoryRepo) ExistIDs(ctx context.Context, ids []uuid.UUID) (bool, error) {
	if len(ids) == 0 {
		return true, nil
	}
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM subcategory WHERE id = ANY($1)`, ids).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == len(ids), nil
}

func (r *SubcategoryRepo) Patch(ctx context.Context, id uuid.UUID, name *string, categoryID *uuid.UUID) (model.Subcategory, error) {
	const q = `
		UPDATE subcategory
		SET name        = COALESCE($2, name),
		    category_id = COALESCE($3, category_id),
		    updated_at  = NOW()
		WHERE id = $1
		RETURNING id, name, category_id, created_at, updated_at`
	return scanSub(r.db.QueryRow(ctx, q, id, name, categoryID))
}

func (r *SubcategoryRepo) Delete(ctx context.Context, id uuid.UUID) (bool, error) {
	ct, err := r.db.Exec(ctx, `DELETE FROM subcategory WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}
