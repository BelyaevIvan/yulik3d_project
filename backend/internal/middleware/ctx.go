// Package middleware — HTTP-middleware для сервиса.
package middleware

import (
	"context"

	"github.com/google/uuid"

	"yulik3d/internal/model"
)

// Ключи для context. Приватные типы — чтобы никто извне не подменил.
type ctxKey int

const (
	ctxUser ctxKey = iota
	ctxSessionID
	ctxRequestID
)

// UserCtx — данные сессии, которые middleware.Auth кладёт в context.
type UserCtx struct {
	ID       uuid.UUID
	Role     model.Role
	FullName string
}

// WithUser — положить в context.
func WithUser(ctx context.Context, u UserCtx) context.Context {
	return context.WithValue(ctx, ctxUser, u)
}

// UserFromCtx — достать из context (и флаг наличия).
func UserFromCtx(ctx context.Context) (UserCtx, bool) {
	v, ok := ctx.Value(ctxUser).(UserCtx)
	return v, ok
}

// WithSessionID / SessionIDFromCtx — чтобы logout-хэндлер мог получить ID для Delete.
func WithSessionID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxSessionID, id)
}

func SessionIDFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(ctxSessionID).(string); ok {
		return v
	}
	return ""
}

// WithRequestID / RequestIDFromCtx.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxRequestID, id)
}

func RequestIDFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(ctxRequestID).(string); ok {
		return v
	}
	return ""
}
