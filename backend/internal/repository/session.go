package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"yulik3d/internal/model"
)

// SessionRepo — хранит сессии в Redis. Ключ session:<id>, JSON-значение.
type SessionRepo struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewSessionRepo(rdb *redis.Client, ttl time.Duration) *SessionRepo {
	return &SessionRepo{rdb: rdb, ttl: ttl}
}

// ErrSessionNotFound — сессии нет или протухла.
var ErrSessionNotFound = errors.New("session not found")

// Create кладёт сессию в Redis с TTL из конфига.
func (r *SessionRepo) Create(ctx context.Context, id string, s model.Session) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return r.rdb.Set(ctx, key(id), b, r.ttl).Err()
}

// Get возвращает сессию или ErrSessionNotFound.
func (r *SessionRepo) Get(ctx context.Context, id string) (model.Session, error) {
	b, err := r.rdb.Get(ctx, key(id)).Bytes()
	if errors.Is(err, redis.Nil) {
		return model.Session{}, ErrSessionNotFound
	}
	if err != nil {
		return model.Session{}, err
	}
	var s model.Session
	if err := json.Unmarshal(b, &s); err != nil {
		return model.Session{}, err
	}
	return s, nil
}

// Touch продлевает TTL сессии до полного значения (sliding expiration).
func (r *SessionRepo) Touch(ctx context.Context, id string) error {
	return r.rdb.Expire(ctx, key(id), r.ttl).Err()
}

// Delete удаляет сессию (logout).
func (r *SessionRepo) Delete(ctx context.Context, id string) error {
	return r.rdb.Del(ctx, key(id)).Err()
}

func key(id string) string { return "session:" + id }
