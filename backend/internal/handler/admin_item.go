package handler

import (
	"net/http"

	"yulik3d/internal/model"
	"yulik3d/internal/service"
)

type AdminItemHandler struct {
	Deps
	admin   *service.AdminItemService
	catalog *service.CatalogService
}

func NewAdminItemHandler(d Deps, admin *service.AdminItemService, catalog *service.CatalogService) *AdminItemHandler {
	return &AdminItemHandler{Deps: d, admin: admin, catalog: catalog}
}

// List godoc
// @Summary      Админский список товаров
// @Description  Возвращает все товары, в т.ч. скрытые. Поддерживает все фильтры каталога + параметр hidden.
// @Tags         admin-items
// @Security     CookieAuth
// @Produce      json
// @Param        hidden            query  string   false  "any | true | false"  Enums(any, true, false)
// @Param        category_type     query  string   false  "figure | other"  Enums(figure, other)
// @Param        category_id       query  string   false  "UUID категории"
// @Param        subcategory_id    query  string   false  "UUID подкатегории"
// @Param        q                 query  string   false  "Поиск по name (ILIKE)"
// @Param        has_sale          query  bool     false  "Только со скидкой"
// @Param        sort              query  string   false  "created_desc | created_asc | price_asc | price_desc | name_asc | name_desc"
// @Param        limit             query  int      false  "default 20, max 100"
// @Param        offset            query  int      false  "default 0"
// @Success      200  {object}  model.ItemCardPage
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Router       /admin/items [get]
func (h *AdminItemHandler) List(w http.ResponseWriter, r *http.Request) {
	f, err := parseCatalogFilter(r, true)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	page, err := h.catalog.ListItems(r.Context(), f, true)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, page)
}

// Create godoc
// @Summary      Создать товар
// @Description  Атомарно: item + subcategory-связи + опции. Картинки загружаются отдельно.
// @Tags         admin-items
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Param        payload  body  model.ItemCreateRequest  true  "Данные товара"
// @Success      201  {object}  model.ItemDetailDTO
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Router       /admin/items [post]
func (h *AdminItemHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.ItemCreateRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.Err(w, r, err)
		return
	}
	dto, err := h.admin.Create(r.Context(), req)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/items/"+dto.ID.String())
	Created(w, dto)
}

// Get godoc
// @Summary      Админская карточка товара
// @Tags         admin-items
// @Security     CookieAuth
// @Produce      json
// @Param        id  path  string  true  "UUID товара"
// @Success      200  {object}  model.ItemDetailDTO
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Router       /admin/items/{id} [get]
func (h *AdminItemHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUIDPath(r, "id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	dto, err := h.admin.Get(r.Context(), id)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, dto)
}

// Update godoc
// @Summary      Полная замена товара (PUT)
// @Description  Опции и связи подкатегорий перезаписываются целиком.
// @Tags         admin-items
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Param        id       path  string                  true  "UUID товара"
// @Param        payload  body  model.ItemUpdateRequest true  "Полные данные товара (опции и subcategory_ids перезаписываются)"
// @Success      200  {object}  model.ItemDetailDTO
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Router       /admin/items/{id} [put]
func (h *AdminItemHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUIDPath(r, "id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	var req model.ItemUpdateRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.Err(w, r, err)
		return
	}
	dto, err := h.admin.Update(r.Context(), id, req)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, dto)
}

// Patch godoc
// @Summary      Частичное обновление товара
// @Description  Основной use-case — быстрый toggle hidden.
// @Tags         admin-items
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Param        id       path  string                 true  "UUID"
// @Param        payload  body  model.ItemPatchRequest true  "Любое подмножество полей"
// @Success      200  {object}  model.ItemDetailDTO
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Router       /admin/items/{id} [patch]
func (h *AdminItemHandler) Patch(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUIDPath(r, "id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	var req model.ItemPatchRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.Err(w, r, err)
		return
	}
	dto, err := h.admin.Patch(r.Context(), id, req)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, dto)
}
