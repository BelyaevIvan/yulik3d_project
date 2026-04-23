package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"yulik3d/internal/model"
)

// Recover — ловит панику, логирует stack trace, отдаёт 500.
func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					if log != nil {
						log.Error("panic",
							"request_id", RequestIDFromCtx(r.Context()),
							"path", r.URL.Path,
							"err", fmt.Sprintf("%v", rec),
							"stack", string(debug.Stack()),
						)
					}
					WriteError(w, r, fmt.Errorf("%w: panic", model.ErrUnavailable), log)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
