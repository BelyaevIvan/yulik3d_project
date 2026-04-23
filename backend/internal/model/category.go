package model

import (
	"time"

	"github.com/google/uuid"
)

// CategoryType — 'figure' | 'other'.
type CategoryType string

const (
	CategoryTypeFigure CategoryType = "figure"
	CategoryTypeOther  CategoryType = "other"
)

// Category — entity.
type Category struct {
	ID        uuid.UUID
	Name      string
	Type      CategoryType
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CategoryDTO — для API.
type CategoryDTO struct {
	ID            uuid.UUID            `json:"id"`
	Name          string               `json:"name"`
	Type          CategoryType         `json:"type"`
	Subcategories []SubcategoryShortDTO `json:"subcategories,omitempty"`
}

// CategoryShortDTO — короткий DTO (вложенный).
type CategoryShortDTO struct {
	ID   uuid.UUID    `json:"id"`
	Name string       `json:"name"`
	Type CategoryType `json:"type"`
}

// CategoryCreateRequest — POST /admin/categories.
type CategoryCreateRequest struct {
	Name string       `json:"name"`
	Type CategoryType `json:"type"`
}

// CategoryPatchRequest — PATCH /admin/categories/:id.
type CategoryPatchRequest struct {
	Name *string       `json:"name,omitempty"`
	Type *CategoryType `json:"type,omitempty"`
}

// Subcategory — entity.
type Subcategory struct {
	ID         uuid.UUID
	Name       string
	CategoryID uuid.UUID
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// SubcategoryDTO — для API.
type SubcategoryDTO struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	CategoryID uuid.UUID `json:"category_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// SubcategoryShortDTO — короткий DTO (вложенный).
type SubcategoryShortDTO struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// SubcategoryWithCategoryDTO — для карточки товара (подкатегория + её категория).
type SubcategoryWithCategoryDTO struct {
	ID       uuid.UUID        `json:"id"`
	Name     string           `json:"name"`
	Category CategoryShortDTO `json:"category"`
}

// SubcategoryCreateRequest — POST /admin/categories/:id/subcategories.
type SubcategoryCreateRequest struct {
	Name string `json:"name"`
}

// SubcategoryPatchRequest — PATCH /admin/subcategories/:id.
type SubcategoryPatchRequest struct {
	Name       *string    `json:"name,omitempty"`
	CategoryID *uuid.UUID `json:"category_id,omitempty"`
}
