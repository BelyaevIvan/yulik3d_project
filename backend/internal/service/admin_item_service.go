package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"yulik3d/internal/model"
	"yulik3d/internal/repository"
)

type AdminItemService struct {
	items         *repository.ItemRepo
	itemOptions   *repository.ItemOptionRepo
	optionTypes   *repository.OptionTypeRepo
	subcategories *repository.SubcategoryRepo
	mainPin       *repository.ItemMainPinRepo
	catalog       *CatalogService
	tx            *repository.TxManager
}

func NewAdminItemService(
	items *repository.ItemRepo,
	itemOptions *repository.ItemOptionRepo,
	optionTypes *repository.OptionTypeRepo,
	subcategories *repository.SubcategoryRepo,
	mainPin *repository.ItemMainPinRepo,
	catalog *CatalogService,
	tx *repository.TxManager,
) *AdminItemService {
	return &AdminItemService{items: items, itemOptions: itemOptions, optionTypes: optionTypes,
		subcategories: subcategories, mainPin: mainPin, catalog: catalog, tx: tx}
}

// syncMainPinAfterItemChange — после изменения товара (Update/Patch) синхронизирует
// закрепления на главной с актуальным состоянием:
//   - если товар стал hidden → снимаем все закрепления
//   - иначе оставляем только те закрепления, типы которых ещё актуальны
//     (товар по-прежнему относится к этому типу через свои подкатегории)
//
// Все ошибки логируются и проглатываются — основной апдейт уже произошёл.
// После Unpin нужно сжать позиции в затронутом типе (Compact).
// logFn — упрощённая обёртка над slog.Default для адаптивного logging
// без необходимости таскать *slog.Logger в этом сервисе.
func logErr(msg string, args ...any) {
	slog.Default().Error(msg, args...)
}

func (s *AdminItemService) syncMainPinAfterItemChange(ctx context.Context, itemID uuid.UUID, log func(msg string, args ...any)) {
	it, err := s.items.GetByID(ctx, itemID)
	if err != nil {
		log("sync main_pin: get item", "err", err, "item_id", itemID)
		return
	}
	pins, err := s.mainPin.ListByItem(ctx, itemID)
	if err != nil {
		log("sync main_pin: list pins", "err", err, "item_id", itemID)
		return
	}
	if len(pins) == 0 {
		return
	}

	if it.Hidden {
		// Скрытый товар — снимаем со всех закреплений и уплотняем оба типа.
		// Всё в одной транзакции: при сбое не должно остаться «дырок» в позициях.
		err := s.tx.Run(ctx, func(ctx context.Context) error {
			if err := s.mainPin.DeleteByItem(ctx, itemID); err != nil {
				return fmt.Errorf("delete by item: %w", err)
			}
			for _, p := range pins {
				if err := s.mainPin.Compact(ctx, p.Type); err != nil {
					return fmt.Errorf("compact %s: %w", p.Type, err)
				}
			}
			return nil
		})
		if err != nil {
			log("sync main_pin: hide tx", "err", err, "item_id", itemID)
		}
		return
	}

	// Не скрыт — проверяем актуальность типов через подкатегории.
	types, err := s.items.CategoryTypesForItem(ctx, itemID)
	if err != nil {
		log("sync main_pin: types for item", "err", err, "item_id", itemID)
		return
	}
	typeSet := make(map[model.CategoryType]bool, len(types))
	for _, t := range types {
		typeSet[t] = true
	}
	for _, p := range pins {
		if typeSet[p.Type] {
			continue // тип ещё актуален, закрепление остаётся
		}
		// Атомарно: удалить + уплотнить.
		err := s.tx.Run(ctx, func(ctx context.Context) error {
			if _, err := s.mainPin.DeleteByItemAndType(ctx, itemID, p.Type); err != nil {
				return fmt.Errorf("delete by type: %w", err)
			}
			if err := s.mainPin.Compact(ctx, p.Type); err != nil {
				return fmt.Errorf("compact: %w", err)
			}
			return nil
		})
		if err != nil {
			log("sync main_pin: type-loss tx", "err", err, "item_id", itemID, "type", p.Type)
		}
	}
}

