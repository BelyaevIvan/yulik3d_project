package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"yulik3d/internal/model"
	"yulik3d/internal/repository"
)

// CatalogService — публичная сторона каталога.
type CatalogService struct {
	items         *repository.ItemRepo
	pictures      *repository.PictureRepo
	options       *repository.ItemOptionRepo
	optionTypes   *repository.OptionTypeRepo
	itemSub       *repository.ItemSubcategoryRepo
	categories    *repository.CategoryRepo
	subcategories *repository.SubcategoryRepo
	mainPin       *repository.ItemMainPinRepo
	minio         *MinioClient
}

func NewCatalogService(
	items *repository.ItemRepo,
	pictures *repository.PictureRepo,
	options *repository.ItemOptionRepo,
	optionTypes *repository.OptionTypeRepo,
	itemSub *repository.ItemSubcategoryRepo,
	categories *repository.CategoryRepo,
	subcategories *repository.SubcategoryRepo,
	mainPin *repository.ItemMainPinRepo,
	minio *MinioClient,
) *CatalogService {
	return &CatalogService{
		items: items, pictures: pictures, options: options, optionTypes: optionTypes,
		itemSub: itemSub, categories: categories, subcategories: subcategories,
		mainPin: mainPin, minio: minio,
	}
}

// ListItems — публичный каталог (hidden=false).
func (s *CatalogService) ListItems(ctx context.Context, f model.CatalogFilter, adminView bool) (model.ListPage[model.ItemCardDTO], error) {
	f.Pagination.Clamp(20, 100)
	if !adminView {
		f.IncludeHidden = false
		f.HiddenOnly = nil
	}
	total, err := s.items.Count(ctx, f)
	if err != nil {
		return model.ListPage[model.ItemCardDTO]{}, fmt.Errorf("count: %w", err)
	}
	items, err := s.items.List(ctx, f)
	if err != nil {
		return model.ListPage[model.ItemCardDTO]{}, fmt.Errorf("list: %w", err)
	}
	cards, err := s.buildCards(ctx, items, adminView)
	if err != nil {
		return model.ListPage[model.ItemCardDTO]{}, err
	}
	return model.ListPage[model.ItemCardDTO]{
		Items:  cards,
		Total:  total,
		Limit:  f.Pagination.Limit,
		Offset: f.Pagination.Offset,
	}, nil
}

// ListMainPage — товары для главной страницы по типу. До 5 штук.
//
// Алгоритм:
//  1. Сначала закреплённые товары этого типа (через item_main_pin), отсортированные по position.
//     Если закреплённый оказался скрыт (теоретически не должно случаться, т.к. при скрытии
//     закрепление снимается, но на всякий случай) — пропускаем его.
//  2. Если набралось меньше 5 — добиваем «свежими видимыми товарами этого типа»
//     (created_at DESC), исключая уже добавленные.
//
// Все товары возвращаются как ItemCardDTO с картинками и категориями.
func (s *CatalogService) ListMainPage(ctx context.Context, t model.CategoryType) ([]model.ItemCardDTO, error) {
	const cap = 5

	// 1. Закреплённые
	pins, err := s.mainPin.ListByType(ctx, t)
	if err != nil {
		return nil, fmt.Errorf("list pins: %w", err)
	}
	pickedItems := make([]model.Item, 0, cap)
	pickedSet := make(map[uuid.UUID]bool, cap)
	for _, p := range pins {
		it, err := s.items.GetByID(ctx, p.ItemID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue // товар удалён, ON DELETE CASCADE должен был сработать — игнор
			}
			return nil, fmt.Errorf("get pinned item: %w", err)
		}
		if it.Hidden {
			continue // защита: показываем только видимые
		}
		pickedItems = append(pickedItems, it)
		pickedSet[it.ID] = true
		if len(pickedItems) >= cap {
			break
		}
	}

	// 2. Fallback — добираем до 5 свежими товарами этого типа
	if len(pickedItems) < cap {
		need := cap - len(pickedItems)
		// CatalogFilter.CategoryType — *string, поэтому приводим из enum-типа.
		typeStr := string(t)
		// Берём с запасом, потом отфильтруем уже выбранные.
		f := model.CatalogFilter{
			CategoryType: &typeStr,
			Sort:         "created_desc",
			Pagination:   model.Pagination{Limit: need + len(pickedItems), Offset: 0},
		}
		fallback, err := s.items.List(ctx, f)
		if err != nil {
			return nil, fmt.Errorf("fallback list: %w", err)
		}
		for _, it := range fallback {
			if pickedSet[it.ID] {
				continue
			}
			pickedItems = append(pickedItems, it)
			pickedSet[it.ID] = true
			if len(pickedItems) >= cap {
				break
			}
		}
	}

	return s.buildCards(ctx, pickedItems, false)
}

