package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"yulik3d/internal/model"
	"yulik3d/internal/repository"
	"yulik3d/pkg/passwordhash"
)

// AuthMailer — интерфейс отправки писем, нужных для AuthService:
// сброса пароля и подтверждения email. Реализуется на уровне очереди
// (queue.Client). Сервис о asynq не знает.
type AuthMailer interface {
	EnqueuePasswordReset(ctx context.Context, to, userName, resetLink string) error
	EnqueueEmailVerify(ctx context.Context, to, userName, verifyLink string) error
}

// AuthService — регистрация, логин, logout, профиль, восстановление пароля,
// подтверждение email.
type AuthService struct {
	users         *repository.UserRepo
	sessions      *repository.SessionRepo
	rate          *repository.RateLimitRepo
	pwReset       *repository.PasswordResetRepo
	emailVerify   *repository.EmailVerifyRepo
	mailer        AuthMailer
	frontendURL   string
	argonParams   passwordhash.Params
	sessionTTL    time.Duration
	rateAttempt   int
	rateWindow    time.Duration
}

func NewAuthService(
	users *repository.UserRepo,
	sessions *repository.SessionRepo,
	rate *repository.RateLimitRepo,
	pwReset *repository.PasswordResetRepo,
	emailVerify *repository.EmailVerifyRepo,
	mailer AuthMailer,
	frontendURL string,
	argonParams passwordhash.Params,
	sessionTTL time.Duration,
	rateAttempts int,
	rateWindow time.Duration,
) *AuthService {
	return &AuthService{
		users: users, sessions: sessions, rate: rate,
		pwReset: pwReset, emailVerify: emailVerify, mailer: mailer,
		frontendURL: strings.TrimRight(frontendURL, "/"),
		argonParams: argonParams, sessionTTL: sessionTTL,
		rateAttempt: rateAttempts, rateWindow: rateWindow,
	}
}

// SessionInfo — результат создания сессии.
type SessionInfo struct {
	ID      string
	Session model.Session
	User    model.User
}

// Register — создать юзера + залогинить. Возвращает сессию для установки cookie.
func (s *AuthService) Register(ctx context.Context, req model.RegisterRequest, ua, ip string) (SessionInfo, error) {
	// Формальная валидация — сервисный слой
	req.Email = normalizeEmail(req.Email)
	if err := validateRegister(req); err != nil {
		return SessionInfo{}, err
	}

	exists, err := s.users.EmailExists(ctx, req.Email)
	if err != nil {
		return SessionInfo{}, fmt.Errorf("check email: %w", err)
	}
	if exists {
		return SessionInfo{}, model.NewConflict("Пользователь с таким email уже зарегистрирован")
	}

	hash, err := passwordhash.Hash(req.Password, s.argonParams)
	if err != nil {
		return SessionInfo{}, fmt.Errorf("hash password: %w", err)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return SessionInfo{}, fmt.Errorf("uuid: %w", err)
	}
	u := model.User{
		ID:           id,
		Email:        req.Email,
		PasswordHash: hash,
		FullName:     strings.TrimSpace(req.FullName),
		Phone:        req.Phone,
		Role:         model.RoleUser,
	}
	if err := s.users.Create(ctx, &u); err != nil {
		return SessionInfo{}, fmt.Errorf("create user: %w", err)
	}

	// Открываем сессию (юзер сразу залогинен — это базовое поведение,
	// от которого зависит весь остальной фронт). Это происходит ДО любой
	// работы с email-подтверждением — никакая ошибка с письмом не повлияет
	// на саму авторизацию.
	info, err := s.openSession(ctx, u, ua, ip)
	if err != nil {
		return SessionInfo{}, err
	}

	// Параллельно (best-effort) ставим письмо подтверждения email в очередь.
	// Любые сбои здесь только логируются — регистрация уже успешна, юзер
	// может позже запросить ссылку повторно через resend.
	s.sendEmailVerify(ctx, u)

	return info, nil
}

// sendEmailVerify — генерация токена подтверждения и постановка письма в
// очередь. Все ошибки проглатываются (только лог) — это не должно ломать
// бизнес-операции, в рамках которых вызывается.
//
// Используется в Register и в EmailVerifyResend.
func (s *AuthService) sendEmailVerify(ctx context.Context, u model.User) {
	token, err := s.emailVerify.CreateToken(ctx, u.ID)
	if err != nil {
		// Не критично — пользователь сможет запросить ещё раз через resend.
		// (Глобального логгера в сервисе нет, см. PasswordResetRequest где log
		// пробрасывается явно. Здесь сбой Redis тут будет и в throttle тоже
		// и попадёт в общие логи приложения через хендлеры.)
		return
	}
	verifyLink := s.frontendURL + "/verify-email?token=" + token
	_ = s.mailer.EnqueueEmailVerify(ctx, u.Email, u.FullName, verifyLink)
}