func (s *AdminItemService) Create(ctx context.Context, req model.ItemCreateRequest) (model.ItemDetailDTO, error) {
	if err := validateItemCreate(req); err != nil {
		return model.ItemDetailDTO{}, err
	}
	// Проверить, что все subcategory_id и type_id существуют
	if ok, err := s.subcategories.ExistIDs(ctx, req.SubcategoryIDs); err != nil {
		return model.ItemDetailDTO{}, fmt.Errorf("check subcategories: %w", err)
	} else if !ok {
		return model.ItemDetailDTO{}, model.NewInvalidInput("Некоторые указанные подкатегории не существуют")
	}
	if err := s.checkOptionTypesExist(ctx, req.Options); err != nil {
		return model.ItemDetailDTO{}, err
	}
	if err := checkOptionDuplicates(req.Options); err != nil {
		return model.ItemDetailDTO{}, err
	}

	var createdID uuid.UUID
	err := s.tx.Run(ctx, func(ctx context.Context) error {
		articul, err := s.items.NextArticul(ctx)
		if err != nil {
			return fmt.Errorf("next articul: %w", err)
		}
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("uuid: %w", err)
		}
		it := model.Item{
			ID:               id,
			Name:             strings.TrimSpace(req.Name),
			DescriptionInfo:  req.DescriptionInfo,
			DescriptionOther: req.DescriptionOther,
			Price:            req.Price,
			Sale:             req.Sale,
			Articul:          articul,
			Hidden:           req.Hidden,
		}
		if err := s.items.Create(ctx, &it); err != nil {
			return fmt.Errorf("create item: %w", err)
		}
		if err := s.items.AttachSubcategories(ctx, id, req.SubcategoryIDs); err != nil {
			return fmt.Errorf("attach subcategories: %w", err)
		}
		for _, o := range req.Options {
			oid, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("uuid: %w", err)
			}
			op := model.ItemOption{
				ID: oid, ItemID: id, TypeID: o.TypeID,
				Value: strings.TrimSpace(o.Value), Price: o.Price, Position: o.Position,
			}
			if err := s.itemOptions.Create(ctx, &op); err != nil {
				return fmt.Errorf("create option: %w", err)
			}
		}
		createdID = id
		return nil
	})
	if err != nil {
		return model.ItemDetailDTO{}, err
	}
	it, err := s.items.GetByID(ctx, createdID)
	if err != nil {
		return model.ItemDetailDTO{}, fmt.Errorf("reload: %w", err)
	}
	return s.catalog.composeDetail(ctx, it)
}

func (s *AdminItemService) Get(ctx context.Context, id uuid.UUID) (model.ItemDetailDTO, error) {
	it, err := s.items.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ItemDetailDTO{}, model.NewNotFound("Товар не найден")
		}
		return model.ItemDetailDTO{}, fmt.Errorf("get item: %w", err)
	}
	return s.catalog.composeDetail(ctx, it)
}

func (s *AdminItemService) Update(ctx context.Context, id uuid.UUID, req model.ItemUpdateRequest) (model.ItemDetailDTO, error) {
	if err := validateItemUpdate(req); err != nil {
		return model.ItemDetailDTO{}, err
	}
	if ok, err := s.subcategories.ExistIDs(ctx, req.SubcategoryIDs); err != nil {
		return model.ItemDetailDTO{}, fmt.Errorf("check subcategories: %w", err)
	} else if !ok {
		return model.ItemDetailDTO{}, model.NewInvalidInput("Некоторые указанные подкатегории не существуют")
	}
	if err := s.checkOptionTypesExist(ctx, req.Options); err != nil {
		return model.ItemDetailDTO{}, err
	}
	if err := checkOptionDuplicates(req.Options); err != nil {
		return model.ItemDetailDTO{}, err
	}

	err := s.tx.Run(ctx, func(ctx context.Context) error {
		it := model.Item{
			ID:               id,
			Name:             strings.TrimSpace(req.Name),
			DescriptionInfo:  req.DescriptionInfo,
			DescriptionOther: req.DescriptionOther,
			Price:            req.Price,
			Sale:             req.Sale,
			Hidden:           req.Hidden,
		}
		if err := s.items.Update(ctx, &it); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return model.NewNotFound("Товар не найден")
			}
			return fmt.Errorf("update item: %w", err)
		}
		if err := s.items.ReplaceSubcategories(ctx, id, req.SubcategoryIDs); err != nil {
			return fmt.Errorf("replace subcategories: %w", err)
		}
		if err := s.itemOptions.DeleteByItem(ctx, id); err != nil {
			return fmt.Errorf("delete options: %w", err)
		}
		for _, o := range req.Options {
			oid, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("uuid: %w", err)
			}
			op := model.ItemOption{
				ID: oid, ItemID: id, TypeID: o.TypeID,
				Value: strings.TrimSpace(o.Value), Price: o.Price, Position: o.Position,
			}
			if err := s.itemOptions.Create(ctx, &op); err != nil {
				return fmt.Errorf("create option: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return model.ItemDetailDTO{}, err
	}
	// Подкатегории/скрытость могли поменяться — синхронизируем закрепления.
	s.syncMainPinAfterItemChange(ctx, id, logErr)
	it, err := s.items.GetByID(ctx, id)
	if err != nil {
		return model.ItemDetailDTO{}, fmt.Errorf("reload: %w", err)
	}
	return s.catalog.composeDetail(ctx, it)
}

func (s *AdminItemService) Patch(ctx context.Context, id uuid.UUID, p model.ItemPatchRequest) (model.ItemDetailDTO, error) {
	if p.Name == nil && p.DescriptionInfo == nil && p.DescriptionOther == nil &&
		p.Price == nil && p.Sale == nil && p.Hidden == nil {
		return model.ItemDetailDTO{}, model.NewInvalidInput("Нет полей для обновления")
	}
	if p.Sale != nil && (*p.Sale < 0 || *p.Sale > 100) {
		return model.ItemDetailDTO{}, model.NewInvalidInput("Скидка должна быть от 0 до 100")
	}
	if p.Price != nil && *p.Price < 0 {
		return model.ItemDetailDTO{}, model.NewInvalidInput("Цена не может быть отрицательной")
	}
	if p.Name != nil {
		v := strings.TrimSpace(*p.Name)
		if v == "" {
			return model.ItemDetailDTO{}, model.NewInvalidInput("Название не может быть пустым")
		}
		p.Name = &v
	}
	if p.DescriptionInfo != nil && strings.TrimSpace(*p.DescriptionInfo) == "" {
		return model.ItemDetailDTO{}, model.NewInvalidInput("«Информация о товаре» не может быть пустой")
	}
	if p.DescriptionOther != nil && strings.TrimSpace(*p.DescriptionOther) == "" {
		return model.ItemDetailDTO{}, model.NewInvalidInput("«Особенности» не могут быть пустыми")
	}
	it, err := s.items.Patch(ctx, id, p)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ItemDetailDTO{}, model.NewNotFound("Товар не найден")
		}
		return model.ItemDetailDTO{}, fmt.Errorf("patch item: %w", err)
	}
	// Если поменяли hidden — синхронизируем закрепления.
	if p.Hidden != nil {
		s.syncMainPinAfterItemChange(ctx, id, logErr)
	}
	return s.catalog.composeDetail(ctx, it)
}

