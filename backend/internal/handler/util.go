// Package handler — HTTP-хэндлеры. Слой приёма-отправки запросов.
package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"yulik3d/internal/middleware"
	"yulik3d/internal/model"
)

// Deps — общие зависимости: логгер. Остальное инжектится конкретно.
type Deps struct {
	Log *slog.Logger
}

// ParseUUIDPath извлекает uuid из path-параметра.
func ParseUUIDPath(r *http.Request, name string) (uuid.UUID, error) {
	v := r.PathValue(name)
	if v == "" {
		return uuid.Nil, model.NewInvalidInput("Отсутствует path-параметр: " + name)
	}
	id, err := uuid.Parse(v)
	if err != nil {
		return uuid.Nil, model.NewInvalidInput("Некорректный UUID")
	}
	return id, nil
}

// ParsePagination читает limit/offset из query.
func ParsePagination(r *http.Request) model.Pagination {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	return model.Pagination{Limit: limit, Offset: offset}
}

// DecodeJSON — читает тело в v. На ошибку возвращает model.ErrInvalidInput.
func DecodeJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return model.NewInvalidInput("Пустое тело запроса")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return model.NewInvalidInput("Некорректный JSON: " + err.Error())
	}
	return nil
}

// OK/Created — короткие обёртки.
func OK(w http.ResponseWriter, v any)      { middleware.WriteJSON(w, http.StatusOK, v) }
func Created(w http.ResponseWriter, v any) { middleware.WriteJSON(w, http.StatusCreated, v) }
func NoContent(w http.ResponseWriter)      { w.WriteHeader(http.StatusNoContent) }

// Err — пишет ошибку в ответ (через middleware).
func (d *Deps) Err(w http.ResponseWriter, r *http.Request, err error) {
	middleware.WriteError(w, r, err, d.Log)
}

// ClientIP — пытается достать настоящий IP (учёт X-Forwarded-For).
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return xr
	}
	// Убираем :port
	host := r.RemoteAddr
	if i := strings.LastIndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	return host
}

// MustUser — достаёт user из context; панику не бросает, выставляет 401.
// Использовать только в хэндлерах ЗА middleware.RequireAuth.
func (d *Deps) MustUser(w http.ResponseWriter, r *http.Request) (middleware.UserCtx, bool) {
	u, ok := middleware.UserFromCtx(r.Context())
	if !ok {
		middleware.WriteError(w, r, model.NewUnauthenticated("Требуется авторизация"), d.Log)
		return middleware.UserCtx{}, false
	}
	return u, true
}
