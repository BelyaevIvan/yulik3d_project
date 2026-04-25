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

type FavoriteService struct {
	favs    *repository.FavoriteRepo
	items   *repository.ItemRepo
	catalog *CatalogService
}

func NewFavoriteService(favs *repository.FavoriteRepo, items *repository.ItemRepo, catalog *CatalogService) *FavoriteService {
	return &FavoriteService{favs: favs, items: items, catalog: catalog}
}

func (s *FavoriteService) List(ctx context.Context, userID uuid.UUID, p model.Pagination) (model.ListPage[model.ItemCardDTO], error) {
	p.Clamp(20, 100)
	total, err := s.favs.Count(ctx, userID)
	if err != nil {
		return model.ListPage[model.ItemCardDTO]{}, fmt.Errorf("count: %w", err)
	}
	items, err := s.favs.ListItems(ctx, userID, p)
	if err != nil {
		return model.ListPage[model.ItemCardDTO]{}, fmt.Errorf("list: %w", err)
	}
	cards, err := s.catalog.BuildItemCards(ctx, items)
	if err != nil {
		return model.ListPage[model.ItemCardDTO]{}, err
	}
	return model.ListPage[model.ItemCardDTO]{
		Items:  cards,
		Total:  total,
		Limit:  p.Limit,
		Offset: p.Offset,
	}, nil
}

func (s *FavoriteService) Add(ctx context.Context, userID, itemID uuid.UUID) (model.FavoriteAddResponse, error) {
	if _, err := s.items.GetByID(ctx, itemID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.FavoriteAddResponse{}, model.NewNotFound("Товар не найден")
		}
		return model.FavoriteAddResponse{}, fmt.Errorf("get item: %w", err)
	}
	t, err := s.favs.Add(ctx, userID, itemID)
	if err != nil {
		return model.FavoriteAddResponse{}, fmt.Errorf("add favorite: %w", err)
	}
	return model.FavoriteAddResponse{ItemID: itemID, CreatedAt: t}, nil
}

func (s *FavoriteService) Remove(ctx context.Context, userID, itemID uuid.UUID) error {
	if _, err := s.favs.Remove(ctx, userID, itemID); err != nil {
		return fmt.Errorf("remove favorite: %w", err)
	}
	return nil
}
