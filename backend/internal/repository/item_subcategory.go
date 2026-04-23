package repository

import (
	"context"

	"github.com/google/uuid"

	"yulik3d/internal/model"
)

type ItemSubcategoryRepo struct{ db *DB }

func NewItemSubcategoryRepo(db *DB) *ItemSubcategoryRepo { return &ItemSubcategoryRepo{db: db} }

// ListForItem — все подкатегории товара с их категориями.
func (r *ItemSubcategoryRepo) ListForItem(ctx context.Context, itemID uuid.UUID) ([]model.SubcategoryWithCategoryDTO, error) {
	const q = `
		SELECT sc.id, sc.name, c.id, c.name, c.type
		FROM item_subcategory isc
		JOIN subcategory sc ON sc.id = isc.subcategory_id
		JOIN category c ON c.id = sc.category_id
		WHERE isc.item_id = $1
		ORDER BY c.name, sc.name`
	rows, err := r.db.Query(ctx, q, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.SubcategoryWithCategoryDTO
	for rows.Next() {
		var s model.SubcategoryWithCategoryDTO
		if err := rows.Scan(&s.ID, &s.Name, &s.Category.ID, &s.Category.Name, &s.Category.Type); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// SubcategoryAndCategoryBatch — для каталога: мапа item_id → данные о подкатегориях/первой категории.
type SubcategoryAndCategoryBatch struct {
	Subcategories map[uuid.UUID][]model.SubcategoryShortDTO
	PrimaryCat    map[uuid.UUID]model.CategoryShortDTO
}

// ListForItems — для списка карточек в каталоге. Собирает все подкатегории по
// набору товаров и «первую» категорию (по c.name).
func (r *ItemSubcategoryRepo) ListForItems(ctx context.Context, itemIDs []uuid.UUID) (SubcategoryAndCategoryBatch, error) {
	batch := SubcategoryAndCategoryBatch{
		Subcategories: make(map[uuid.UUID][]model.SubcategoryShortDTO),
		PrimaryCat:    make(map[uuid.UUID]model.CategoryShortDTO),
	}
	if len(itemIDs) == 0 {
		return batch, nil
	}
	const q = `
		SELECT isc.item_id, sc.id, sc.name, c.id, c.name, c.type
		FROM item_subcategory isc
		JOIN subcategory sc ON sc.id = isc.subcategory_id
		JOIN category c ON c.id = sc.category_id
		WHERE isc.item_id = ANY($1)
		ORDER BY isc.item_id, c.name, sc.name`
	rows, err := r.db.Query(ctx, q, itemIDs)
	if err != nil {
		return batch, err
	}
	defer rows.Close()
	for rows.Next() {
		var itemID uuid.UUID
		var sub model.SubcategoryShortDTO
		var cat model.CategoryShortDTO
		if err := rows.Scan(&itemID, &sub.ID, &sub.Name, &cat.ID, &cat.Name, &cat.Type); err != nil {
			return batch, err
		}
		batch.Subcategories[itemID] = append(batch.Subcategories[itemID], sub)
		if _, ok := batch.PrimaryCat[itemID]; !ok {
			batch.PrimaryCat[itemID] = cat
		}
	}
	return batch, rows.Err()
}
