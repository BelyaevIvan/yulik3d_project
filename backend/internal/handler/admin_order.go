package handler

import (
	"net/http"

	"github.com/google/uuid"

	"yulik3d/internal/model"
	"yulik3d/internal/service"
)

type AdminOrderHandler struct {
	Deps
	orders *service.OrderService
}

func NewAdminOrderHandler(d Deps, orders *service.OrderService) *AdminOrderHandler {
	return &AdminOrderHandler{Deps: d, orders: orders}
}

// List godoc
// @Summary      Админский список заказов (очередь)
// @Tags         admin-orders
// @Security     CookieAuth
// @Produce      json
// @Param        status   query  string  false  "Фильтр по статусу"  Enums(created,confirmed,manufacturing,delivering,completed,cancelled)
// @Param        user_id  query  string  false  "UUID пользователя"
// @Param        q        query  string  false  "Поиск по имени/телефону"
// @Param        limit    query  int     false  "default 20, max 100"
// @Param        offset   query  int     false  "default 0"
// @Success      200  {object}  model.OrderAdminListPage
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Router       /admin/orders [get]
func (h *AdminOrderHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := model.OrderAdminListFilter{
		Query:      q.Get("q"),
		Pagination: ParsePagination(r),
	}
	if v := q.Get("status"); v != "" {
		s := model.OrderStatus(v)
		f.Status = &s
	}
	if v := q.Get("user_id"); v != "" {
		uid, err := uuid.Parse(v)
		if err != nil {
			h.Err(w, r, model.NewInvalidInput("user_id: некорректный UUID"))
			return
		}
		f.UserID = &uid
	}
	page, err := h.orders.AdminList(r.Context(), f)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, page)
}

// Get godoc
// @Summary      Админские детали заказа
// @Tags         admin-orders
// @Security     CookieAuth
// @Produce      json
// @Param        id  path  string  true  "UUID"
// @Success      200  {object}  model.OrderAdminDetailDTO
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Router       /admin/orders/{id} [get]
func (h *AdminOrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUIDPath(r, "id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	dto, err := h.orders.AdminGet(r.Context(), id)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, dto)
}

// PatchStatus godoc
// @Summary      Смена статуса заказа
// @Description  Допустимы только переходы вперёд по цепочке или в cancelled.
// @Tags         admin-orders
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Param        id       path  string                          true  "UUID"
// @Param        payload  body  model.OrderStatusPatchRequest   true  "Новый статус"
// @Success      200  {object}  model.OrderAdminDetailDTO
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Failure      409  {object}  model.ErrorResponse
// @Router       /admin/orders/{id}/status [patch]
func (h *AdminOrderHandler) PatchStatus(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUIDPath(r, "id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	var req model.OrderStatusPatchRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.Err(w, r, err)
		return
	}
	dto, err := h.orders.AdminPatchStatus(r.Context(), id, req.Status)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, dto)
}

// Patch godoc
// @Summary      Обновить внутренние поля заказа (admin_note)
// @Tags         admin-orders
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Param        id       path  string                        true  "UUID"
// @Param        payload  body  model.OrderAdminPatchRequest  true  "Внутренние поля заказа (admin_note)"
// @Success      200  {object}  model.OrderAdminDetailDTO
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Router       /admin/orders/{id} [patch]
func (h *AdminOrderHandler) Patch(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUIDPath(r, "id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	var req model.OrderAdminPatchRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.Err(w, r, err)
		return
	}
	dto, err := h.orders.AdminPatchNote(r.Context(), id, req.AdminNote)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, dto)
}
