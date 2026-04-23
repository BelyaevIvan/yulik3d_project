package handler

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"yulik3d/internal/model"
	"yulik3d/internal/service"
)

type CatalogHandler struct {
	Deps
	catalog *service.CatalogService
}

func NewCatalogHandler(d Deps, catalog *service.CatalogService) *CatalogHandler {
	return &CatalogHandler{Deps: d, catalog: catalog}
}

// ListItems godoc
// @Summary      Список товаров каталога (публичный)
// @Description  Только видимые (hidden=false). Фильтры, поиск, сортировка, пагинация.
// @Tags         catalog
// @Produce      json
// @Param        category_type     query  string   false  "figure | other"  Enums(figure, other)
// @Param        category_id       query  string   false  "UUID категории"
// @Param        subcategory_id    query  string   false  "UUID подкатегории"
// @Param        q                 query  string   false  "Поиск по name (ILIKE)"
// @Param        has_sale          query  bool     false  "Только со скидкой"
// @Param        sort              query  string   false  "created_desc | created_asc | price_asc | price_desc | name_asc | name_desc"  Enums(created_desc,created_asc,price_asc,price_desc,name_asc,name_desc)
// @Param        limit             query  int      false  "default 20, max 100"
// @Param        offset            query  int      false  "default 0"
// @Success      200  {object}  model.ItemCardPage
// @Failure      400  {object}  model.ErrorResponse
// @Router       /items [get]
func (h *CatalogHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	f, err := parseCatalogFilter(r, false)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	page, err := h.catalog.ListItems(r.Context(), f, false)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, page)
}

// GetItem godoc
// @Summary      Карточка товара (публичная)
// @Description  Работает и для скрытых товаров (hidden=true) — фронт показывает пометку.
// @Tags         catalog
// @Produce      json
// @Param        id   path  string  true  "UUID товара"
// @Success      200  {object}  model.ItemDetailDTO
// @Failure      404  {object}  model.ErrorResponse
// @Router       /items/{id} [get]
func (h *CatalogHandler) GetItem(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUIDPath(r, "id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	dto, err := h.catalog.GetItem(r.Context(), id)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, dto)
}

// ListCategories godoc
// @Summary      Список категорий
// @Tags         catalog
// @Produce      json
// @Param        type                 query  string  false  "figure | other"  Enums(figure, other)
// @Param        with_subcategories   query  bool    false  "Вложить подкатегории"
// @Success      200  {object}  map[string][]model.CategoryDTO
// @Router       /categories [get]
func (h *CatalogHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	var tp *model.CategoryType
	if v := r.URL.Query().Get("type"); v != "" {
		t := model.CategoryType(v)
		if t != model.CategoryTypeFigure && t != model.CategoryTypeOther {
			h.Err(w, r, model.NewInvalidInput("type должен быть figure или other"))
			return
		}
		tp = &t
	}
	with, _ := strconv.ParseBool(r.URL.Query().Get("with_subcategories"))
	cats, err := h.catalog.ListCategories(r.Context(), tp, with)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, map[string][]model.CategoryDTO{"categories": cats})
}

// ListSubcategories godoc
// @Summary      Подкатегории категории
// @Tags         catalog
// @Produce      json
// @Param        id   path  string  true  "UUID категории"
// @Success      200  {object}  map[string][]model.SubcategoryShortDTO
// @Failure      404  {object}  model.ErrorResponse
// @Router       /categories/{id}/subcategories [get]
func (h *CatalogHandler) ListSubcategories(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUIDPath(r, "id")
	if err != nil {
		h.Err(w, r, err)
		return
	}
	subs, err := h.catalog.ListSubcategories(r.Context(), id)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, map[string][]model.SubcategoryShortDTO{"subcategories": subs})
}

// ---------- helpers ----------

func parseCatalogFilter(r *http.Request, adminView bool) (model.CatalogFilter, error) {
	q := r.URL.Query()
	f := model.CatalogFilter{
		Query:      q.Get("q"),
		Sort:       q.Get("sort"),
		Pagination: ParsePagination(r),
	}
	if v := q.Get("category_type"); v != "" {
		if v != string(model.CategoryTypeFigure) && v != string(model.CategoryTypeOther) {
			return f, model.NewInvalidInput("category_type должен быть figure или other")
		}
		f.CategoryType = &v
	}
	if v := q.Get("category_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return f, model.NewInvalidInput("category_id: некорректный UUID")
		}
		f.CategoryID = &id
	}
	if v := q.Get("subcategory_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return f, model.NewInvalidInput("subcategory_id: некорректный UUID")
		}
		f.SubcategoryID = &id
	}
	if v := q.Get("has_sale"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return f, model.NewInvalidInput("has_sale: ожидается bool")
		}
		f.HasSale = &b
	}
	if adminView {
		f.IncludeHidden = true
		if v := q.Get("hidden"); v != "" && v != "any" {
			b, err := strconv.ParseBool(v)
			if err != nil {
				return f, model.NewInvalidInput("hidden: ожидается bool или 'any'")
			}
			f.HiddenOnly = &b
		}
	}
	return f, nil
}
