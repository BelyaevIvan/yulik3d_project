package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimitRepo — простой фиксированный счётчик в Redis.
type RateLimitRepo struct {
	rdb *redis.Client
}

func NewRateLimitRepo(rdb *redis.Client) *RateLimitRepo {
	return &RateLimitRepo{rdb: rdb}
}

// Incr инкрементит счётчик, при первом обращении ставит TTL.
// Возвращает текущее значение.
func (r *RateLimitRepo) Incr(ctx context.Context, key string, window time.Duration) (int64, error) {
	n, err := r.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if n == 1 {
		if err := r.rdb.Expire(ctx, key, window).Err(); err != nil {
			return n, err
		}
	}
	return n, nil
}

// Reset удаляет счётчик.
func (r *RateLimitRepo) Reset(ctx context.Context, key string) error {
	return r.rdb.Del(ctx, key).Err()
}
