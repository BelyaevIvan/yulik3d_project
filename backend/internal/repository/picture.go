package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"yulik3d/internal/model"
)

type PictureRepo struct{ db *DB }

func NewPictureRepo(db *DB) *PictureRepo { return &PictureRepo{db: db} }

// CreatePicture вставляет запись в picture.
func (r *PictureRepo) CreatePicture(ctx context.Context, p *model.Picture) error {
	const q = `INSERT INTO picture (id, object_key) VALUES ($1, $2) RETURNING created_at, updated_at`
	return r.db.QueryRow(ctx, q, p.ID, p.ObjectKey).Scan(&p.CreatedAt, &p.UpdatedAt)
}

// AttachToItem создаёт связь item_picture с position.
func (r *PictureRepo) AttachToItem(ctx context.Context, itemID, pictureID uuid.UUID, position int) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO item_picture (item_id, picture_id, position) VALUES ($1, $2, $3)`,
		itemID, pictureID, position)
	return err
}

// NextPosition возвращает MAX(position)+1 для галереи товара.
func (r *PictureRepo) NextPosition(ctx context.Context, itemID uuid.UUID) (int, error) {
	var maxPos int
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(MAX(position), 0) FROM item_picture WHERE item_id = $1`,
		itemID).Scan(&maxPos)
	if err != nil {
		return 0, err
	}
	return maxPos + 1, nil
}

// PictureRow — объединённая строка item_picture + picture.
type PictureRow struct {
	PictureID uuid.UUID
	ObjectKey string
	Position  int
}

// ListByItem возвращает все картинки товара упорядоченные по position.
func (r *PictureRepo) ListByItem(ctx context.Context, itemID uuid.UUID) ([]PictureRow, error) {
	const q = `
		SELECT p.id, p.object_key, ip.position
		FROM item_picture ip JOIN picture p ON p.id = ip.picture_id
		WHERE ip.item_id = $1
		ORDER BY ip.position`
	rows, err := r.db.Query(ctx, q, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PictureRow
	for rows.Next() {
		var x PictureRow
		if err := rows.Scan(&x.PictureID, &x.ObjectKey, &x.Position); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// PrimaryPictureKeys — для батч-запроса титульных картинок по списку товаров.
// Возвращает map[item_id]object_key (только если есть картинки).
func (r *PictureRepo) PrimaryPictureKeys(ctx context.Context, itemIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	if len(itemIDs) == 0 {
		return map[uuid.UUID]string{}, nil
	}
	const q = `
		SELECT DISTINCT ON (ip.item_id) ip.item_id, p.object_key
		FROM item_picture ip JOIN picture p ON p.id = ip.picture_id
		WHERE ip.item_id = ANY($1)
		ORDER BY ip.item_id, ip.position`
	rows, err := r.db.Query(ctx, q, itemIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID]string, len(itemIDs))
	for rows.Next() {
		var itemID uuid.UUID
		var key string
		if err := rows.Scan(&itemID, &key); err != nil {
			return nil, err
		}
		out[itemID] = key
	}
	return out, rows.Err()
}

// DeleteLink удаляет связь. Возвращает true если была.
func (r *PictureRepo) DeleteLink(ctx context.Context, itemID, pictureID uuid.UUID) (bool, error) {
	ct, err := r.db.Exec(ctx, `DELETE FROM item_picture WHERE item_id = $1 AND picture_id = $2`, itemID, pictureID)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}

// CountLinks возвращает количество ссылок на картинку (чтобы решить — удалять
// запись picture или нет).
func (r *PictureRepo) CountLinks(ctx context.Context, pictureID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM item_picture WHERE picture_id = $1`, pictureID).Scan(&n)
	return n, err
}

// DeletePicture удаляет запись picture (и возвращает object_key, чтобы удалить файл в MinIO).
func (r *PictureRepo) DeletePicture(ctx context.Context, id uuid.UUID) (string, error) {
	var key string
	err := r.db.QueryRow(ctx, `DELETE FROM picture WHERE id = $1 RETURNING object_key`, id).Scan(&key)
	if IsNoRows(err) {
		return "", pgx.ErrNoRows
	}
	return key, err
}

// UpdatePosition меняет позицию связи.
func (r *PictureRepo) UpdatePosition(ctx context.Context, itemID, pictureID uuid.UUID, position int) error {
	_, err := r.db.Exec(ctx,
		`UPDATE item_picture SET position = $3 WHERE item_id = $1 AND picture_id = $2`,
		itemID, pictureID, position)
	return err
}
