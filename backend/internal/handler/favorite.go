package handler

import (
	"net/http"

	"yulik3d/internal/model"
	"yulik3d/internal/service"
)

type FavoriteHandler struct {
	Deps
	favs *service.FavoriteService
}

func NewFavoriteHandler(d Deps, favs *service.FavoriteService) *FavoriteHandler {
	return &FavoriteHandler{Deps: d, favs: favs}
}

// List godoc
// @Summary      Мои избранные товары
// @Tags         favorites
// @Security     CookieAuth
// @Produce      json
// @Param        limit   query  int  false  "default 20, max 100"
// @Param        offset  query  int  false  "default 0"
// @Success      200  {object}  model.ItemCardPage
// @Failure      401  {object}  model.ErrorResponse
// @Router       /favorites [get]
func (h *FavoriteHandler) List(w http.ResponseWriter, r *http.Request) {
	u, ok := h.MustUser(w, r)
	if !ok {
		return
	}
	var p model.Pagination = ParsePagination(r)
	page, err := h.favs.List(r.Context(), u.ID, p)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, page)
}

// Add godoc
// @Summary      Добавить товар в избранное
// @Description  Идемпотентно — повторный вызов не ошибка.
// @Tags         favorites
// @Security     CookieAuth
// @Produce      json
// @Param        item_id  path  string  true  "UUID товара"
// @Success      200  {object}  model.FavoriteAddResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Router       /favorites/{item_id} [post]
func (h *FavoriteHandler) Add(w http.ResponseWriter, r *http.Request) {
	u, ok := h.MustUser(w, r)
	if !ok {
		return
	}
	itemID, err := ParseUUIDPath(r, "item_id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	resp, err := h.favs.Add(r.Context(), u.ID, itemID)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, resp)
}

// Remove godoc
// @Summary      Удалить из избранного
// @Description  Идемпотентно.
// @Tags         favorites
// @Security     CookieAuth
// @Param        item_id  path  string  true  "UUID товара"
// @Success      204
// @Failure      401  {object}  model.ErrorResponse
// @Router       /favorites/{item_id} [delete]
func (h *FavoriteHandler) Remove(w http.ResponseWriter, r *http.Request) {
	u, ok := h.MustUser(w, r)
	if !ok {
		return
	}
	itemID, err := ParseUUIDPath(r, "item_id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	if err := h.favs.Remove(r.Context(), u.ID, itemID); err != nil {
		h.Err(w, r, err)
		return
	}
	NoContent(w)
}
