package repository

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// EmailVerifyRepo — токены подтверждения email в Redis.
//
// Ключи (живут в той же DB 0, что сессии и pwreset, но с собственным префиксом):
//
//	emailverify:<token>           → user_id, TTL = TokenTTL (24h)
//	emailverify:throttle:<email>  → "1",     TTL = Throttle (60s)
//
// GETDEL обеспечивает атомарную «прочитать-и-инвалидировать»: токен нельзя
// использовать дважды. С сессией пользователя (ключ session:<id>) пересечений
// нет — другой префикс, другой формат значения.
type EmailVerifyRepo struct {
	rdb      *redis.Client
	tokenTTL time.Duration
	throttle time.Duration
}

func NewEmailVerifyRepo(rdb *redis.Client, tokenTTL, throttle time.Duration) *EmailVerifyRepo {
	return &EmailVerifyRepo{rdb: rdb, tokenTTL: tokenTTL, throttle: throttle}
}

// ErrEmailVerifyTokenInvalid — возвращается, когда токен не найден,
// уже использован или истёк.
var ErrEmailVerifyTokenInvalid = errors.New("email verify token invalid or expired")

// CreateToken — генерирует криптостойкий токен и записывает в Redis.
// Возвращает сам токен для отправки в письме.
func (r *EmailVerifyRepo) CreateToken(ctx context.Context, userID uuid.UUID) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	if err := r.rdb.Set(ctx, "emailverify:"+token, userID.String(), r.tokenTTL).Err(); err != nil {
		return "", fmt.Errorf("redis set: %w", err)
	}
	return token, nil
}

// ConsumeToken — атомарно прочитать и удалить токен. Возвращает userID или ErrEmailVerifyTokenInvalid.
func (r *EmailVerifyRepo) ConsumeToken(ctx context.Context, token string) (uuid.UUID, error) {
	val, err := r.rdb.GetDel(ctx, "emailverify:"+token).Result()
	if errors.Is(err, redis.Nil) {
		return uuid.Nil, ErrEmailVerifyTokenInvalid
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("redis getdel: %w", err)
	}
	id, err := uuid.Parse(val)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse uuid from token: %w", err)
	}
	return id, nil
}

// AcquireThrottle — true, если throttle для email сейчас не активен (можно слать).
// SET NX EX: если ключ уже есть → false.
func (r *EmailVerifyRepo) AcquireThrottle(ctx context.Context, email string) (bool, error) {
	ok, err := r.rdb.SetNX(ctx, "emailverify:throttle:"+email, "1", r.throttle).Result()
	if err != nil {
		return false, fmt.Errorf("redis setnx: %w", err)
	}
	return ok, nil
}
