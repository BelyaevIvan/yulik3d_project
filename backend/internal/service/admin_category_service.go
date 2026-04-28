package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"yulik3d/internal/model"
	"yulik3d/internal/repository"
)

type AdminCategoryService struct {
	categories    *repository.CategoryRepo
	subcategories *repository.SubcategoryRepo
}

func NewAdminCategoryService(categories *repository.CategoryRepo, subcategories *repository.SubcategoryRepo) *AdminCategoryService {
	return &AdminCategoryService{categories: categories, subcategories: subcategories}
}

// ---------- Categories ----------

func (s *AdminCategoryService) CreateCategory(ctx context.Context, req model.CategoryCreateRequest) (model.CategoryDTO, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return model.CategoryDTO{}, model.NewInvalidInput("Укажите название категории")
	}
	if req.Type != model.CategoryTypeFigure && req.Type != model.CategoryTypeOther {
		return model.CategoryDTO{}, model.NewInvalidInput("Тип должен быть figure или other")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return model.CategoryDTO{}, fmt.Errorf("uuid: %w", err)
	}
	c := model.Category{ID: id, Name: name, Type: req.Type}
	if err := s.categories.Create(ctx, &c); err != nil {
		return model.CategoryDTO{}, fmt.Errorf("create: %w", err)
	}
	return model.CategoryDTO{ID: c.ID, Name: c.Name, Type: c.Type}, nil
}

func (s *AdminCategoryService) PatchCategory(ctx context.Context, id uuid.UUID, req model.CategoryPatchRequest) (model.CategoryDTO, error) {
	if req.Name == nil && req.Type == nil {
		return model.CategoryDTO{}, model.NewInvalidInput("Нет полей для обновления")
	}
	if req.Name != nil {
		v := strings.TrimSpace(*req.Name)
		if v == "" {
			return model.CategoryDTO{}, model.NewInvalidInput("Название не может быть пустым")
		}
		req.Name = &v
	}
	if req.Type != nil {
		if *req.Type != model.CategoryTypeFigure && *req.Type != model.CategoryTypeOther {
			return model.CategoryDTO{}, model.NewInvalidInput("Тип должен быть figure или other")
		}
	}
	c, err := s.categories.Patch(ctx, id, req.Name, req.Type)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.CategoryDTO{}, model.NewNotFound("Категория не найдена")
		}
		return model.CategoryDTO{}, fmt.Errorf("patch: %w", err)
	}
	return model.CategoryDTO{ID: c.ID, Name: c.Name, Type: c.Type}, nil
}

func (s *AdminCategoryService) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	if _, err := s.categories.GetByID(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.NewNotFound("Категория не найдена")
		}
		return fmt.Errorf("get category: %w", err)
	}
	n, err := s.categories.CountItems(ctx, id)
	if err != nil {
		return fmt.Errorf("count items: %w", err)
	}
	if n > 0 {
		return model.NewInvalidInput(fmt.Sprintf(
			"В категории есть товары (%d шт.). Сначала отвяжите их от подкатегорий этой категории или смените им подкатегорию.", n,
		))
	}
	ok, err := s.categories.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if !ok {
		return model.NewNotFound("Категория не найдена")
	}
	return nil
}

// ---------- Subcategories ----------

func (s *AdminCategoryService) CreateSubcategory(ctx context.Context, categoryID uuid.UUID, req model.SubcategoryCreateRequest) (model.SubcategoryDTO, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return model.SubcategoryDTO{}, model.NewInvalidInput("Укажите название подкатегории")
	}
	if _, err := s.categories.GetByID(ctx, categoryID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.SubcategoryDTO{}, model.NewNotFound("Категория не найдена")
		}
		return model.SubcategoryDTO{}, fmt.Errorf("get category: %w", err)
	}
	id, err := uuid.NewV7()
	if err != nil {
		return model.SubcategoryDTO{}, fmt.Errorf("uuid: %w", err)
	}
	sub := model.Subcategory{ID: id, Name: name, CategoryID: categoryID}
	if err := s.subcategories.Create(ctx, &sub); err != nil {
		return model.SubcategoryDTO{}, fmt.Errorf("create: %w", err)
	}
	return model.SubcategoryDTO{
		ID: sub.ID, Name: sub.Name, CategoryID: sub.CategoryID, CreatedAt: sub.CreatedAt,
	}, nil
}

func (s *AdminCategoryService) PatchSubcategory(ctx context.Context, id uuid.UUID, req model.SubcategoryPatchRequest) (model.SubcategoryDTO, error) {
	if req.Name == nil && req.CategoryID == nil {
		return model.SubcategoryDTO{}, model.NewInvalidInput("Нет полей для обновления")
	}
	if req.Name != nil {
		v := strings.TrimSpace(*req.Name)
		if v == "" {
			return model.SubcategoryDTO{}, model.NewInvalidInput("Название не может быть пустым")
		}
		req.Name = &v
	}
	if req.CategoryID != nil {
		if _, err := s.categories.GetByID(ctx, *req.CategoryID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return model.SubcategoryDTO{}, model.NewInvalidInput("Указанная категория не существует")
			}
			return model.SubcategoryDTO{}, fmt.Errorf("check category: %w", err)
		}
	}
	sub, err := s.subcategories.Patch(ctx, id, req.Name, req.CategoryID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.SubcategoryDTO{}, model.NewNotFound("Подкатегория не найдена")
		}
		return model.SubcategoryDTO{}, fmt.Errorf("patch: %w", err)
	}
	return model.SubcategoryDTO{
		ID: sub.ID, Name: sub.Name, CategoryID: sub.CategoryID, CreatedAt: sub.CreatedAt,
	}, nil
}

func (s *AdminCategoryService) DeleteSubcategory(ctx context.Context, id uuid.UUID) error {
	if _, err := s.subcategories.GetByID(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.NewNotFound("Подкатегория не найдена")
		}
		return fmt.Errorf("get subcategory: %w", err)
	}
	n, err := s.subcategories.CountItems(ctx, id)
	if err != nil {
		return fmt.Errorf("count items: %w", err)
	}
	if n > 0 {
		return model.NewInvalidInput(fmt.Sprintf(
			"В подкатегории есть товары (%d шт.). Сначала отвяжите их от этой подкатегории.", n,
		))
	}
	ok, err := s.subcategories.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if !ok {
		return model.NewNotFound("Подкатегория не найдена")
	}
	return nil
}
