package handler

import (
	"net/http"

	"yulik3d/internal/model"
	"yulik3d/internal/service"
)

type OrderHandler struct {
	Deps
	orders *service.OrderService
}

func NewOrderHandler(d Deps, orders *service.OrderService) *OrderHandler {
	return &OrderHandler{Deps: d, orders: orders}
}

// Create godoc
// @Summary      Оформить заказ
// @Description  Бэк пересчитывает все цены из БД (цены из запроса игнорируются). Только для авторизованных.
// @Tags         orders
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Param        payload  body  model.OrderCreateRequest  true  "Позиции и контакты"
// @Success      201  {object}  model.OrderDetailDTO
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      409  {object}  model.ErrorResponse
// @Router       /orders [post]
func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	u, ok := h.MustUser(w, r)
	if !ok {
		return
	}
	var req model.OrderCreateRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.Err(w, r, err)
		return
	}
	dto, err := h.orders.Create(r.Context(), u.ID, req)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	Created(w, dto)
}

// ListMy godoc
// @Summary      История моих заказов
// @Tags         orders
// @Security     CookieAuth
// @Produce      json
// @Param        status  query  string  false  "Фильтр по статусу"  Enums(created,confirmed,manufacturing,delivering,completed,cancelled)
// @Param        limit   query  int     false  "default 20, max 100"
// @Param        offset  query  int     false  "default 0"
// @Success      200  {object}  model.OrderListPage
// @Failure      401  {object}  model.ErrorResponse
// @Router       /orders [get]
func (h *OrderHandler) ListMy(w http.ResponseWriter, r *http.Request) {
	u, ok := h.MustUser(w, r)
	if !ok {
		return
	}
	var status *model.OrderStatus
	if v := r.URL.Query().Get("status"); v != "" {
		s := model.OrderStatus(v)
		status = &s
	}
	page, err := h.orders.ListMy(r.Context(), u.ID, status, ParsePagination(r))
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, page)
}

// GetMy godoc
// @Summary      Детали моего заказа
// @Description  Чужой заказ возвращает 404 (не 403), чтобы не светить факт существования.
// @Tags         orders
// @Security     CookieAuth
// @Produce      json
// @Param        id  path  string  true  "UUID заказа"
// @Success      200  {object}  model.OrderDetailDTO
// @Failure      401  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Router       /orders/{id} [get]
func (h *OrderHandler) GetMy(w http.ResponseWriter, r *http.Request) {
	u, ok := h.MustUser(w, r)
	if !ok {
		return
	}
	id, err := ParseUUIDPath(r, "id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	dto, err := h.orders.GetMy(r.Context(), u.ID, id)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, dto)
}