// Login — вход. Учитывает rate-limit.
func (s *AuthService) Login(ctx context.Context, req model.LoginRequest, ua, ip string) (SessionInfo, error) {
	req.Email = normalizeEmail(req.Email)
	if req.Email == "" || req.Password == "" {
		return SessionInfo{}, model.NewInvalidInput("Укажите email и пароль")
	}

	// Rate limit — fail closed если Redis недоступен.
	if err := s.checkRateLimit(ctx, ip, req.Email); err != nil {
		return SessionInfo{}, err
	}

	u, err := s.users.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SessionInfo{}, model.NewUnauthenticated("Неверный email или пароль")
		}
		return SessionInfo{}, fmt.Errorf("get user: %w", err)
	}
	if err := passwordhash.Verify(req.Password, u.PasswordHash); err != nil {
		if errors.Is(err, passwordhash.ErrMismatch) {
			return SessionInfo{}, model.NewUnauthenticated("Неверный email или пароль")
		}
		return SessionInfo{}, fmt.Errorf("verify password: %w", err)
	}

	// На успехе — сбросить счётчики для email (IP трогать не будем, чтобы один
	// успешный логин не обнулил атаки с того же IP на другие аккаунты).
	_ = s.rate.Reset(ctx, rateKeyEmail(req.Email))

	return s.openSession(ctx, u, ua, ip)
}

// Logout — удалить сессию.
func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	return s.sessions.Delete(ctx, sessionID)
}

// GetSession — возвращает сессию из Redis. Middleware зовёт.
func (s *AuthService) GetSession(ctx context.Context, sessionID string) (model.Session, error) {
	sess, err := s.sessions.Get(ctx, sessionID)
	if err != nil {
		if errors.Is(err, repository.ErrSessionNotFound) {
			return model.Session{}, model.NewUnauthenticated("Сессия не найдена или истекла")
		}
		return model.Session{}, fmt.Errorf("session get: %w", err)
	}
	// Sliding: продлить TTL если прошло > 50%.
	if time.Until(sess.ExpiresAt) < s.sessionTTL/2 {
		_ = s.sessions.Touch(ctx, sessionID)
	}
	return sess, nil
}

// GetMe — полный профиль текущего пользователя.
func (s *AuthService) GetMe(ctx context.Context, userID uuid.UUID) (model.UserDTO, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.UserDTO{}, model.NewNotFound("Пользователь не найден")
		}
		return model.UserDTO{}, fmt.Errorf("get user: %w", err)
	}
	return u.ToDTO(), nil
}

// UpdateMe — редактирование full_name / phone + опциональная смена пароля.
// Для смены пароля передаются оба поля — old_password (проверяется) и new_password.
func (s *AuthService) UpdateMe(ctx context.Context, userID uuid.UUID, req model.UpdateMeRequest) (model.UserDTO, error) {
	if req.FullName == nil && req.Phone == nil && req.NewPassword == nil && req.OldPassword == nil {
		return model.UserDTO{}, model.NewInvalidInput("Нет полей для обновления")
	}
	if (req.OldPassword == nil) != (req.NewPassword == nil) {
		return model.UserDTO{}, model.NewInvalidInput("Для смены пароля нужно передать и старый, и новый пароль")
	}
	if req.FullName != nil {
		v := strings.TrimSpace(*req.FullName)
		if v == "" {
			return model.UserDTO{}, model.NewInvalidInput("Полное имя не может быть пустым")
		}
		if len(v) > 200 {
			return model.UserDTO{}, model.NewInvalidInput("Полное имя слишком длинное (макс. 200 символов)")
		}
		req.FullName = &v
	}

	// Смена пароля — проверка старого + хэширование нового
	var newHash *string
	if req.NewPassword != nil {
		if len(*req.NewPassword) < 8 {
			return model.UserDTO{}, model.NewInvalidInput("Новый пароль должен быть не короче 8 символов")
		}
		if len(*req.NewPassword) > 128 {
			return model.UserDTO{}, model.NewInvalidInput("Новый пароль слишком длинный (макс. 128 символов)")
		}
		cur, err := s.users.GetByID(ctx, userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return model.UserDTO{}, model.NewNotFound("Пользователь не найден")
			}
			return model.UserDTO{}, fmt.Errorf("get user: %w", err)
		}
		if err := passwordhash.Verify(*req.OldPassword, cur.PasswordHash); err != nil {
			if errors.Is(err, passwordhash.ErrMismatch) {
				return model.UserDTO{}, model.NewUnauthenticated("Неверный старый пароль")
			}
			return model.UserDTO{}, fmt.Errorf("verify password: %w", err)
		}
		hash, err := passwordhash.Hash(*req.NewPassword, s.argonParams)
		if err != nil {
			return model.UserDTO{}, fmt.Errorf("hash password: %w", err)
		}
		newHash = &hash
	}

	u, err := s.users.UpdateProfile(ctx, userID, req.FullName, req.Phone, newHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.UserDTO{}, model.NewNotFound("Пользователь не найден")
		}
		return model.UserDTO{}, fmt.Errorf("update profile: %w", err)
	}
	return u.ToDTO(), nil
}

