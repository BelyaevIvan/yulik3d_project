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

// PasswordResetRepo — хранит токены ресета пароля в Redis.
//
// Ключи:
//
//	pwreset:<token>           → user_id, TTL = TokenTTL
//	pwreset:throttle:<email>  → "1",     TTL = Throttle (защита от перебора)
//
// GETDEL обеспечивает атомарную «прочитать-и-инвалидировать»: токен нельзя
// использовать дважды.
type PasswordResetRepo struct {
	rdb      *redis.Client
	tokenTTL time.Duration
	throttle time.Duration
}

func NewPasswordResetRepo(rdb *redis.Client, tokenTTL, throttle time.Duration) *PasswordResetRepo {
	return &PasswordResetRepo{rdb: rdb, tokenTTL: tokenTTL, throttle: throttle}
}

// ErrTokenInvalid — возврат, когда токен не найден или уже использован/истёк.
var ErrTokenInvalid = errors.New("password reset token invalid or expired")

// CreateToken — генерирует криптостойкий токен и записывает в Redis.
// Возвращает сам токен для отправки в письме.
func (r *PasswordResetRepo) CreateToken(ctx context.Context, userID uuid.UUID) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	if err := r.rdb.Set(ctx, "pwreset:"+token, userID.String(), r.tokenTTL).Err(); err != nil {
		return "", fmt.Errorf("redis set: %w", err)
	}
	return token, nil
}

// ConsumeToken — атомарно прочитать и удалить токен. Возвращает userID или ErrTokenInvalid.
func (r *PasswordResetRepo) ConsumeToken(ctx context.Context, token string) (uuid.UUID, error) {
	val, err := r.rdb.GetDel(ctx, "pwreset:"+token).Result()
	if errors.Is(err, redis.Nil) {
		return uuid.Nil, ErrTokenInvalid
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

// AcquireThrottle — true, если throttle для этого email сейчас не активен (можно слать).
// Внутри использует SET NX EX: если ключ уже есть → false.
func (r *PasswordResetRepo) AcquireThrottle(ctx context.Context, email string) (bool, error) {
	ok, err := r.rdb.SetNX(ctx, "pwreset:throttle:"+email, "1", r.throttle).Result()
	if err != nil {
		return false, fmt.Errorf("redis setnx: %w", err)
	}
	return ok, nil
}
