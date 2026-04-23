package model

import (
	"time"

	"github.com/google/uuid"
)

// Item — entity.
type Item struct {
	ID               uuid.UUID
	Name             string
	DescriptionInfo  string
	DescriptionOther string
	Price            int
	Sale             int
	Articul          string
	Hidden           bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// FinalPrice считает итоговую цену с учётом скидки.
func (i *Item) FinalPrice() int {
	if i.Sale <= 0 {
		return i.Price
	}
	if i.Sale >= 100 {
		return 0
	}
	// round(price * (100 - sale) / 100) — банковское округление не нужно, обычное
	p := i.Price * (100 - i.Sale)
	r := p / 100
	if p%100 >= 50 {
		r++
	}
	return r
}

// ItemCardDTO — короткая карточка для списка в каталоге/избранном.
type ItemCardDTO struct {
	ID                uuid.UUID          `json:"id"`
	Name              string             `json:"name"`
	Articul           string             `json:"articul"`
	Price             int                `json:"price"`
	Sale              int                `json:"sale"`
	FinalPrice        int                `json:"final_price"`
	Hidden            bool               `json:"hidden,omitempty"`
	PrimaryPictureURL *string            `json:"primary_picture_url"`
	Category          *CategoryShortDTO  `json:"category"`
	Subcategories     []SubcategoryShortDTO `json:"subcategories"`
}

// ItemDetailDTO — полная карточка товара.
type ItemDetailDTO struct {
	ID                uuid.UUID             `json:"id"`
	Name              string                `json:"name"`
	Articul           string                `json:"articul"`
	DescriptionInfo   string                `json:"description_info"`
	DescriptionOther  string                `json:"description_other"`
	Price             int                   `json:"price"`
	Sale              int                   `json:"sale"`
	FinalPrice        int                   `json:"final_price"`
	Hidden            bool                  `json:"hidden"`
	Pictures          []PictureDTO          `json:"pictures"`
	Options           []OptionGroupDTO      `json:"options"`
	Subcategories     []SubcategoryWithCategoryDTO `json:"subcategories"`
	CreatedAt         time.Time             `json:"created_at"`
	UpdatedAt         time.Time             `json:"updated_at"`
}

// CatalogFilter — параметры GET /items.
type CatalogFilter struct {
	CategoryType  *string // "figure" | "other"
	CategoryID    *uuid.UUID
	SubcategoryID *uuid.UUID
	Query         string
	HasSale       *bool
	IncludeHidden bool // для админки
	HiddenOnly    *bool // для админки (может быть фильтр true/false/any)
	Sort          string // created_desc | created_asc | price_asc | price_desc | name_asc | name_desc
	Pagination    Pagination
}

// ItemCreateRequest — тело POST /admin/items.
type ItemCreateRequest struct {
	Name             string                    `json:"name"`
	DescriptionInfo  string                    `json:"description_info"`
	DescriptionOther string                    `json:"description_other"`
	Price            int                       `json:"price"`
	Sale             int                       `json:"sale"`
	Hidden           bool                      `json:"hidden"`
	SubcategoryIDs   []uuid.UUID               `json:"subcategory_ids"`
	Options          []ItemOptionCreateRequest `json:"options"`
}

// ItemOptionCreateRequest — вложенная структура в ItemCreateRequest и для POST /admin/items/:id/options.
type ItemOptionCreateRequest struct {
	TypeID   uuid.UUID `json:"type_id"`
	Value    string    `json:"value"`
	Price    int       `json:"price"`
	Position int       `json:"position"`
}

// ItemUpdateRequest — тело PUT /admin/items/:id. Все поля обязательны.
type ItemUpdateRequest struct {
	Name             string                    `json:"name"`
	DescriptionInfo  string                    `json:"description_info"`
	DescriptionOther string                    `json:"description_other"`
	Price            int                       `json:"price"`
	Sale             int                       `json:"sale"`
	Hidden           bool                      `json:"hidden"`
	SubcategoryIDs   []uuid.UUID               `json:"subcategory_ids"`
	Options          []ItemOptionCreateRequest `json:"options"`
}

// ItemPatchRequest — тело PATCH /admin/items/:id. Любое подмножество.
type ItemPatchRequest struct {
	Name             *string `json:"name,omitempty"`
	DescriptionInfo  *string `json:"description_info,omitempty"`
	DescriptionOther *string `json:"description_other,omitempty"`
	Price            *int    `json:"price,omitempty"`
	Sale             *int    `json:"sale,omitempty"`
	Hidden           *bool   `json:"hidden,omitempty"`
}
