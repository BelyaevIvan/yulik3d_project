package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"yulik3d/internal/model"
)

// ItemMainPin — закрепление товара на главной для конкретного типа.
type ItemMainPin struct {
	ItemID   uuid.UUID
	Type     model.CategoryType
	Position int
}

type ItemMainPinRepo struct{ db *DB }

func NewItemMainPinRepo(db *DB) *ItemMainPinRepo { return &ItemMainPinRepo{db: db} }

// ErrPositionTaken — позиция уже занята другим товаром в этом типе.
var ErrPositionTaken = errors.New("main_pin: position already taken")

// ListByType — все закрепления одного типа, отсортированные по position.
func (r *ItemMainPinRepo) ListByType(ctx context.Context, t model.CategoryType) ([]ItemMainPin, error) {
	const q = `SELECT item_id, type, position FROM item_main_pin WHERE type = $1 ORDER BY position`
	rows, err := r.db.Query(ctx, q, t)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ItemMainPin
	for rows.Next() {
		var p ItemMainPin
		if err := rows.Scan(&p.ItemID, &p.Type, &p.Position); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListByItem — все закрепления одного товара (может быть в обоих типах).
func (r *ItemMainPinRepo) ListByItem(ctx context.Context, itemID uuid.UUID) ([]ItemMainPin, error) {
	const q = `SELECT item_id, type, position FROM item_main_pin WHERE item_id = $1`
	rows, err := r.db.Query(ctx, q, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ItemMainPin
	for rows.Next() {
		var p ItemMainPin
		if err := rows.Scan(&p.ItemID, &p.Type, &p.Position); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// CountByType — сколько товаров уже закреплено в типе (для лимита 5).
func (r *ItemMainPinRepo) CountByType(ctx context.Context, t model.CategoryType) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM item_main_pin WHERE type = $1`, t).Scan(&n)
	return n, err
}

// Insert — закрепить товар в типе на конкретной позиции. Если позиция занята
// другим товаром — вернёт ErrPositionTaken (UNIQUE violation).
// Если товар уже закреплён в этом типе — вернёт ErrPositionTaken (PK violation).
func (r *ItemMainPinRepo) Insert(ctx context.Context, p ItemMainPin) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO item_main_pin (item_id, type, position) VALUES ($1, $2, $3)`,
		p.ItemID, p.Type, p.Position)
	if err != nil && PgErrCode(err) == PgCodeUniqueViolation {
		return ErrPositionTaken
	}
	return err
}

// DeleteByItemAndType — снять закрепление одного товара в одном типе.
// Возвращает true если запись была удалена, false если её не было.
func (r *ItemMainPinRepo) DeleteByItemAndType(ctx context.Context, itemID uuid.UUID, t model.CategoryType) (bool, error) {
	ct, err := r.db.Exec(ctx, `DELETE FROM item_main_pin WHERE item_id = $1 AND type = $2`, itemID, t)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}

// DeleteByItem — снять все закрепления товара (вызывается при скрытии).
func (r *ItemMainPinRepo) DeleteByItem(ctx context.Context, itemID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM item_main_pin WHERE item_id = $1`, itemID)
	return err
}

// Compact — уплотнить позиции в типе после удаления слота: 1, 3, 4 → 1, 2, 3.
//
// Реализован через DELETE + bulk INSERT (а не через UPDATE), чтобы избежать
// промежуточных состояний, нарушающих CHECK (position BETWEEN 1 AND 5) или
// UNIQUE (type, position). Любой UPDATE-трюк упирался бы в один из них.
//
// Должен вызываться внутри транзакции (например, после Unpin) — иначе между
// DELETE и INSERT для главной возможна короткая видимость пустого списка.
func (r *ItemMainPinRepo) Compact(ctx context.Context, t model.CategoryType) error {
	current, err := r.ListByType(ctx, t)
	if err != nil {
		return fmt.Errorf("list current: %w", err)
	}
	if len(current) == 0 {
		return nil
	}
	pins := make([]ItemMainPin, len(current))
	for i, c := range current {
		pins[i] = ItemMainPin{ItemID: c.ItemID, Type: t, Position: i + 1}
	}
	return r.ReplaceForType(ctx, t, pins)
}

// ReplaceForType — атомарная замена всех закреплений одного типа на новый набор.
// Используется при drag-and-drop reorder.
//
// ВАЖНО: вызывать ТОЛЬКО внутри tx.Run (нужна транзакция для целостности).
func (r *ItemMainPinRepo) ReplaceForType(ctx context.Context, t model.CategoryType, pins []ItemMainPin) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM item_main_pin WHERE type = $1`, t); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if len(pins) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString(`INSERT INTO item_main_pin (item_id, type, position) VALUES `)
	args := make([]any, 0, len(pins)*3)
	for i, p := range pins {
		if i > 0 {
			b.WriteString(",")
		}
		idx := i*3 + 1
		fmt.Fprintf(&b, "($%d, $%d, $%d)", idx, idx+1, idx+2)
		args = append(args, p.ItemID, p.Type, p.Position)
	}
	if _, err := r.db.Exec(ctx, b.String(), args...); err != nil {
		if PgErrCode(err) == PgCodeUniqueViolation {
			return ErrPositionTaken
		}
		return fmt.Errorf("insert batch: %w", err)
	}
	return nil
}
