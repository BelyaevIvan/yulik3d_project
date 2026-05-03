package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"yulik3d/internal/model"
	"yulik3d/internal/repository"
)

// AdminMainPageService — управление закреплением товаров на главной.
//
// Каждый товар может быть закреплён независимо в figure и/или other.
// Лимит 5 на тип, проверяется и в БД (UNIQUE+CHECK), и здесь (для понятных ошибок).
type AdminMainPageService struct {
	mainPin *repository.ItemMainPinRepo
	items   *repository.ItemRepo
	catalog *CatalogService
	tx      *repository.TxManager
	log     *slog.Logger
}

func NewAdminMainPageService(
	mainPin *repository.ItemMainPinRepo,
	items *repository.ItemRepo,
	catalog *CatalogService,
	tx *repository.TxManager,
	log *slog.Logger,
) *AdminMainPageService {
	return &AdminMainPageService{mainPin: mainPin, items: items, catalog: catalog, tx: tx, log: log}
}

// MainPageDTO — что показывает админ-экран «Главная страница».
type MainPageDTO struct {
	Figures []model.ItemCardDTO `json:"figures"`
	Others  []model.ItemCardDTO `json:"others"`
}

// List — закреплённые товары обоих типов с детальной инфой (карточки).
func (s *AdminMainPageService) List(ctx context.Context) (MainPageDTO, error) {
	figures, err := s.listType(ctx, model.CategoryTypeFigure)
	if err != nil {
		return MainPageDTO{}, fmt.Errorf("figures: %w", err)
	}
	others, err := s.listType(ctx, model.CategoryTypeOther)
	if err != nil {
		return MainPageDTO{}, fmt.Errorf("others: %w", err)
	}
	return MainPageDTO{Figures: figures, Others: others}, nil
}

func (s *AdminMainPageService) listType(ctx context.Context, t model.CategoryType) ([]model.ItemCardDTO, error) {
	pins, err := s.mainPin.ListByType(ctx, t)
	if err != nil {
		return nil, err
	}
	items := make([]model.Item, 0, len(pins))
	for _, p := range pins {
		it, err := s.items.GetByID(ctx, p.ItemID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return nil, err
		}
		items = append(items, it)
	}
	return s.catalog.BuildItemCards(ctx, items)
}

// Pin — закрепить товар. Если position == nil → следующая свободная.
//
// Защиты:
//   - Товар существует и не скрыт
//   - Товар принадлежит указанному типу (хотя бы одна подкатегория этого типа)
//   - Не превышен лимит 5
//   - Позиция не занята другим товаром (UNIQUE constraint на уровне БД)
//   - Товар ещё не закреплён в этом типе (PK constraint на уровне БД)
func (s *AdminMainPageService) Pin(ctx context.Context, itemID uuid.UUID, t model.CategoryType, position *int) error {
	it, err := s.items.GetByID(ctx, itemID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.NewNotFound("Товар не найден")
		}
		return fmt.Errorf("get item: %w", err)
	}
	if it.Hidden {
		return model.NewInvalidInput("Скрытый товар нельзя закрепить на главной")
	}
	ok, err := s.items.HasCategoryType(ctx, itemID, t)
	if err != nil {
		return fmt.Errorf("check type: %w", err)
	}
	if !ok {
		return model.NewInvalidInput("Этот товар не относится к выбранному разделу")
	}

	count, err := s.mainPin.CountByType(ctx, t)
	if err != nil {
		return fmt.Errorf("count: %w", err)
	}
	if count >= 5 {
		return model.NewInvalidInput("В разделе уже закреплено максимум 5 товаров")
	}

	// Если позиция не указана — берём следующую свободную (count + 1).
	pos := count + 1
	if position != nil {
		if *position < 1 || *position > 5 {
			return model.NewInvalidInput("Позиция должна быть от 1 до 5")
		}
		pos = *position
	}

	err = s.mainPin.Insert(ctx, repository.ItemMainPin{ItemID: itemID, Type: t, Position: pos})
	if err != nil {
		if errors.Is(err, repository.ErrPositionTaken) {
			return model.NewConflict("Эта позиция уже занята другим товаром или товар уже закреплён в этом разделе")
		}
		return fmt.Errorf("insert: %w", err)
	}
	return nil
}

// Unpin — открепить товар + уплотнить оставшиеся позиции.
//
// DELETE и Compact (внутри которого DELETE+INSERT) выполняются в одной
// транзакции, чтобы при сбое не остаться в состоянии «запись удалена,
// но позиции не уплотнены» (это создавало бы дырку 1, 3, 4).
func (s *AdminMainPageService) Unpin(ctx context.Context, itemID uuid.UUID, t model.CategoryType) error {
	var notFound bool
	err := s.tx.Run(ctx, func(ctx context.Context) error {
		deleted, err := s.mainPin.DeleteByItemAndType(ctx, itemID, t)
		if err != nil {
			return fmt.Errorf("delete: %w", err)
		}
		if !deleted {
			notFound = true
			return nil
		}
		if err := s.mainPin.Compact(ctx, t); err != nil {
			return fmt.Errorf("compact: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if notFound {
		return model.NewNotFound("Закрепление не найдено")
	}
	return nil
}

// ReorderEntry — элемент массива при drag-and-drop.
type ReorderEntry struct {
	ItemID   uuid.UUID `json:"item_id"`
	Position int       `json:"position"`
}

// Reorder — атомарно сменить порядок закреплений в типе.
//
// На вход — полный набор закреплений типа в новом порядке. Сервис проверяет:
//   - все ItemID существуют в текущем закреплении этого типа (нельзя добавить
//     новый товар через reorder — для этого Pin)
//   - позиции 1..N без дубликатов и без дырок
func (s *AdminMainPageService) Reorder(ctx context.Context, t model.CategoryType, entries []ReorderEntry) error {
	if len(entries) == 0 {
		return model.NewInvalidInput("Передайте текущий список закреплений")
	}
	// Должно быть ровно столько элементов, сколько закреплений сейчас.
	current, err := s.mainPin.ListByType(ctx, t)
	if err != nil {
		return fmt.Errorf("list current: %w", err)
	}
	if len(entries) != len(current) {
		return model.NewInvalidInput("Передан неполный или избыточный список закреплений")
	}
	currentSet := make(map[uuid.UUID]bool, len(current))
	for _, c := range current {
		currentSet[c.ItemID] = true
	}

	seenItem := make(map[uuid.UUID]bool, len(entries))
	seenPos := make(map[int]bool, len(entries))
	for _, e := range entries {
		if !currentSet[e.ItemID] {
			return model.NewInvalidInput("Один из товаров не закреплён в этом разделе")
		}
		if seenItem[e.ItemID] {
			return model.NewInvalidInput("Дублирующийся товар в порядке")
		}
		seenItem[e.ItemID] = true
		if e.Position < 1 || e.Position > len(entries) {
			return model.NewInvalidInput("Некорректная позиция")
		}
		if seenPos[e.Position] {
			return model.NewInvalidInput("Дублирующаяся позиция")
		}
		seenPos[e.Position] = true
	}

	pins := make([]repository.ItemMainPin, 0, len(entries))
	for _, e := range entries {
		pins = append(pins, repository.ItemMainPin{ItemID: e.ItemID, Type: t, Position: e.Position})
	}

	err = s.tx.Run(ctx, func(ctx context.Context) error {
		return s.mainPin.ReplaceForType(ctx, t, pins)
	})
	if err != nil {
		if errors.Is(err, repository.ErrPositionTaken) {
			return model.NewConflict("Конфликт позиций — обновите страницу")
		}
		return err
	}
	return nil
}