// ---------- Восстановление пароля ----------

// PasswordResetRequest — пользователь запрашивает ссылку на сброс пароля.
//
// Контракт:
//   - Всегда возвращает nil (не палит существование email — защита от перебора).
//   - Если email найден И throttle прошёл → создаёт токен и кладёт письмо в очередь.
//   - Иначе — тихо игнорирует.
//   - Внутренние ошибки (Redis недоступен, БД недоступна) логируются, наружу не пробрасываются.
func (s *AuthService) PasswordResetRequest(ctx context.Context, log *slog.Logger, email string) error {
	email = normalizeEmail(email)
	if email == "" || !strings.Contains(email, "@") {
		return nil // некорректный email — тихо игнор, чтобы не дать сигнал атакующему
	}

	// Throttle — не более одного запроса в TTL для одного email.
	ok, err := s.pwReset.AcquireThrottle(ctx, email)
	if err != nil {
		log.Error("pwreset throttle", "err", err, "email", email)
		return nil
	}
	if !ok {
		log.Info("pwreset throttled", "email", email)
		return nil
	}

	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Info("pwreset: user not found", "email", email)
			return nil
		}
		log.Error("pwreset get user", "err", err)
		return nil
	}

	token, err := s.pwReset.CreateToken(ctx, u.ID)
	if err != nil {
		log.Error("pwreset create token", "err", err, "user_id", u.ID)
		return nil
	}

	resetLink := s.frontendURL + "/password-reset?token=" + token
	if err := s.mailer.EnqueuePasswordReset(ctx, u.Email, u.FullName, resetLink); err != nil {
		log.Error("pwreset enqueue", "err", err, "user_id", u.ID)
		// токен уже создан — пользователь может попробовать ещё раз через минуту;
		// оставляем нерабочий токен в Redis на TTL, ничего страшного
	}
	return nil
}

// PasswordResetConfirm — установить новый пароль по токену.
func (s *AuthService) PasswordResetConfirm(ctx context.Context, token, newPassword string) error {
	if strings.TrimSpace(token) == "" {
		return model.NewInvalidInput("Токен не указан")
	}
	if len(newPassword) < 8 {
		return model.NewInvalidInput("Пароль должен быть не короче 8 символов")
	}
	if len(newPassword) > 128 {
		return model.NewInvalidInput("Пароль слишком длинный (макс. 128 символов)")
	}
	userID, err := s.pwReset.ConsumeToken(ctx, token)
	if err != nil {
		if errors.Is(err, repository.ErrTokenInvalid) {
			return model.NewInvalidInput("Ссылка недействительна или срок её действия истёк")
		}
		return fmt.Errorf("consume token: %w", err)
	}
	hash, err := passwordhash.Hash(newPassword, s.argonParams)
	if err != nil {
		return fmt.Errorf("hash: %w", err)
	}
	if _, err := s.users.UpdateProfile(ctx, userID, nil, nil, &hash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.NewNotFound("Пользователь не найден")
		}
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

// ---------- Подтверждение email ----------

// EmailVerifyConfirm — установить email_verified=true по токену.
// Токен инвалидируется атомарно (одноразовое использование).
//
// Эндпоинт публичный: пользователь мог быть разлогинен в момент клика по
// ссылке. Сама валидация — только по токену, к сессии не привязана.
func (s *AuthService) EmailVerifyConfirm(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return model.NewInvalidInput("Токен не указан")
	}
	userID, err := s.emailVerify.ConsumeToken(ctx, token)
	if err != nil {
		if errors.Is(err, repository.ErrEmailVerifyTokenInvalid) {
			return model.NewInvalidInput("Ссылка недействительна или срок её действия истёк")
		}
		return fmt.Errorf("consume email verify token: %w", err)
	}
	if err := s.users.SetEmailVerified(ctx, userID); err != nil {
		return fmt.Errorf("set email verified: %w", err)
	}
	return nil
}

