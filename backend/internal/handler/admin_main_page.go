package handler

import (
	"net/http"

	"github.com/google/uuid"

	"yulik3d/internal/model"
	"yulik3d/internal/service"
)

// AdminMainPageHandler — управление закреплениями на главной странице.
type AdminMainPageHandler struct {
	Deps
	svc *service.AdminMainPageService
}

func NewAdminMainPageHandler(d Deps, svc *service.AdminMainPageService) *AdminMainPageHandler {
	return &AdminMainPageHandler{Deps: d, svc: svc}
}

// PinRequestDTO — тело POST /admin/main.
type PinRequestDTO struct {
	ItemID   string             `json:"item_id" example:"018f7d3e-..."`
	Type     model.CategoryType `json:"type" example:"figure"`
	Position *int               `json:"position,omitempty" example:"1"` // 1..5; если nil — следующая свободная
}

// ReorderRequestDTO — тело PATCH /admin/main/{type}/reorder.
type ReorderRequestDTO struct {
	Order []service.ReorderEntry `json:"order"`
}

// List godoc
// @Summary      Закреплённые на главной товары
// @Description  Возвращает закреплённые товары обоих типов с детальной инфой.
// @Tags         admin-main
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  service.MainPageDTO
// @Router       /admin/main [get]
func (h *AdminMainPageHandler) List(w http.ResponseWriter, r *http.Request) {
	dto, err := h.svc.List(r.Context())
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, dto)
}

// Pin godoc
// @Summary      Закрепить товар на главной
// @Description  Закрепляет товар в указанном разделе. Position (1..5) опциональна — без неё ставится следующая свободная.
// @Tags         admin-main
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Param        payload  body  handler.PinRequestDTO  true  "item_id, type, опционально position"
// @Success      200  {object}  model.OKResponse
// @Failure      400  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Failure      409  {object}  model.ErrorResponse
// @Router       /admin/main [post]
func (h *AdminMainPageHandler) Pin(w http.ResponseWriter, r *http.Request) {
	var req PinRequestDTO
	if err := DecodeJSON(r, &req); err != nil {
		h.Err(w, r, err)
		return
	}
	itemID, err := uuid.Parse(req.ItemID)
	if err != nil {
		h.Err(w, r, model.NewInvalidInput("Некорректный item_id"))
		return
	}
	if req.Type != model.CategoryTypeFigure && req.Type != model.CategoryTypeOther {
		h.Err(w, r, model.NewInvalidInput("type должен быть figure или other"))
		return
	}
	if err := h.svc.Pin(r.Context(), itemID, req.Type, req.Position); err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, model.OKResponse{OK: true})
}

// Unpin godoc
// @Summary      Открепить товар от главной
// @Tags         admin-main
// @Security     CookieAuth
// @Param        type     path  string  true  "Тип: figure или other"
// @Param        item_id  path  string  true  "UUID товара"
// @Success      200  {object}  model.OKResponse
// @Failure      404  {object}  model.ErrorResponse
// @Router       /admin/main/{type}/{item_id} [delete]
func (h *AdminMainPageHandler) Unpin(w http.ResponseWriter, r *http.Request) {
	t := model.CategoryType(r.PathValue("type"))
	if t != model.CategoryTypeFigure && t != model.CategoryTypeOther {
		h.Err(w, r, model.NewInvalidInput("type должен быть figure или other"))
		return
	}
	itemID, err := uuid.Parse(r.PathValue("item_id"))
	if err != nil {
		h.Err(w, r, model.NewInvalidInput("Некорректный item_id"))
		return
	}
	if err := h.svc.Unpin(r.Context(), itemID, t); err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, model.OKResponse{OK: true})
}

// Reorder godoc
// @Summary      Сменить порядок закреплённых товаров в типе (drag-and-drop)
// @Tags         admin-main
// @Security     CookieAuth
// @Accept       json
// @Param        type     path  string  true  "Тип: figure или other"
// @Param        payload  body  handler.ReorderRequestDTO  true  "Полный список закреплений в новом порядке"
// @Success      200  {object}  model.OKResponse
// @Failure      400  {object}  model.ErrorResponse
// @Router       /admin/main/{type}/reorder [patch]
func (h *AdminMainPageHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	t := model.CategoryType(r.PathValue("type"))
	if t != model.CategoryTypeFigure && t != model.CategoryTypeOther {
		h.Err(w, r, model.NewInvalidInput("type должен быть figure или other"))
		return
	}
	var req ReorderRequestDTO
	if err := DecodeJSON(r, &req); err != nil {
		h.Err(w, r, err)
		return
	}
	if err := h.svc.Reorder(r.Context(), t, req.Order); err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, model.OKResponse{OK: true})
}
