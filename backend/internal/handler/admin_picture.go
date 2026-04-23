package handler

import (
	"net/http"
	"strconv"

	"yulik3d/internal/model"
	"yulik3d/internal/service"
)

type AdminPictureHandler struct {
	Deps
	pics     *service.AdminPictureService
	maxBytes int64
}

func NewAdminPictureHandler(d Deps, pics *service.AdminPictureService, maxBytes int64) *AdminPictureHandler {
	return &AdminPictureHandler{Deps: d, pics: pics, maxBytes: maxBytes}
}

// Upload godoc
// @Summary      Загрузить картинку к товару
// @Description  multipart/form-data с полем file (png/jpg/webp до 10 МБ). Опционально position.
// @Tags         admin-pictures
// @Security     CookieAuth
// @Accept       mpfd
// @Produce      json
// @Param        id        path      string  true   "UUID товара"
// @Param        file      formData  file    true   "Файл изображения"
// @Param        position  formData  int     false  "Позиция в галерее (1 = титульная)"
// @Success      201  {object}  model.PictureDTO
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Failure      413  {object}  model.ErrorResponse
// @Failure      415  {object}  model.ErrorResponse
// @Router       /admin/items/{id}/pictures [post]
func (h *AdminPictureHandler) Upload(w http.ResponseWriter, r *http.Request) {
	itemID, err := ParseUUIDPath(r, "id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	// Ограничение размера тела на уровне net/http
	r.Body = http.MaxBytesReader(w, r.Body, h.maxBytes+1024) // +1KB под заголовки формы
	if err := r.ParseMultipartForm(h.maxBytes + 1024); err != nil {
		h.Err(w, r, model.NewInvalidInput("Некорректное multipart-тело: "+err.Error()))
		return
	}
	file, fh, err := r.FormFile("file")
	if err != nil {
		h.Err(w, r, model.NewInvalidInput("Поле file обязательно"))
		return
	}
	defer file.Close()

	var pos *int
	if v := r.FormValue("position"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			h.Err(w, r, model.NewInvalidInput("Позиция должна быть положительным целым числом"))
			return
		}
		pos = &n
	}
	dto, err := h.pics.Upload(r.Context(), itemID, file, fh.Size, fh.Filename, fh.Header.Get("Content-Type"), pos)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	Created(w, dto)
}

// Delete godoc
// @Summary      Удалить картинку товара
// @Tags         admin-pictures
// @Security     CookieAuth
// @Param        item_id     path  string  true  "UUID товара"
// @Param        picture_id  path  string  true  "UUID картинки"
// @Success      204
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Router       /admin/items/{item_id}/pictures/{picture_id} [delete]
func (h *AdminPictureHandler) Delete(w http.ResponseWriter, r *http.Request) {
	itemID, err := ParseUUIDPath(r, "item_id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	picID, err := ParseUUIDPath(r, "picture_id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	if err := h.pics.Delete(r.Context(), itemID, picID); err != nil {
		h.Err(w, r, err)
		return
	}
	NoContent(w)
}

// Reorder godoc
// @Summary      Переупорядочить картинки товара
// @Tags         admin-pictures
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Param        id       path  string                 true  "UUID товара"
// @Param        payload  body  model.ReorderRequest   true  "Массив {picture_id, position}"
// @Success      200  {object}  map[string][]model.PictureDTO
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Router       /admin/items/{id}/pictures/reorder [patch]
func (h *AdminPictureHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	itemID, err := ParseUUIDPath(r, "id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	var req model.ReorderRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.Err(w, r, err)
		return
	}
	pics, err := h.pics.Reorder(r.Context(), itemID, req)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, map[string][]model.PictureDTO{"pictures": pics})
}
