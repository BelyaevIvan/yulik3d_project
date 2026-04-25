package model

import (
	"time"

	"github.com/google/uuid"
)

// Favorite — entity.
type Favorite struct {
	UserID    uuid.UUID
	ItemID    uuid.UUID
	CreatedAt time.Time
}

// FavoriteAddResponse — ответ POST /favorites/:item_id.
type FavoriteAddResponse struct {
	ItemID    uuid.UUID `json:"item_id"`
	CreatedAt time.Time `json:"created_at"`
}