// BuildItemCards — публичный helper для переиспользования в FavoriteService.
func (s *CatalogService) BuildItemCards(ctx context.Context, items []model.Item) ([]model.ItemCardDTO, error) {
	return s.buildCards(ctx, items, true) // в избранном показываем hidden
}

func (s *CatalogService) buildCards(ctx context.Context, items []model.Item, adminView bool) ([]model.ItemCardDTO, error) {
	if len(items) == 0 {
		return []model.ItemCardDTO{}, nil
	}
	ids := make([]uuid.UUID, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}
	primaryKeys, err := s.pictures.PrimaryPictureKeys(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("primary pictures: %w", err)
	}
	batch, err := s.itemSub.ListForItems(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("subcategories batch: %w", err)
	}
	out := make([]model.ItemCardDTO, 0, len(items))
	for _, it := range items {
		var urlPtr *string
		if key, ok := primaryKeys[it.ID]; ok {
			u := s.minio.URL(key)
			urlPtr = &u
		}
		card := model.ItemCardDTO{
			ID:                it.ID,
			Name:              it.Name,
			Articul:           it.Articul,
			Price:             it.Price,
			Sale:              it.Sale,
			FinalPrice:        it.FinalPrice(),
			Hidden:            it.Hidden,
			PrimaryPictureURL: urlPtr,
			Subcategories:     batch.Subcategories[it.ID],
		}
		if cat, ok := batch.PrimaryCat[it.ID]; ok {
			c := cat
			card.Category = &c
		}
		if !adminView {
			card.Hidden = false // в публичной выдаче просто не показываем флаг
		}
		out = append(out, card)
	}
	return out, nil
}

// GetItem — полная карточка. Работает в т.ч. для hidden (с флагом в DTO).
func (s *CatalogService) GetItem(ctx context.Context, id uuid.UUID) (model.ItemDetailDTO, error) {
	it, err := s.items.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ItemDetailDTO{}, model.NewNotFound("Товар не найден")
		}
		return model.ItemDetailDTO{}, fmt.Errorf("get item: %w", err)
	}
	return s.composeDetail(ctx, it)
}

func (s *CatalogService) composeDetail(ctx context.Context, it model.Item) (model.ItemDetailDTO, error) {
	// Pictures
	picRows, err := s.pictures.ListByItem(ctx, it.ID)
	if err != nil {
		return model.ItemDetailDTO{}, fmt.Errorf("pictures: %w", err)
	}
	pics := make([]model.PictureDTO, 0, len(picRows))
	for _, r := range picRows {
		pics = append(pics, model.PictureDTO{
			ID:       r.PictureID,
			URL:      s.minio.URL(r.ObjectKey),
			Position: r.Position,
		})
	}

	// Options grouped
	opts, err := s.options.ListByItem(ctx, it.ID)
	if err != nil {
		return model.ItemDetailDTO{}, fmt.Errorf("options: %w", err)
	}
	typeIDs := make([]uuid.UUID, 0)
	typeSet := make(map[uuid.UUID]bool)
	for _, o := range opts {
		if !typeSet[o.TypeID] {
			typeSet[o.TypeID] = true
			typeIDs = append(typeIDs, o.TypeID)
		}
	}
	types, err := s.optionTypes.ListByIDs(ctx, typeIDs)
	if err != nil {
		return model.ItemDetailDTO{}, fmt.Errorf("option types: %w", err)
	}
	typesMap := make(map[uuid.UUID]model.OptionType, len(types))
	for _, t := range types {
		typesMap[t.ID] = t
	}
	groups := groupOptions(opts, typesMap)

	// Subcategories + category
	subs, err := s.itemSub.ListForItem(ctx, it.ID)
	if err != nil {
		return model.ItemDetailDTO{}, fmt.Errorf("subcategories: %w", err)
	}

	return model.ItemDetailDTO{
		ID:                it.ID,
		Name:              it.Name,
		Articul:           it.Articul,
		DescriptionInfo:   it.DescriptionInfo,
		DescriptionOther:  it.DescriptionOther,
		Price:             it.Price,
		Sale:              it.Sale,
		FinalPrice:        it.FinalPrice(),
		Hidden:            it.Hidden,
		Pictures:          pics,
		Options:           groups,
		Subcategories:     subs,
		CreatedAt:         it.CreatedAt,
		UpdatedAt:         it.UpdatedAt,
	}, nil
}

