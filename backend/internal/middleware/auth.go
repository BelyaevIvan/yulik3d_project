package middleware

import (
	"log/slog"
	"net/http"

	"yulik3d/internal/model"
	"yulik3d/pkg/cookie"
)

// GetSessionFunc — минимальный контракт сервиса для middleware. Передаётся
// из main.go как замыкание над AuthService.GetSession.
type GetSessionFunc func(r *http.Request, id string) (model.Session, error)

// RequireAuth — middleware, которая читает cookie, грузит сессию и кладёт
// user в ctx. Без сессии — 401.
func RequireAuth(cookieName string, getSession GetSessionFunc, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sid := cookie.Read(r, cookieName)
			if sid == "" {
				WriteError(w, r, model.NewUnauthenticated("Требуется авторизация"), log)
				return
			}
			sess, err := getSession(r, sid)
			if err != nil {
				WriteError(w, r, err, log)
				return
			}
			ctx := WithUser(r.Context(), UserCtx{
				ID:       sess.UserID,
				Role:     sess.Role,
				FullName: sess.FullName,
			})
			ctx = WithSessionID(ctx, sid)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole — гарантирует, что в ctx есть user и его роль совпадает.
// Должна использоваться ПОСЛЕ RequireAuth.
func RequireRole(role model.Role, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, ok := UserFromCtx(r.Context())
			if !ok {
				WriteError(w, r, model.NewUnauthenticated("Требуется авторизация"), log)
				return
			}
			if u.Role != role {
				WriteError(w, r, model.NewForbidden("Нет доступа"), log)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RejectAuthed — для endpoints, которые не должны вызываться уже залогиненным
// (register/login). Возвращает 409 если сессия активна.
func RejectAuthed(cookieName string, getSession GetSessionFunc, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if sid := cookie.Read(r, cookieName); sid != "" {
				if _, err := getSession(r, sid); err == nil {
					WriteError(w, r, model.NewConflict("Вы уже авторизованы"), log)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
