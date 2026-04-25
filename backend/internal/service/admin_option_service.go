package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"yulik3d/internal/model"
	"yulik3d/internal/repository"
)

type AdminOptionService struct {
	optionTypes *repository.OptionTypeRepo
	itemOptions *repository.ItemOptionRepo
	items       *repository.ItemRepo
}

func NewAdminOptionService(optionTypes *repository.OptionTypeRepo, itemOptions *repository.ItemOptionRepo, items *repository.ItemRepo) *AdminOptionService {
	return &AdminOptionService{optionTypes: optionTypes, itemOptions: itemOptions, items: items}
}

// ---------- Option types ----------

var codeRegex = regexp.MustCompile(`^[a-z0-9_]{2,50}$`)

func (s *AdminOptionService) ListTypes(ctx context.Context) ([]model.OptionTypeDTO, error) {
	types, err := s.optionTypes.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list option types: %w", err)
	}
	out := make([]model.OptionTypeDTO, 0, len(types))
	for _, t := range types {
		out = append(out, t.ToDTO())
	}
	return out, nil
}

func (s *AdminOptionService) CreateType(ctx context.Context, req model.OptionTypeCreateRequest) (model.OptionTypeDTO, error) {
	code := strings.ToLower(strings.TrimSpace(req.Code))
	label := strings.TrimSpace(req.Label)
	if !codeRegex.MatchString(code) {
		return model.OptionTypeDTO{}, model.NewInvalidInput("Код должен содержать только a-z, 0-9, _ (длина 2–50)")
	}
	if label == "" || len(label) > 100 {
		return model.OptionTypeDTO{}, model.NewInvalidInput("Укажите название (до 100 символов)")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return model.OptionTypeDTO{}, fmt.Errorf("uuid: %w", err)
	}
	t := model.OptionType{ID: id, Code: code, Label: label}
	if err := s.optionTypes.Create(ctx, &t); err != nil {
		if repository.PgErrCode(err) == repository.PgCodeUniqueViolation {
			return model.OptionTypeDTO{}, model.NewConflict("Тип опции с таким кодом уже существует")
		}
		return model.OptionTypeDTO{}, fmt.Errorf("create: %w", err)
	}
	return t.ToDTO(), nil
}

func (s *AdminOptionService) PatchType(ctx context.Context, id uuid.UUID, req model.OptionTypePatchRequest) (model.OptionTypeDTO, error) {
	if req.Label == nil {
		return model.OptionTypeDTO{}, model.NewInvalidInput("Нет полей для обновления")
	}
	label := strings.TrimSpace(*req.Label)
	if label == "" || len(label) > 100 {
		return model.OptionTypeDTO{}, model.NewInvalidInput("Укажите название (до 100 символов)")
	}
	t, err := s.optionTypes.PatchLabel(ctx, id, label)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.OptionTypeDTO{}, model.NewNotFound("Тип опции не найден")
		}
		return model.OptionTypeDTO{}, fmt.Errorf("patch: %w", err)
	}
	return t.ToDTO(), nil
}

func (s *AdminOptionService) DeleteType(ctx context.Context, id uuid.UUID) error {
	ok, err := s.optionTypes.Delete(ctx, id)
	if err != nil {
		if repository.PgErrCode(err) == repository.PgCodeForeignKeyViolation {
			return model.NewConflict("Тип опции используется в товарах — сначала удалите связанные опции")
		}
		return fmt.Errorf("delete: %w", err)
	}
	if !ok {
		return model.NewNotFound("Тип опции не найден")
	}
	return nil
}

// ---------- Item options ----------

func (s *AdminOptionService) CreateItemOption(ctx context.Context, itemID uuid.UUID, req model.ItemOptionCreateRequest) (model.ItemOptionDTO, error) {
	if _, err := s.items.GetByID(ctx, itemID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ItemOptionDTO{}, model.NewNotFound("Товар не найден")
		}
		return model.ItemOptionDTO{}, fmt.Errorf("get item: %w", err)
	}
	t, err := s.optionTypes.GetByID(ctx, req.TypeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ItemOptionDTO{}, model.NewInvalidInput("Указанный тип опции не существует")
		}
		return model.ItemOptionDTO{}, fmt.Errorf("get type: %w", err)
	}
	val := strings.TrimSpace(req.Value)
	if val == "" {
		return model.ItemOptionDTO{}, model.NewInvalidInput("Укажите значение опции")
	}
	if req.Price < 0 {
		return model.ItemOptionDTO{}, model.NewInvalidInput("Цена не может быть отрицательной")
	}

	id, err := uuid.NewV7()
	if err != nil {
		return model.ItemOptionDTO{}, fmt.Errorf("uuid: %w", err)
	}
	op := model.ItemOption{ID: id, ItemID: itemID, TypeID: req.TypeID, Value: val, Price: req.Price, Position: req.Position}
	if err := s.itemOptions.Create(ctx, &op); err != nil {
		if repository.PgErrCode(err) == repository.PgCodeUniqueViolation {
			return model.ItemOptionDTO{}, model.NewConflict("Такое значение для этого типа опции уже существует")
		}
		return model.ItemOptionDTO{}, fmt.Errorf("create: %w", err)
	}
	return model.ItemOptionDTO{
		ID:       op.ID,
		ItemID:   op.ItemID,
		Type:     model.OptionTypeShortDTO{ID: t.ID, Code: t.Code, Label: t.Label},
		Value:    op.Value,
		Price:    op.Price,
		Position: op.Position,
	}, nil
}

func (s *AdminOptionService) PatchItemOption(ctx context.Context, id uuid.UUID, req model.ItemOptionPatchRequest) (model.ItemOptionDTO, error) {
	if req.Value == nil && req.Price == nil && req.Position == nil {
		return model.ItemOptionDTO{}, model.NewInvalidInput("Нет полей для обновления")
	}
	if req.Value != nil {
		v := strings.TrimSpace(*req.Value)
		if v == "" {
			return model.ItemOptionDTO{}, model.NewInvalidInput("Значение не может быть пустым")
		}
		req.Value = &v
	}
	if req.Price != nil && *req.Price < 0 {
		return model.ItemOptionDTO{}, model.NewInvalidInput("Цена не может быть отрицательной")
	}

	op, err := s.itemOptions.Patch(ctx, id, req.Value, req.Price, req.Position)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ItemOptionDTO{}, model.NewNotFound("Опция не найдена")
		}
		if repository.PgErrCode(err) == repository.PgCodeUniqueViolation {
			return model.ItemOptionDTO{}, model.NewConflict("Такое значение для этого типа опции уже существует")
		}
		return model.ItemOptionDTO{}, fmt.Errorf("patch: %w", err)
	}
	t, err := s.optionTypes.GetByID(ctx, op.TypeID)
	if err != nil {
		return model.ItemOptionDTO{}, fmt.Errorf("get type: %w", err)
	}
	return model.ItemOptionDTO{
		ID:       op.ID,
		ItemID:   op.ItemID,
		Type:     model.OptionTypeShortDTO{ID: t.ID, Code: t.Code, Label: t.Label},
		Value:    op.Value,
		Price:    op.Price,
		Position: op.Position,
	}, nil
}

func (s *AdminOptionService) DeleteItemOption(ctx context.Context, id uuid.UUID) error {
	ok, err := s.itemOptions.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if !ok {
		return model.NewNotFound("Опция не найдена")
	}
	return nil
}
