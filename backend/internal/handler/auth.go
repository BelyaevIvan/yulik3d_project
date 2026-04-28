package handler

import (
	"net/http"

	"yulik3d/internal/middleware"
	"yulik3d/internal/model"
	"yulik3d/internal/service"
	"yulik3d/pkg/cookie"
)

type AuthHandler struct {
	Deps
	auth       *service.AuthService
	cookieOpts cookie.Options
}

func NewAuthHandler(d Deps, auth *service.AuthService, opts cookie.Options) *AuthHandler {
	return &AuthHandler{Deps: d, auth: auth, cookieOpts: opts}
}

// Register godoc
// @Summary      Регистрация нового пользователя
// @Description  Создаёт пользователя, сразу логинит, ставит session cookie. Гости only.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload  body      model.RegisterRequest  true  "Данные регистрации"
// @Success      201  {object}  model.UserDTO
// @Failure      400  {object}  model.ErrorResponse
// @Failure      409  {object}  model.ErrorResponse
// @Failure      500  {object}  model.ErrorResponse
// @Router       /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.Err(w, r, err)
		return
	}
	info, err := h.auth.Register(r.Context(), req, r.UserAgent(), ClientIP(r))
	if err != nil {
		h.Err(w, r, err)
		return
	}
	cookie.Set(w, h.cookieOpts, info.ID)
	Created(w, info.User.ToDTO())
}

// Login godoc
// @Summary      Вход по email/паролю
// @Description  Создаёт сессию, ставит session cookie. Гости only.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload  body      model.LoginRequest  true  "Логин и пароль"
// @Success      200  {object}  model.UserDTO
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      409  {object}  model.ErrorResponse
// @Failure      429  {object}  model.ErrorResponse
// @Router       /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.Err(w, r, err)
		return
	}
	info, err := h.auth.Login(r.Context(), req, r.UserAgent(), ClientIP(r))
	if err != nil {
		h.Err(w, r, err)
		return
	}
	cookie.Set(w, h.cookieOpts, info.ID)
	OK(w, info.User.ToDTO())
}

// Logout godoc
// @Summary      Выход
// @Description  Удаляет сессию в Redis и чистит cookie.
// @Tags         auth
// @Security     CookieAuth
// @Success      204
// @Failure      401  {object}  model.ErrorResponse
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	sid := middleware.SessionIDFromCtx(r.Context())
	if sid == "" {
		h.Err(w, r, model.NewUnauthenticated("Требуется авторизация"))
		return
	}
	_ = h.auth.Logout(r.Context(), sid)
	cookie.Clear(w, h.cookieOpts)
	NoContent(w)
}

// Me godoc
// @Summary      Текущий пользователь
// @Description  Полный профиль авторизованного пользователя (email, phone и т.д.)
// @Tags         user
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  model.UserDTO
// @Failure      401  {object}  model.ErrorResponse
// @Router       /me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	u, ok := h.MustUser(w, r)
	if !ok {
		return
	}
	dto, err := h.auth.GetMe(r.Context(), u.ID)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, dto)
}

// UpdateMe godoc
// @Summary      Обновить профиль
// @Description  Редактирование full_name / phone и/или смена пароля. Для смены пароля нужно передать оба поля — old_password (проверяется против текущего хэша) и new_password (мин. 8 символов). Можно обновлять поля профиля и пароль в одном запросе.
// @Tags         user
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Param        payload  body  model.UpdateMeRequest  true  "Новые значения профиля и/или пары old_password + new_password"
// @Success      200  {object}  model.UserDTO
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Router       /me [patch]
func (h *AuthHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	u, ok := h.MustUser(w, r)
	if !ok {
		return
	}
	var req model.UpdateMeRequest
	if err := DecodeJSON(r, &req); err != nil {
		h.Err(w, r, err)
		return
	}
	dto, err := h.auth.UpdateMe(r.Context(), u.ID, req)
	if err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, dto)
}

// PasswordResetRequest godoc
// @Summary      Запросить ссылку на восстановление пароля
// @Description  Принимает email. Если пользователь существует — отправляет письмо со ссылкой (TTL 1 час). По соображениям безопасности всегда возвращает 200, даже если email не найден или сработал throttle.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload  body      model.PasswordResetRequestDTO  true  "Email пользователя"
// @Success      200      {object}  model.OKResponse
// @Failure      400      {object}  model.ErrorResponse
// @Router       /auth/password/reset-request [post]
func (h *AuthHandler) PasswordResetRequest(w http.ResponseWriter, r *http.Request) {
	var req model.PasswordResetRequestDTO
	if err := DecodeJSON(r, &req); err != nil {
		h.Err(w, r, err)
		return
	}
	// Сервис всегда возвращает nil — внутренние ошибки логируются, наружу не текут.
	_ = h.auth.PasswordResetRequest(r.Context(), h.Log, req.Email)
	OK(w, model.OKResponse{OK: true})
}

// PasswordResetConfirm godoc
// @Summary      Подтвердить восстановление пароля и установить новый
// @Description  Принимает токен из ссылки в письме и новый пароль. Токен инвалидируется атомарно (одноразовое использование).
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload  body      model.PasswordResetConfirmDTO  true  "Токен и новый пароль"
// @Success      200      {object}  model.OKResponse
// @Failure      400      {object}  model.ErrorResponse
// @Router       /auth/password/reset-confirm [post]
func (h *AuthHandler) PasswordResetConfirm(w http.ResponseWriter, r *http.Request) {
	var req model.PasswordResetConfirmDTO
	if err := DecodeJSON(r, &req); err != nil {
		h.Err(w, r, err)
		return
	}
	if err := h.auth.PasswordResetConfirm(r.Context(), req.Token, req.NewPassword); err != nil {
		h.Err(w, r, err)
		return
	}
	OK(w, model.OKResponse{OK: true})
}
