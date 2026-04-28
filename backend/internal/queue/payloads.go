// Package queue — обёртки над hibiken/asynq для асинхронной отправки писем.
//
// Декомпозиция:
//   - payloads.go — типы задач (ключи и payload-ы)
//   - asynq.go    — клиент (enqueue) и сервер (воркер)
//   - handlers.go — обработчики задач (рендер + отправка через mail.Mailer)
package queue

const (
	TaskEmailOrderCreatedAdmin   = "email:order_created_admin"
	TaskEmailOrderStatusChanged  = "email:order_status_changed"
	TaskEmailPasswordReset       = "email:password_reset"
)

// EmailOrderCreatedAdminPayload — все данные для письма админу о новом заказе.
// Передаём заранее подготовленные значения (с учётом форматирования), чтобы
// воркер не лез в БД и не зависел от состояния (товар может быть удалён к
// моменту отправки — нам это неважно, у нас snapshot в самом заказе).
type EmailOrderCreatedAdminPayload struct {
	To              string             `json:"to"`
	OrderID         string             `json:"order_id"`         // полный UUID
	OrderIDShort    string             `json:"order_id_short"`   // первые 8 символов
	CreatedAt       string             `json:"created_at"`       // отформатированная дата
	Total           int                `json:"total"`
	UserEmail       string             `json:"user_email"`
	ContactFullName string             `json:"contact_full_name"`
	ContactPhone    string             `json:"contact_phone"`
	CustomerComment string             `json:"customer_comment"`
	Items           []OrderLinePayload `json:"items"`
	AdminURL        string             `json:"admin_url"`
}

type OrderLinePayload struct {
	Name           string                   `json:"name"`
	Quantity       int                      `json:"quantity"`
	UnitTotalPrice int                      `json:"unit_total_price"`
	Subtotal       int                      `json:"subtotal"`
	Options        []OrderLineOptionPayload `json:"options"`
}

type OrderLineOptionPayload struct {
	TypeLabel string `json:"type_label"`
	Value     string `json:"value"`
	Price     int    `json:"price"`
}

// EmailOrderStatusChangedPayload — для письма пользователю о смене статуса.
type EmailOrderStatusChangedPayload struct {
	To           string   `json:"to"`
	UserName     string   `json:"user_name"`
	OrderIDShort string   `json:"order_id_short"`
	StatusKey    string   `json:"status_key"`
	Total        int      `json:"total"`
	ItemsSummary []string `json:"items_summary"` // только для статуса "created"
	AdminNote    string   `json:"admin_note"`    // только для "cancelled"
	OrderURL     string   `json:"order_url"`
}

// EmailPasswordResetPayload — для письма с ссылкой ресета.
type EmailPasswordResetPayload struct {
	To        string `json:"to"`
	UserName  string `json:"user_name"`
	ResetLink string `json:"reset_link"`
}
