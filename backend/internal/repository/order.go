package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"yulik3d/internal/model"
)

type OrderRepo struct{ db *DB }

func NewOrderRepo(db *DB) *OrderRepo { return &OrderRepo{db: db} }

const orderCols = `id, user_id, status, total_price, customer_comment, admin_note, contact_phone, contact_full_name, created_at, updated_at`

func scanOrder(row pgx.Row) (model.Order, error) {
	var o model.Order
	err := row.Scan(&o.ID, &o.UserID, &o.Status, &o.TotalPrice, &o.CustomerComment, &o.AdminNote,
		&o.ContactPhone, &o.ContactFullName, &o.CreatedAt, &o.UpdatedAt)
	return o, err
}

// CreateOrder вставляет заказ. ID должен быть сгенерирован на вызывающей стороне.
func (r *OrderRepo) CreateOrder(ctx context.Context, o *model.Order) error {
	const q = `
		INSERT INTO "order" (id, user_id, status, total_price, customer_comment, admin_note, contact_phone, contact_full_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at`
	return r.db.QueryRow(ctx, q, o.ID, o.UserID, o.Status, o.TotalPrice,
		o.CustomerComment, o.AdminNote, o.ContactPhone, o.ContactFullName).Scan(&o.CreatedAt, &o.UpdatedAt)
}

// CreateOrderItem вставляет позицию заказа.
func (r *OrderRepo) CreateOrderItem(ctx context.Context, oi *model.OrderItem) error {
	const q = `
		INSERT INTO order_item (id, order_id, item_id, quantity, unit_base_price, unit_total_price,
		                         item_name_snapshot, item_articul_snapshot)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.Exec(ctx, q, oi.ID, oi.OrderID, oi.ItemID, oi.Quantity,
		oi.UnitBasePrice, oi.UnitTotalPrice, oi.ItemNameSnapshot, oi.ItemArticulSnapshot)
	return err
}

// CreateOrderItemOption — вставляет снапшот опции.
func (r *OrderRepo) CreateOrderItemOption(ctx context.Context, oio *model.OrderItemOption) error {
	const q = `
		INSERT INTO order_item_option (id, order_item_id, type_code_snapshot, type_label_snapshot, value_snapshot, price_snapshot)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.Exec(ctx, q, oio.ID, oio.OrderItemID, oio.TypeCodeSnapshot,
		oio.TypeLabelSnapshot, oio.ValueSnapshot, oio.PriceSnapshot)
	return err
}

// GetByID — без ownership проверки, её делает сервис.
func (r *OrderRepo) GetByID(ctx context.Context, id uuid.UUID) (model.Order, error) {
	const q = `SELECT ` + orderCols + ` FROM "order" WHERE id = $1`
	return scanOrder(r.db.QueryRow(ctx, q, id))
}

// UpdateStatus меняет статус (без проверки перехода — это делает сервис).
func (r *OrderRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.OrderStatus) (model.Order, error) {
	const q = `
		UPDATE "order" SET status = $2, updated_at = NOW() WHERE id = $1
		RETURNING ` + orderCols
	return scanOrder(r.db.QueryRow(ctx, q, id, status))
}

// UpdateAdminNote — патч admin_note.
func (r *OrderRepo) UpdateAdminNote(ctx context.Context, id uuid.UUID, note *string) (model.Order, error) {
	const q = `
		UPDATE "order" SET admin_note = $2, updated_at = NOW() WHERE id = $1
		RETURNING ` + orderCols
	return scanOrder(r.db.QueryRow(ctx, q, id, note))
}