// EmailVerifyResend — отправить пользователю новое письмо подтверждения.
//
// Контракт идентичен PasswordResetRequest:
//   - Всегда возвращает nil (не палим существование email)
//   - Throttle 60 сек на email (защита от перебора)
//   - Если email уже подтверждён — тихо игнорируем (письма не шлём)
//   - Внутренние ошибки логируются, наружу не пробрасываются
func (s *AuthService) EmailVerifyResend(ctx context.Context, log *slog.Logger, email string) error {
	email = normalizeEmail(email)
	if email == "" || !strings.Contains(email, "@") {
		return nil
	}
	ok, err := s.emailVerify.AcquireThrottle(ctx, email)
	if err != nil {
		log.Error("emailverify throttle", "err", err, "email", email)
		return nil
	}
	if !ok {
		log.Info("emailverify throttled", "email", email)
		return nil
	}
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Info("emailverify: user not found", "email", email)
			return nil
		}
		log.Error("emailverify get user", "err", err)
		return nil
	}
	if u.EmailVerified {
		log.Info("emailverify: already verified", "email", email)
		return nil
	}
	token, err := s.emailVerify.CreateToken(ctx, u.ID)
	if err != nil {
		log.Error("emailverify create token", "err", err, "user_id", u.ID)
		return nil
	}
	verifyLink := s.frontendURL + "/verify-email?token=" + token
	if err := s.mailer.EnqueueEmailVerify(ctx, u.Email, u.FullName, verifyLink); err != nil {
		log.Error("emailverify enqueue", "err", err, "user_id", u.ID)
	}
	return nil
}

// ---------- helpers ----------

func (s *AuthService) openSession(ctx context.Context, u model.User, ua, ip string) (SessionInfo, error) {
	sid, err := newSessionID()
	if err != nil {
		return SessionInfo{}, fmt.Errorf("session id: %w", err)
	}
	now := time.Now().UTC()
	sess := model.Session{
		UserID:    u.ID,
		Role:      u.Role,
		FullName:  u.FullName,
		CreatedAt: now,
		ExpiresAt: now.Add(s.sessionTTL),
		UserAgent: ua,
		IP:        ip,
	}
	if err := s.sessions.Create(ctx, sid, sess); err != nil {
		return SessionInfo{}, fmt.Errorf("session create: %w", err)
	}
	return SessionInfo{ID: sid, Session: sess, User: u}, nil
}

func (s *AuthService) checkRateLimit(ctx context.Context, ip, email string) error {
	// email счётчик
	n, err := s.rate.Incr(ctx, rateKeyEmail(email), s.rateWindow)
	if err != nil {
		return fmt.Errorf("rate email: %w", err)
	}
	if int(n) > s.rateAttempt {
		return model.NewRateLimited("Слишком много попыток входа — попробуйте позже")
	}
	if ip != "" {
		n, err := s.rate.Incr(ctx, rateKeyIP(ip), s.rateWindow)
		if err != nil {
			return fmt.Errorf("rate ip: %w", err)
		}
		if int(n) > s.rateAttempt*4 { // per-ip шире, чтобы не блокировать NAT
			return model.NewRateLimited("Слишком много попыток входа с этого IP")
		}
	}
	return nil
}

func rateKeyEmail(email string) string { return "rl:login:email:" + email }
func rateKeyIP(ip string) string        { return "rl:login:ip:" + ip }

func newSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func validateRegister(r model.RegisterRequest) error {
	if r.Email == "" || !strings.Contains(r.Email, "@") {
		return model.NewInvalidInput("Некорректный email")
	}
	if len(r.Password) < 8 {
		return model.NewInvalidInput("Пароль должен быть не короче 8 символов")
	}
	if len(r.Password) > 128 {
		return model.NewInvalidInput("Пароль слишком длинный (макс. 128 символов)")
	}
	fn := strings.TrimSpace(r.FullName)
	if fn == "" {
		return model.NewInvalidInput("Укажите полное имя")
	}
	if len(fn) > 200 {
		return model.NewInvalidInput("Полное имя слишком длинное (макс. 200 символов)")
	}
	return nil
}
