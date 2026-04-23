package model

import (
	"time"

	"github.com/google/uuid"
)

// OptionType — entity справочника типов.
type OptionType struct {
	ID        uuid.UUID
	Code      string
	Label     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// OptionTypeDTO — для API.
type OptionTypeDTO struct {
	ID        uuid.UUID `json:"id"`
	Code      string    `json:"code"`
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"created_at"`
}

func (o *OptionType) ToDTO() OptionTypeDTO {
	return OptionTypeDTO{
		ID:        o.ID,
		Code:      o.Code,
		Label:     o.Label,
		CreatedAt: o.CreatedAt,
	}
}

// ItemOption — entity опции товара.
type ItemOption struct {
	ID        uuid.UUID
	ItemID    uuid.UUID
	TypeID    uuid.UUID
	Value     string
	Price     int
	Position  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ItemOptionValueDTO — value-часть для группы в детальной карточке.
type ItemOptionValueDTO struct {
	ID       uuid.UUID `json:"id"`
	Value    string    `json:"value"`
	Price    int       `json:"price"`
	Position int       `json:"position"`
}

// OptionGroupDTO — группа значений одного типа для товара.
type OptionGroupDTO struct {
	Type   OptionTypeShortDTO   `json:"type"`
	Values []ItemOptionValueDTO `json:"values"`
}

// OptionTypeShortDTO — короткий DTO типа, вложенный в группу.
type OptionTypeShortDTO struct {
	ID    uuid.UUID `json:"id"`
	Code  string    `json:"code"`
	Label string    `json:"label"`
}

// ItemOptionDTO — полная карточка опции (ответ POST/PATCH /admin/items/:id/options и т.п.).
type ItemOptionDTO struct {
	ID       uuid.UUID          `json:"id"`
	ItemID   uuid.UUID          `json:"item_id"`
	Type     OptionTypeShortDTO `json:"type"`
	Value    string             `json:"value"`
	Price    int                `json:"price"`
	Position int                `json:"position"`
}

// OptionTypeCreateRequest — POST /admin/option-types.
type OptionTypeCreateRequest struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

// OptionTypePatchRequest — PATCH /admin/option-types/:id.
type OptionTypePatchRequest struct {
	Label *string `json:"label,omitempty"`
}

// ItemOptionPatchRequest — PATCH /admin/item-options/:id.
type ItemOptionPatchRequest struct {
	Value    *string `json:"value,omitempty"`
	Price    *int    `json:"price,omitempty"`
	Position *int    `json:"position,omitempty"`
}
