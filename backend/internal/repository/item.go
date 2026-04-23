package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"yulik3d/internal/model"
)

type ItemRepo struct{ db *DB }

func NewItemRepo(db *DB) *ItemRepo { return &ItemRepo{db: db} }

const itemCols = `id, name, description_info, description_other, price, sale, articul, hidden, created_at, updated_at`

func scanItem(row pgx.Row) (model.Item, error) {
	var i model.Item
	err := row.Scan(&i.ID, &i.Name, &i.DescriptionInfo, &i.DescriptionOther, &i.Price, &i.Sale,
		&i.Articul, &i.Hidden, &i.CreatedAt, &i.UpdatedAt)
	return i, err
}

func (r *ItemRepo) Create(ctx context.Context, it *model.Item) error {
	const q = `
		INSERT INTO item (id, name, description_info, description_other, price, sale, articul, hidden)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at`
	return r.db.QueryRow(ctx, q,
		it.ID, it.Name, it.DescriptionInfo, it.DescriptionOther,
		it.Price, it.Sale, it.Articul, it.Hidden).Scan(&it.CreatedAt, &it.UpdatedAt)
}

func (r *ItemRepo) GetByID(ctx context.Context, id uuid.UUID) (model.Item, error) {
	const q = `SELECT ` + itemCols + ` FROM item WHERE id = $1`
	return scanItem(r.db.QueryRow(ctx, q, id))
}

// NextArticul возвращает строку "CAT-00042" на основе item_articul_seq.
func (r *ItemRepo) NextArticul(ctx context.Context) (string, error) {
	var n int64
	if err := r.db.QueryRow(ctx, `SELECT nextval('item_articul_seq')`).Scan(&n); err != nil {
		return "", err
	}
	return fmt.Sprintf("CAT-%05d", n), nil
}

// Update — полное обновление через PUT (без articul — он неизменяем).
func (r *ItemRepo) Update(ctx context.Context, it *model.Item) error {
	const q = `
		UPDATE item
		SET name = $2, description_info = $3, description_other = $4,
		    price = $5, sale = $6, hidden = $7, updated_at = NOW()
		WHERE id = $1
		RETURNING created_at, updated_at`
	return r.db.QueryRow(ctx, q,
		it.ID, it.Name, it.DescriptionInfo, it.DescriptionOther,
		it.Price, it.Sale, it.Hidden).Scan(&it.CreatedAt, &it.UpdatedAt)
}

// Patch — частичное обновление. Передаются только ненулевые указатели.
func (r *ItemRepo) Patch(ctx context.Context, id uuid.UUID, p model.ItemPatchRequest) (model.Item, error) {
	const q = `
		UPDATE item
		SET name              = COALESCE($2, name),
		    description_info  = COALESCE($3, description_info),
		    description_other = COALESCE($4, description_other),
		    price             = COALESCE($5, price),
		    sale              = COALESCE($6, sale),
		    hidden            = COALESCE($7, hidden),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING ` + itemCols
	return scanItem(r.db.QueryRow(ctx, q, id, p.Name, p.DescriptionInfo, p.DescriptionOther, p.Price, p.Sale, p.Hidden))
}

