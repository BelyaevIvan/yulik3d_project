package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// statusRecorder — оборачиваем w, чтобы поймать статус.
type statusRecorder struct {
	http.ResponseWriter
	status int
	size   int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Write(b []byte) (int, error) {
	if sr.status == 0 {
		sr.status = http.StatusOK
	}
	n, err := sr.ResponseWriter.Write(b)
	sr.size += n
	return n, err
}

// Logging — access-log + request-id в context.
func Logging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rid := r.Header.Get("X-Request-ID")
			if rid == "" {
				rid = uuid.NewString()
			}
			w.Header().Set("X-Request-ID", rid)
			ctx := WithRequestID(r.Context(), rid)
			r = r.WithContext(ctx)

			sr := &statusRecorder{ResponseWriter: w}
			start := time.Now()
			next.ServeHTTP(sr, r)
			dur := time.Since(start)

			if log != nil {
				log.Info("http",
					"request_id", rid,
					"method", r.Method,
					"path", r.URL.Path,
					"status", sr.status,
					"size", sr.size,
					"duration_ms", dur.Milliseconds(),
					"remote", r.RemoteAddr,
				)
			}
		})
	}
}
