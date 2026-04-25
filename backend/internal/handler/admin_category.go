package handler

import (
	"net/http"

	"yulik3d/internal/model"
	"yulik3d/internal/service"
)

type AdminCategoryHandler struct {
	Deps
	cats *service.AdminCategoryService
}

func NewAdminCategoryHandler(d Deps, cats *service.AdminCategoryService) *AdminCategoryHandler {
	return &AdminCategoryHandler{Deps: d, cats: cats}
}

// CreateCategory godoc
// @Summary      Создать категорию
// @Tags         admin-categories
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Param        payload  body  model.CategoryCreateRequest  true  "Новая категория"
// @Success      201  {object}  model.CategoryDTO
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Router       /admin/categories [post]
func (h *AdminCategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req model.CategoryCreateRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.Err(w, r, err)
		return
	}
	dto, err := h.cats.CreateCategory(r.Context(), req)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	Created(w, dto)
}

// PatchCategory godoc
// @Summary      Обновить категорию
// @Tags         admin-categories
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Param        id       path  string                       true  "UUID"
// @Param        payload  body  model.CategoryPatchRequest   true  "Любое подмножество полей"
// @Success      200  {object}  model.CategoryDTO
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Router       /admin/categories/{id} [patch]
func (h *AdminCategoryHandler) PatchCategory(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUIDPath(r, "id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	var req model.CategoryPatchRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.Err(w, r, err)
		return
	}
	dto, err := h.cats.PatchCategory(r.Context(), id, req)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, dto)
}

// DeleteCategory godoc
// @Summary      Удалить категорию
// @Description  CASCADE: подкатегории и связи с товарами удаляются.
// @Tags         admin-categories
// @Security     CookieAuth
// @Param        id  path  string  true  "UUID"
// @Success      204
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Router       /admin/categories/{id} [delete]
func (h *AdminCategoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUIDPath(r, "id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	if err := h.cats.DeleteCategory(r.Context(), id); err != nil {
		h.Err(w, r, err)
		return
	}
	NoContent(w)
}

// CreateSubcategory godoc
// @Summary      Создать подкатегорию
// @Tags         admin-categories
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Param        id       path  string                          true  "UUID категории"
// @Param        payload  body  model.SubcategoryCreateRequest  true  "Новая подкатегория"
// @Success      201  {object}  model.SubcategoryDTO
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Router       /admin/categories/{id}/subcategories [post]
func (h *AdminCategoryHandler) CreateSubcategory(w http.ResponseWriter, r *http.Request) {
	catID, err := ParseUUIDPath(r, "id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	var req model.SubcategoryCreateRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.Err(w, r, err)
		return
	}
	dto, err := h.cats.CreateSubcategory(r.Context(), catID, req)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	Created(w, dto)
}

// PatchSubcategory godoc
// @Summary      Обновить подкатегорию
// @Tags         admin-categories
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Param        id       path  string                         true  "UUID подкатегории"
// @Param        payload  body  model.SubcategoryPatchRequest  true  "name или category_id (можно перенести в другую категорию)"
// @Success      200  {object}  model.SubcategoryDTO
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Router       /admin/subcategories/{id} [patch]
func (h *AdminCategoryHandler) PatchSubcategory(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUIDPath(r, "id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	var req model.SubcategoryPatchRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.Err(w, r, err)
		return
	}
	dto, err := h.cats.PatchSubcategory(r.Context(), id, req)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, dto)
}

// DeleteSubcategory godoc
// @Summary      Удалить подкатегорию
// @Tags         admin-categories
// @Security     CookieAuth
// @Param        id  path  string  true  "UUID подкатегории"
// @Success      204
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Router       /admin/subcategories/{id} [delete]
func (h *AdminCategoryHandler) DeleteSubcategory(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUIDPath(r, "id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	if err := h.cats.DeleteSubcategory(r.Context(), id); err != nil {
		h.Err(w, r, err)
		return
	}
	NoContent(w)
}
