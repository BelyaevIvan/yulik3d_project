package middleware

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"yulik3d/internal/model"
)

// WriteError — универсальный JSON-ответ с ошибкой. Использует model.ErrorResponse.
func WriteError(w http.ResponseWriter, r *http.Request, err error, log *slog.Logger) {
	status := mapStatus(err)
	msg := model.MessageOf(err)
	if status >= 500 {
		msg = "Внутренняя ошибка сервера"
	}
	resp := model.ErrorResponse{
		StatusCode: status,
		URL:        r.URL.Path,
		Message:    msg,
		Date:       time.Now().UTC().Format(time.RFC3339),
	}

	// Лог: для 5xx — error, для 4xx — warn/info
	if log != nil {
		attrs := []any{
			"status", status,
			"method", r.Method,
			"path", r.URL.Path,
			"request_id", RequestIDFromCtx(r.Context()),
			"err", err.Error(),
		}
		if status >= 500 {
			log.Error("request error", attrs...)
		} else {
			log.Info("request error", attrs...)
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

func mapStatus(err error) int {
	switch {
	case errors.Is(err, model.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, model.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, model.ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, model.ErrUnauthenticated):
		return http.StatusUnauthorized
	case errors.Is(err, model.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, model.ErrRateLimited):
		return http.StatusTooManyRequests
	case errors.Is(err, model.ErrUnavailable):
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// WriteJSON — хелпер успешного ответа.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
