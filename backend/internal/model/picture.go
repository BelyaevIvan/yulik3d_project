package model

import (
	"time"

	"github.com/google/uuid"
)

// Picture — entity.
type Picture struct {
	ID        uuid.UUID
	ObjectKey string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ItemPicture — связь товара и картинки (для получения позиции).
type ItemPicture struct {
	ItemID    uuid.UUID
	PictureID uuid.UUID
	Position  int
}

// PictureDTO — для API.
type PictureDTO struct {
	ID       uuid.UUID `json:"id"`
	URL      string    `json:"url"`
	Position int       `json:"position"`
}

// ReorderRequest — тело PATCH /admin/items/:id/pictures/reorder.
type ReorderRequest struct {
	Order []ReorderEntry `json:"order"`
}

type ReorderEntry struct {
	PictureID uuid.UUID `json:"picture_id"`
	Position  int       `json:"position"`
}
