package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"yulik3d/internal/model"
	"yulik3d/internal/repository"
)

type AdminPictureService struct {
	items    *repository.ItemRepo
	pictures *repository.PictureRepo
	minio    *MinioClient
	tx       *repository.TxManager
	maxBytes int64
}

func NewAdminPictureService(items *repository.ItemRepo, pictures *repository.PictureRepo, minio *MinioClient, tx *repository.TxManager, maxBytes int64) *AdminPictureService {
	return &AdminPictureService{items: items, pictures: pictures, minio: minio, tx: tx, maxBytes: maxBytes}
}

// allowedMimes — что принимаем.
var allowedMimes = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/webp": ".webp",
}

// Upload — грузит файл в MinIO и создаёт записи picture + item_picture.
func (s *AdminPictureService) Upload(ctx context.Context, itemID uuid.UUID, r io.Reader, size int64, filename, contentType string, position *int) (model.PictureDTO, error) {
	// Проверка существования товара
	if _, err := s.items.GetByID(ctx, itemID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.PictureDTO{}, model.NewNotFound("Товар не найден")
		}
		return model.PictureDTO{}, fmt.Errorf("get item: %w", err)
	}

	// Размер
	if size > s.maxBytes {
		return model.PictureDTO{}, model.NewInvalidInput("Файл слишком большой")
	}
	if size <= 0 {
		return model.PictureDTO{}, model.NewInvalidInput("Пустой файл")
	}

	// MIME
	ct := strings.Split(contentType, ";")[0]
	ct = strings.TrimSpace(ct)
	ext, ok := allowedMimes[ct]
	if !ok {
		// Попытка из расширения файла
		if guessed := mime.TypeByExtension(filepath.Ext(filename)); guessed != "" {
			ct = strings.Split(guessed, ";")[0]
			ext, ok = allowedMimes[ct]
		}
	}
	if !ok {
		return model.PictureDTO{}, model.NewInvalidInput("Неподдерживаемый формат (только png, jpg, webp)")
	}

	// Генерация id и object_key
	picID, err := uuid.NewV7()
	if err != nil {
		return model.PictureDTO{}, fmt.Errorf("uuid: %w", err)
	}
	objectKey := fmt.Sprintf("items/%s/%s%s", itemID.String(), picID.String(), ext)

	// Upload в MinIO сначала (снаружи транзакции).
	if err := s.minio.Put(ctx, objectKey, r, size, ct); err != nil {
		return model.PictureDTO{}, fmt.Errorf("minio put: %w", err)
	}

	// Запись в БД — атомарно.
	var finalPos int
	err = s.tx.Run(ctx, func(ctx context.Context) error {
		p := &model.Picture{ID: picID, ObjectKey: objectKey}
		if err := s.pictures.CreatePicture(ctx, p); err != nil {
			return fmt.Errorf("create picture: %w", err)
		}
		if position != nil {
			finalPos = *position
		} else {
			np, err := s.pictures.NextPosition(ctx, itemID)
			if err != nil {
				return fmt.Errorf("next position: %w", err)
			}
			finalPos = np
		}
		if err := s.pictures.AttachToItem(ctx, itemID, picID, finalPos); err != nil {
			return fmt.Errorf("attach: %w", err)
		}
		return nil
	})
	if err != nil {
		// Компенсация — удалить файл из MinIO
		_ = s.minio.Delete(ctx, objectKey)
		return model.PictureDTO{}, err
	}

	return model.PictureDTO{
		ID:       picID,
		URL:      s.minio.URL(objectKey),
		Position: finalPos,
	}, nil
}

// Delete — удалить связь, и если картинка больше нигде не используется — удалить её из MinIO.
func (s *AdminPictureService) Delete(ctx context.Context, itemID, pictureID uuid.UUID) error {
	var keyToDelete string
	err := s.tx.Run(ctx, func(ctx context.Context) error {
		ok, err := s.pictures.DeleteLink(ctx, itemID, pictureID)
		if err != nil {
			return fmt.Errorf("delete link: %w", err)
		}
		if !ok {
			return model.NewNotFound("Связь с картинкой не найдена")
		}
		n, err := s.pictures.CountLinks(ctx, pictureID)
		if err != nil {
			return fmt.Errorf("count links: %w", err)
		}
		if n == 0 {
			key, err := s.pictures.DeletePicture(ctx, pictureID)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("delete picture: %w", err)
			}
			keyToDelete = key
		}
		return nil
	})
	if err != nil {
		return err
	}
	if keyToDelete != "" {
		_ = s.minio.Delete(ctx, keyToDelete) // best-effort
	}
	return nil
}

// Reorder — задать новые позиции. В теле должны быть перечислены все текущие картинки товара.
func (s *AdminPictureService) Reorder(ctx context.Context, itemID uuid.UUID, req model.ReorderRequest) ([]model.PictureDTO, error) {
	if _, err := s.items.GetByID(ctx, itemID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.NewNotFound("Товар не найден")
		}
		return nil, fmt.Errorf("get item: %w", err)
	}
	current, err := s.pictures.ListByItem(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("list pictures: %w", err)
	}
	if len(req.Order) != len(current) {
		return nil, model.NewInvalidInput("В order нужно передать все текущие картинки товара")
	}
	cur := make(map[uuid.UUID]bool, len(current))
	for _, c := range current {
		cur[c.PictureID] = true
	}
	seenPos := make(map[int]bool, len(req.Order))
	for _, e := range req.Order {
		if !cur[e.PictureID] {
			return nil, model.NewInvalidInput("Картинка не принадлежит этому товару")
		}
		if e.Position <= 0 {
			return nil, model.NewInvalidInput("Позиция должна быть положительной")
		}
		if seenPos[e.Position] {
			return nil, model.NewInvalidInput("Дублирующаяся позиция")
		}
		seenPos[e.Position] = true
	}

	err = s.tx.Run(ctx, func(ctx context.Context) error {
		for _, e := range req.Order {
			if err := s.pictures.UpdatePosition(ctx, itemID, e.PictureID, e.Position); err != nil {
				return fmt.Errorf("update position: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	picRows, err := s.pictures.ListByItem(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("reload: %w", err)
	}
	out := make([]model.PictureDTO, 0, len(picRows))
	for _, r := range picRows {
		out = append(out, model.PictureDTO{ID: r.PictureID, URL: s.minio.URL(r.ObjectKey), Position: r.Position})
	}
	return out, nil
}