func groupOptions(opts []model.ItemOption, typesMap map[uuid.UUID]model.OptionType) []model.OptionGroupDTO {
	// Сохраняем порядок появления типа.
	order := []uuid.UUID{}
	seen := map[uuid.UUID]int{}
	for _, o := range opts {
		if _, ok := seen[o.TypeID]; !ok {
			seen[o.TypeID] = len(order)
			order = append(order, o.TypeID)
		}
	}
	groups := make([]model.OptionGroupDTO, len(order))
	for i, tid := range order {
		t := typesMap[tid]
		groups[i] = model.OptionGroupDTO{
			Type:   model.OptionTypeShortDTO{ID: t.ID, Code: t.Code, Label: t.Label},
			Values: []model.ItemOptionValueDTO{},
		}
	}
	for _, o := range opts {
		idx := seen[o.TypeID]
		groups[idx].Values = append(groups[idx].Values, model.ItemOptionValueDTO{
			ID:       o.ID,
			Value:    o.Value,
			Price:    o.Price,
			Position: o.Position,
		})
	}
	return groups
}

// ---------- Categories ----------

// ListCategories — с опциональной вложенной подгрузкой подкатегорий.
func (s *CatalogService) ListCategories(ctx context.Context, filterType *model.CategoryType, withSubcategories bool) ([]model.CategoryDTO, error) {
	cats, err := s.categories.List(ctx, filterType)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	out := make([]model.CategoryDTO, 0, len(cats))
	if !withSubcategories {
		for _, c := range cats {
			out = append(out, model.CategoryDTO{ID: c.ID, Name: c.Name, Type: c.Type})
		}
		return out, nil
	}
	ids := make([]uuid.UUID, 0, len(cats))
	for _, c := range cats {
		ids = append(ids, c.ID)
	}
	subs, err := s.subcategories.ListByCategoryIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("subcategories: %w", err)
	}
	subsByCat := make(map[uuid.UUID][]model.SubcategoryShortDTO)
	for _, sb := range subs {
		subsByCat[sb.CategoryID] = append(subsByCat[sb.CategoryID], model.SubcategoryShortDTO{ID: sb.ID, Name: sb.Name})
	}
	for _, c := range cats {
		out = append(out, model.CategoryDTO{
			ID:            c.ID,
			Name:          c.Name,
			Type:          c.Type,
			Subcategories: subsByCat[c.ID],
		})
	}
	return out, nil
}

// ListSubcategories — подкатегории конкретной категории.
func (s *CatalogService) ListSubcategories(ctx context.Context, categoryID uuid.UUID) ([]model.SubcategoryShortDTO, error) {
	if _, err := s.categories.GetByID(ctx, categoryID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.NewNotFound("Категория не найдена")
		}
		return nil, fmt.Errorf("get category: %w", err)
	}
	subs, err := s.subcategories.ListByCategory(ctx, categoryID)
	if err != nil {
		return nil, fmt.Errorf("list subcategories: %w", err)
	}
	out := make([]model.SubcategoryShortDTO, 0, len(subs))
	for _, sb := range subs {
		out = append(out, model.SubcategoryShortDTO{ID: sb.ID, Name: sb.Name})
	}
	return out, nil
}
