package model

// Pagination — параметры листинга.
type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// Clamp приводит значения к корректным границам.
func (p *Pagination) Clamp(defaultLimit, maxLimit int) {
	if p.Limit <= 0 {
		p.Limit = defaultLimit
	}
	if p.Limit > maxLimit {
		p.Limit = maxLimit
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
}

// ListPage — обёртка для постраничных ответов (для DTO).
type ListPage[T any] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// ---------------------------------------------------------------------------
// Конкретные обёртки для Swagger.
//
// swag (на 2026-04) криво парсит дженерик-синтаксис `ListPage[T]` в @Success-
// аннотациях — иногда генерит, иногда падает. Поэтому для swagger ссылаемся на
// конкретные структуры с тем же JSON-формой. Хэндлеры в коде продолжают
// возвращать ListPage[T] — JSON-вывод идентичен.
// ---------------------------------------------------------------------------

// ItemCardPage — Swagger-обёртка для ListPage[ItemCardDTO].
type ItemCardPage struct {
	Items  []ItemCardDTO `json:"items"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

// OrderListPage — Swagger-обёртка для ListPage[OrderListItemDTO].
type OrderListPage struct {
	Items  []OrderListItemDTO `json:"items"`
	Total  int                `json:"total"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
}

// OrderAdminListPage — Swagger-обёртка для ListPage[OrderAdminListItemDTO].
type OrderAdminListPage struct {
	Items  []OrderAdminListItemDTO `json:"items"`
	Total  int                     `json:"total"`
	Limit  int                     `json:"limit"`
	Offset int                     `json:"offset"`
}