// ReplaceSubcategories — удалить все item_subcategory и вставить новые (в транзакции).
func (r *ItemRepo) ReplaceSubcategories(ctx context.Context, itemID uuid.UUID, subIDs []uuid.UUID) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM item_subcategory WHERE item_id = $1`, itemID); err != nil {
		return err
	}
	return r.AttachSubcategories(ctx, itemID, subIDs)
}

// AttachSubcategories — bulk-insert связей.
func (r *ItemRepo) AttachSubcategories(ctx context.Context, itemID uuid.UUID, subIDs []uuid.UUID) error {
	if len(subIDs) == 0 {
		return nil
	}
	// Собираем VALUES ($1,$2), ($1,$3), ...
	var b strings.Builder
	b.WriteString(`INSERT INTO item_subcategory (item_id, subcategory_id) VALUES `)
	args := make([]any, 0, len(subIDs)+1)
	args = append(args, itemID)
	for i, sid := range subIDs {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "($1, $%d)", i+2)
		args = append(args, sid)
	}
	b.WriteString(` ON CONFLICT DO NOTHING`)
	_, err := r.db.Exec(ctx, b.String(), args...)
	return err
}

// ---- Каталог ----

// ListFilter — SQL-строка + args для WHERE (без префикса WHERE).
func buildCatalogWhere(f model.CatalogFilter) (string, []any) {
	var cond []string
	var args []any
	i := 1

	if !f.IncludeHidden {
		cond = append(cond, "i.hidden = false")
	} else if f.HiddenOnly != nil {
		cond = append(cond, fmt.Sprintf("i.hidden = $%d", i))
		args = append(args, *f.HiddenOnly)
		i++
	}

	if f.CategoryType != nil {
		cond = append(cond, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM item_subcategory isc
			JOIN subcategory sc ON sc.id = isc.subcategory_id
			JOIN category c ON c.id = sc.category_id
			WHERE isc.item_id = i.id AND c.type = $%d
		)`, i))
		args = append(args, *f.CategoryType)
		i++
	}
	if f.CategoryID != nil {
		cond = append(cond, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM item_subcategory isc
			JOIN subcategory sc ON sc.id = isc.subcategory_id
			WHERE isc.item_id = i.id AND sc.category_id = $%d
		)`, i))
		args = append(args, *f.CategoryID)
		i++
	}
	if f.SubcategoryID != nil {
		cond = append(cond, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM item_subcategory isc WHERE isc.item_id = i.id AND isc.subcategory_id = $%d
		)`, i))
		args = append(args, *f.SubcategoryID)
		i++
	}
	if f.Query != "" {
		cond = append(cond, fmt.Sprintf("i.name ILIKE $%d", i))
		args = append(args, "%"+f.Query+"%")
		i++
	}
	if f.HasSale != nil {
		if *f.HasSale {
			cond = append(cond, "i.sale > 0")
		} else {
			cond = append(cond, "i.sale = 0")
		}
	}

	if len(cond) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(cond, " AND "), args
}

func orderByClause(sort string) string {
	switch sort {
	case "price_asc":
		return "ORDER BY (i.price * (100 - i.sale) / 100) ASC, i.id"
	case "price_desc":
		return "ORDER BY (i.price * (100 - i.sale) / 100) DESC, i.id"
	case "name_asc":
		return "ORDER BY i.name ASC, i.id"
	case "name_desc":
		return "ORDER BY i.name DESC, i.id"
	case "created_asc":
		return "ORDER BY i.created_at ASC, i.id"
	default: // created_desc
		return "ORDER BY i.created_at DESC, i.id"
	}
}

func (r *ItemRepo) Count(ctx context.Context, f model.CatalogFilter) (int, error) {
	where, args := buildCatalogWhere(f)
	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM item i`+where, args...).Scan(&total)
	return total, err
}

func (r *ItemRepo) List(ctx context.Context, f model.CatalogFilter) ([]model.Item, error) {
	where, args := buildCatalogWhere(f)
	limIdx := len(args) + 1
	offIdx := len(args) + 2
	q := `SELECT ` + renameItemCols("i.") + ` FROM item i` + where + ` ` + orderByClause(f.Sort) +
		fmt.Sprintf(` LIMIT $%d OFFSET $%d`, limIdx, offIdx)
	args = append(args, f.Pagination.Limit, f.Pagination.Offset)
	rows, err := r.db.Query(ctx, q, args...)
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

// renameItemCols — превращает "id, name, ..." в "i.id, i.name, ...".
func renameItemCols(prefix string) string {
	parts := strings.Split(itemCols, ", ")
	for i, p := range parts {
		parts[i] = prefix + p
	}
	return strings.Join(parts, ", ")
}
