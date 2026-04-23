package handler

import (
	"net/http"

	"yulik3d/internal/model"
	"yulik3d/internal/service"
)

type AdminOptionHandler struct {
	Deps
	opts *service.AdminOptionService
}

func NewAdminOptionHandler(d Deps, opts *service.AdminOptionService) *AdminOptionHandler {
	return &AdminOptionHandler{Deps: d, opts: opts}
}

// ListTypes godoc
// @Summary      Список типов опций
// @Tags         admin-options
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  map[string][]model.OptionTypeDTO
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Router       /admin/option-types [get]
func (h *AdminOptionHandler) ListTypes(w http.ResponseWriter, r *http.Request) {
	list, err := h.opts.ListTypes(r.Context())
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, map[string][]model.OptionTypeDTO{"option_types": list})
}

// CreateType godoc
// @Summary      Создать тип опции
// @Tags         admin-options
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Param        payload  body  model.OptionTypeCreateRequest  true  "Новый тип опции (code + label)"
// @Success      201  {object}  model.OptionTypeDTO
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      409  {object}  model.ErrorResponse
// @Router       /admin/option-types [post]
func (h *AdminOptionHandler) CreateType(w http.ResponseWriter, r *http.Request) {
	var req model.OptionTypeCreateRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.Err(w, r, err)
		return
	}
	dto, err := h.opts.CreateType(r.Context(), req)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	Created(w, dto)
}

// PatchType godoc
// @Summary      Обновить тип опции (label)
// @Tags         admin-options
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Param        id       path  string                         true  "UUID"
// @Param        payload  body  model.OptionTypePatchRequest   true  "Новый label"
// @Success      200  {object}  model.OptionTypeDTO
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Router       /admin/option-types/{id} [patch]
func (h *AdminOptionHandler) PatchType(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUIDPath(r, "id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	var req model.OptionTypePatchRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.Err(w, r, err)
		return
	}
	dto, err := h.opts.PatchType(r.Context(), id, req)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, dto)
}

// DeleteType godoc
// @Summary      Удалить тип опции
// @Description  409 Conflict, если тип используется в item_option.
// @Tags         admin-options
// @Security     CookieAuth
// @Param        id  path  string  true  "UUID"
// @Success      204
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Failure      409  {object}  model.ErrorResponse
// @Router       /admin/option-types/{id} [delete]
func (h *AdminOptionHandler) DeleteType(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUIDPath(r, "id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	if err := h.opts.DeleteType(r.Context(), id); err != nil {
		h.Err(w, r, err)
		return
	}
	NoContent(w)
}

// CreateItemOption godoc
// @Summary      Добавить опцию к товару
// @Tags         admin-options
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Param        id       path  string                          true  "UUID товара"
// @Param        payload  body  model.ItemOptionCreateRequest   true  "type_id, value, price, position"
// @Success      201  {object}  model.ItemOptionDTO
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Failure      409  {object}  model.ErrorResponse
// @Router       /admin/items/{id}/options [post]
func (h *AdminOptionHandler) CreateItemOption(w http.ResponseWriter, r *http.Request) {
	itemID, err := ParseUUIDPath(r, "id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	var req model.ItemOptionCreateRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.Err(w, r, err)
		return
	}
	dto, err := h.opts.CreateItemOption(r.Context(), itemID, req)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	Created(w, dto)
}

// PatchItemOption godoc
// @Summary      Обновить опцию товара
// @Tags         admin-options
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Param        id       path  string                         true  "UUID item_option"
// @Param        payload  body  model.ItemOptionPatchRequest   true  "Любое подмножество value/price/position"
// @Success      200  {object}  model.ItemOptionDTO
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Failure      409  {object}  model.ErrorResponse
// @Router       /admin/item-options/{id} [patch]
func (h *AdminOptionHandler) PatchItemOption(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUIDPath(r, "id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	var req model.ItemOptionPatchRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.Err(w, r, err)
		return
	}
	dto, err := h.opts.PatchItemOption(r.Context(), id, req)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, dto)
}

// DeleteItemOption godoc
// @Summary      Удалить опцию товара
// @Tags         admin-options
// @Security     CookieAuth
// @Param        id  path  string  true  "UUID item_option"
// @Success      204
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Router       /admin/item-options/{id} [delete]
func (h *AdminOptionHandler) DeleteItemOption(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUIDPath(r, "id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	if err := h.opts.DeleteItemOption(r.Context(), id); err != nil {
		h.Err(w, r, err)
		return
	}
	NoContent(w)
}
