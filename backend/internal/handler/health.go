package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"yulik3d/internal/model"
)

type HealthHandler struct {
	Deps
	pool *pgxpool.Pool
	rdb  *redis.Client
}

func NewHealthHandler(d Deps, pool *pgxpool.Pool, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{Deps: d, pool: pool, rdb: rdb}
}

// Health godoc
// @Summary      Health check
// @Description  Проверяет доступность Postgres и Redis. Возвращает 200 ok или 503 если одна из зависимостей недоступна.
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]string
// @Failure      503  {object}  model.ErrorResponse
// @Router       /health [get]
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.pool.Ping(ctx); err != nil {
		h.Err(w, r, model.WithCause(model.ErrUnavailable, err))
		return
	}
	if err := h.rdb.Ping(ctx).Err(); err != nil {
		h.Err(w, r, model.WithCause(model.ErrUnavailable, err))
		return
	}
	OK(w, map[string]string{"status": "ok"})
}