// ListOrderItems — позиции заказа.
func (r *OrderRepo) ListOrderItems(ctx context.Context, orderID uuid.UUID) ([]model.OrderItem, error) {
	const q = `
		SELECT id, order_id, item_id, quantity, unit_base_price, unit_total_price,
		       item_name_snapshot, item_articul_snapshot
		FROM order_item WHERE order_id = $1 ORDER BY id`
	rows, err := r.db.Query(ctx, q, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.OrderItem
	for rows.Next() {
		var oi model.OrderItem
		if err := rows.Scan(&oi.ID, &oi.OrderID, &oi.ItemID, &oi.Quantity,
			&oi.UnitBasePrice, &oi.UnitTotalPrice, &oi.ItemNameSnapshot, &oi.ItemArticulSnapshot); err != nil {
			return nil, err
		}
		out = append(out, oi)
	}
	return out, rows.Err()
}

// ListOrderItemOptions — батч опций по нескольким order_item_id.
func (r *OrderRepo) ListOrderItemOptions(ctx context.Context, orderItemIDs []uuid.UUID) (map[uuid.UUID][]model.OrderItemOption, error) {
	out := make(map[uuid.UUID][]model.OrderItemOption)
	if len(orderItemIDs) == 0 {
		return out, nil
	}
	const q = `
		SELECT id, order_item_id, type_code_snapshot, type_label_snapshot, value_snapshot, price_snapshot
		FROM order_item_option WHERE order_item_id = ANY($1)`
	rows, err := r.db.Query(ctx, q, orderItemIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var o model.OrderItemOption
		if err := rows.Scan(&o.ID, &o.OrderItemID, &o.TypeCodeSnapshot, &o.TypeLabelSnapshot,
			&o.ValueSnapshot, &o.PriceSnapshot); err != nil {
			return nil, err
		}
		out[o.OrderItemID] = append(out[o.OrderItemID], o)
	}
	return out, rows.Err()
}

// ---- User history ----

func (r *OrderRepo) CountForUser(ctx context.Context, userID uuid.UUID, status *model.OrderStatus) (int, error) {
	q := `SELECT COUNT(*) FROM "order" WHERE user_id = $1`
	args := []any{userID}
	if status != nil {
		q += ` AND status = $2`
		args = append(args, *status)
	}
	var n int
	err := r.db.QueryRow(ctx, q, args...).Scan(&n)
	return n, err
}

// ListForUser — список заказов пользователя с items_count.
func (r *OrderRepo) ListForUser(ctx context.Context, userID uuid.UUID, status *model.OrderStatus, p model.Pagination) ([]model.OrderListItemDTO, error) {
	q := `
		SELECT o.id, o.status, o.total_price,
		       (SELECT COUNT(*) FROM order_item oi WHERE oi.order_id = o.id) AS items_count,
		       o.created_at, o.updated_at
		FROM "order" o WHERE o.user_id = $1`
	args := []any{userID}
	if status != nil {
		q += ` AND o.status = $2`
		args = append(args, *status)
	}
	q += fmt.Sprintf(` ORDER BY o.created_at DESC LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	args = append(args, p.Limit, p.Offset)
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.OrderListItemDTO
	for rows.Next() {
		var d model.OrderListItemDTO
		if err := rows.Scan(&d.ID, &d.Status, &d.TotalPrice, &d.ItemsCount, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ---- Admin list ----

func (r *OrderRepo) CountAdmin(ctx context.Context, f model.OrderAdminListFilter) (int, error) {
	where, args := buildOrderAdminWhere(f)
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM "order" o`+where, args...).Scan(&n)
	return n, err
}

func (r *OrderRepo) ListAdmin(ctx context.Context, f model.OrderAdminListFilter) ([]model.OrderAdminListItemDTO, error) {
	where, args := buildOrderAdminWhere(f)
	q := `
		SELECT o.id, u.id, u.email, u.full_name,
		       o.status, o.total_price,
		       (SELECT COUNT(*) FROM order_item oi WHERE oi.order_id = o.id) AS items_count,
		       o.contact_phone, o.contact_full_name,
		       o.customer_comment, o.admin_note,
		       o.created_at, o.updated_at
		FROM "order" o JOIN "user" u ON u.id = o.user_id` + where +
		fmt.Sprintf(` ORDER BY o.created_at DESC LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	args = append(args, f.Pagination.Limit, f.Pagination.Offset)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.OrderAdminListItemDTO
	for rows.Next() {
		var d model.OrderAdminListItemDTO
		if err := rows.Scan(&d.ID, &d.User.ID, &d.User.Email, &d.User.FullName,
			&d.Status, &d.TotalPrice, &d.ItemsCount,
			&d.ContactPhone, &d.ContactFullName,
			&d.CustomerComment, &d.AdminNote,
			&d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func buildOrderAdminWhere(f model.OrderAdminListFilter) (string, []any) {
	var cond []string
	var args []any
	i := 1
	if f.Status != nil {
		cond = append(cond, fmt.Sprintf("o.status = $%d", i))
		args = append(args, *f.Status)
		i++
	}
	if f.UserID != nil {
		cond = append(cond, fmt.Sprintf("o.user_id = $%d", i))
		args = append(args, *f.UserID)
		i++
	}
	if f.Query != "" {
		cond = append(cond, fmt.Sprintf("(o.contact_full_name ILIKE $%d OR o.contact_phone ILIKE $%d)", i, i))
		args = append(args, "%"+f.Query+"%")
		i++
	}
	if len(cond) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(cond, " AND "), args
}