// ---------- helpers ----------

func (s *AdminItemService) checkOptionTypesExist(ctx context.Context, opts []model.ItemOptionCreateRequest) error {
	if len(opts) == 0 {
		return nil
	}
	set := map[uuid.UUID]bool{}
	ids := make([]uuid.UUID, 0)
	for _, o := range opts {
		if !set[o.TypeID] {
			set[o.TypeID] = true
			ids = append(ids, o.TypeID)
		}
	}
	types, err := s.optionTypes.ListByIDs(ctx, ids)
	if err != nil {
		return fmt.Errorf("list option types: %w", err)
	}
	if len(types) != len(ids) {
		return model.NewInvalidInput("Некоторые указанные типы опций не существуют")
	}
	return nil
}

func checkOptionDuplicates(opts []model.ItemOptionCreateRequest) error {
	type key struct {
		tid   uuid.UUID
		value string
	}
	seen := map[key]bool{}
	for _, o := range opts {
		k := key{o.TypeID, strings.TrimSpace(o.Value)}
		if seen[k] {
			return model.NewInvalidInput(fmt.Sprintf("Дублирующееся значение опции: %s", o.Value))
		}
		seen[k] = true
	}
	return nil
}

func validateItemCreate(r model.ItemCreateRequest) error {
	if err := validateItemCommon(r.Name, r.DescriptionInfo, r.DescriptionOther, r.Price, r.Sale); err != nil {
		return err
	}
	if len(r.SubcategoryIDs) == 0 {
		return model.NewInvalidInput("Укажите хотя бы одну подкатегорию (вместе с её категорией)")
	}
	return nil
}

func validateItemUpdate(r model.ItemUpdateRequest) error {
	if err := validateItemCommon(r.Name, r.DescriptionInfo, r.DescriptionOther, r.Price, r.Sale); err != nil {
		return err
	}
	if len(r.SubcategoryIDs) == 0 {
		return model.NewInvalidInput("У товара должна остаться хотя бы одна подкатегория (вместе с её категорией)")
	}
	return nil
}

func validateItemCommon(name, info, other string, price, sale int) error {
	if strings.TrimSpace(name) == "" {
		return model.NewInvalidInput("Укажите название")
	}
	if strings.TrimSpace(info) == "" {
		return model.NewInvalidInput("Укажите описание (Информация о товаре)")
	}
	if strings.TrimSpace(other) == "" {
		return model.NewInvalidInput("Укажите особенности товара")
	}
	if price < 0 {
		return model.NewInvalidInput("Цена не может быть отрицательной")
	}
	if sale < 0 || sale > 100 {
		return model.NewInvalidInput("Скидка должна быть от 0 до 100")
	}
	return nil
}
