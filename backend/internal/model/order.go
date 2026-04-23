package model

import (
	"time"

	"github.com/google/uuid"
)

// OrderStatus — статус заказа.
type OrderStatus string

const (
	OrderStatusCreated       OrderStatus = "created"
	OrderStatusConfirmed     OrderStatus = "confirmed"
	OrderStatusManufacturing OrderStatus = "manufacturing"
	OrderStatusDelivering    OrderStatus = "delivering"
	OrderStatusCompleted     OrderStatus = "completed"
	OrderStatusCancelled     OrderStatus = "cancelled"
)

// ValidStatusTransitions — матрица разрешённых переходов.
var ValidStatusTransitions = map[OrderStatus][]OrderStatus{
	OrderStatusCreated:       {OrderStatusConfirmed, OrderStatusCancelled},
	OrderStatusConfirmed:     {OrderStatusManufacturing, OrderStatusCancelled},
	OrderStatusManufacturing: {OrderStatusDelivering, OrderStatusCancelled},
	OrderStatusDelivering:    {OrderStatusCompleted, OrderStatusCancelled},
	OrderStatusCompleted:     {},
	OrderStatusCancelled:     {},
}

// CanTransition проверяет допустимость перехода.
func CanTransition(from, to OrderStatus) bool {
	for _, s := range ValidStatusTransitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

// Order — entity.
type Order struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Status          OrderStatus
	TotalPrice      int
	CustomerComment *string
	AdminNote       *string
	ContactPhone    string
	ContactFullName string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// OrderItem — entity.
type OrderItem struct {
	ID                   uuid.UUID
	OrderID              uuid.UUID
	ItemID               uuid.UUID
	Quantity             int
	UnitBasePrice        int
	UnitTotalPrice       int
	ItemNameSnapshot     string
	ItemArticulSnapshot  string
}

// OrderItemOption — entity.
type OrderItemOption struct {
	ID                  uuid.UUID
	OrderItemID         uuid.UUID
	TypeCodeSnapshot    string
	TypeLabelSnapshot   string
	ValueSnapshot       string
	PriceSnapshot       int
}

// ---------- Запросы ----------

// OrderCreateRequest — POST /orders.
type OrderCreateRequest struct {
	Items            []OrderItemCreate `json:"items"`
	CustomerComment  *string           `json:"customer_comment,omitempty"`
	ContactPhone     string            `json:"contact_phone"`
	ContactFullName  string            `json:"contact_full_name"`
}

type OrderItemCreate struct {
	ItemID    uuid.UUID   `json:"item_id"`
	Quantity  int         `json:"quantity"`
	OptionIDs []uuid.UUID `json:"option_ids"`
}

// OrderStatusPatchRequest — PATCH /admin/orders/:id/status.
type OrderStatusPatchRequest struct {
	Status OrderStatus `json:"status"`
}

// OrderAdminPatchRequest — PATCH /admin/orders/:id.
type OrderAdminPatchRequest struct {
	AdminNote *string `json:"admin_note,omitempty"`
}

// ---------- DTO ----------

// OrderListItemDTO — краткий вид для списка заказов пользователя.
type OrderListItemDTO struct {
	ID         uuid.UUID   `json:"id"`
	Status     OrderStatus `json:"status"`
	TotalPrice int         `json:"total_price"`
	ItemsCount int         `json:"items_count"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

// OrderDetailDTO — детали заказа для пользователя (без admin_note).
type OrderDetailDTO struct {
	ID              uuid.UUID          `json:"id"`
	Status          OrderStatus        `json:"status"`
	TotalPrice      int                `json:"total_price"`
	CustomerComment *string            `json:"customer_comment"`
	ContactPhone    string             `json:"contact_phone"`
	ContactFullName string             `json:"contact_full_name"`
	Items           []OrderItemDTO     `json:"items"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

// OrderItemDTO — одна позиция заказа с опциями.
type OrderItemDTO struct {
	ID                    uuid.UUID                 `json:"id"`
	ItemID                uuid.UUID                 `json:"item_id"`
	ItemNameSnapshot      string                    `json:"item_name_snapshot"`
	ItemArticulSnapshot   string                    `json:"item_articul_snapshot"`
	Quantity              int                       `json:"quantity"`
	UnitBasePrice         int                       `json:"unit_base_price"`
	UnitTotalPrice        int                       `json:"unit_total_price"`
	Options               []OrderItemOptionDTO      `json:"options"`
}

// OrderItemOptionDTO — выбранная опция позиции (снапшот).
type OrderItemOptionDTO struct {
	TypeCodeSnapshot  string `json:"type_code_snapshot"`
	TypeLabelSnapshot string `json:"type_label_snapshot"`
	ValueSnapshot     string `json:"value_snapshot"`
	PriceSnapshot     int    `json:"price_snapshot"`
}

// OrderAdminListItemDTO — краткий вид для админской очереди.
type OrderAdminListItemDTO struct {
	ID              uuid.UUID     `json:"id"`
	User            UserShortDTO  `json:"user"`
	Status          OrderStatus   `json:"status"`
	TotalPrice      int           `json:"total_price"`
	ItemsCount      int           `json:"items_count"`
	ContactPhone    string        `json:"contact_phone"`
	ContactFullName string        `json:"contact_full_name"`
	CustomerComment *string       `json:"customer_comment"`
	AdminNote       *string       `json:"admin_note"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

// OrderAdminDetailDTO — детали заказа для админа (с admin_note и полными данными user).
type OrderAdminDetailDTO struct {
	ID              uuid.UUID       `json:"id"`
	User            UserFullShortDTO `json:"user"`
	Status          OrderStatus     `json:"status"`
	TotalPrice      int             `json:"total_price"`
	CustomerComment *string         `json:"customer_comment"`
	AdminNote       *string         `json:"admin_note"`
	ContactPhone    string          `json:"contact_phone"`
	ContactFullName string          `json:"contact_full_name"`
	Items           []OrderItemDTO  `json:"items"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// UserShortDTO — минимум о пользователе (для списка заказов).
type UserShortDTO struct {
	ID       uuid.UUID `json:"id"`
	Email    string    `json:"email"`
	FullName string    `json:"full_name"`
}

// UserFullShortDTO — для деталей админского заказа (чуть больше).
type UserFullShortDTO struct {
	ID       uuid.UUID `json:"id"`
	Email    string    `json:"email"`
	FullName string    `json:"full_name"`
	Phone    *string   `json:"phone,omitempty"`
}

// OrderListFilter — фильтр GET /orders (user history).
type OrderListFilter struct {
	Status     *OrderStatus
	Pagination Pagination
}

// OrderAdminListFilter — фильтр GET /admin/orders.
type OrderAdminListFilter struct {
	Status     *OrderStatus
	UserID     *uuid.UUID
	Query      string
	Pagination Pagination
}
