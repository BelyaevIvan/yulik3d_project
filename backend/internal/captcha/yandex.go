// Package captcha — проверка токенов Yandex SmartCaptcha на стороне сервера.
//
// Принцип работы:
//
//  1. Фронт получает токен из виджета SmartCaptcha (visible-чекбокс).
//  2. Фронт передаёт токен в теле запроса (например, captcha_token в RegisterRequest).
//  3. Бэкенд (через Verifier.Verify) делает HTTP-запрос к Yandex с этим токеном
//     + секретным server-key, получает {status: "ok"} или {status: "failed"}.
//  4. На основе ответа решает, пускать ли дальше.
//
// Поведение при сбое Yandex API настраивается через Mode (см. ниже).
package captcha

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Verifier — интерфейс, который реализует YandexVerifier и (для тестов/dev)
// stubVerifier. Сервисы зависят от интерфейса.
type Verifier interface {
	Verify(ctx context.Context, token, userIP string) error
}

// ErrCaptchaFailed — токен признан невалидным самим Yandex (юзер не прошёл).
var ErrCaptchaFailed = errors.New("captcha verification failed")

// ErrCaptchaUnavailable — Yandex API не отвечает (сетевой сбой / 5xx).
// Сервис должен решить, отказывать или пропускать.
var ErrCaptchaUnavailable = errors.New("captcha service unavailable")

// Mode — что делать при сбое Yandex API.
//
// FailClosed — отказывать пользователю (безопаснее, рекомендуется для prod).
// FailOpen   — пропускать пользователя (удобнее для dev / при отсутствии интернета).
type Mode int

const (
	FailClosed Mode = iota
	FailOpen
)

// YandexVerifier — реальный клиент к https://smartcaptcha.yandexcloud.net/validate.
type YandexVerifier struct {
	endpoint  string
	serverKey string
	mode      Mode
	httpc     *http.Client
	log       *slog.Logger
}

// New создаёт Verifier:
//   - если enabled=false → возвращает stub, всегда OK (для отключения капчи в dev)
//   - если enabled=true и serverKey пустой → стартап-ошибка не делаем,
//     но логируем warn и возвращаем stub: иначе локальная разработка без
//     ключа сразу ляжет
//   - иначе возвращает YandexVerifier с указанным fail-mode
func New(enabled bool, endpoint, serverKey string, mode Mode, log *slog.Logger) Verifier {
	if !enabled {
		log.Info("captcha disabled — all tokens will pass", "reason", "CAPTCHA_ENABLED=false")
		return stubVerifier{always: nil}
	}
	if serverKey == "" {
		log.Warn("captcha enabled but CAPTCHA_SERVER_KEY is empty — all tokens will pass")
		return stubVerifier{always: nil}
	}
	return &YandexVerifier{
		endpoint:  endpoint,
		serverKey: serverKey,
		mode:      mode,
		httpc:     &http.Client{Timeout: 5 * time.Second},
		log:       log,
	}
}

// Verify — отдаёт nil если юзер прошёл капчу.
//   - ErrCaptchaFailed   — токен отвергнут Yandex
//   - ErrCaptchaUnavailable — Yandex недоступен и mode=FailClosed
//   - В режиме FailOpen при сбое возвращает nil (и пишет warn в лог)
//
// Согласно официальной доке (cloud.yandex.ru/docs/smartcaptcha/concepts/validation)
// запрос — POST application/x-www-form-urlencoded. secret/token/ip в теле.
func (v *YandexVerifier) Verify(ctx context.Context, token, userIP string) error {
	if strings.TrimSpace(token) == "" {
		return ErrCaptchaFailed
	}
	form := url.Values{}
	form.Set("secret", v.serverKey)
	form.Set("token", token)
	if userIP != "" {
		form.Set("ip", userIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		v.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return v.handleNetErr("build request", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := v.httpc.Do(req)
	if err != nil {
		return v.handleNetErr("http do", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return v.handleNetErr(fmt.Sprintf("http status %d", resp.StatusCode), nil)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
	if err != nil {
		return v.handleNetErr("read body", err)
	}

	// Ответ Yandex: {"status":"ok","host":"..."} или {"status":"failed","message":"..."}
	// Документация: https://yandex.cloud/ru/docs/smartcaptcha/concepts/validation
	var r struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Host    string `json:"host"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return v.handleNetErr("parse response", err)
	}
	if r.Status != "ok" {
		v.log.Info("captcha rejected by yandex", "status", r.Status, "message", r.Message, "host", r.Host)
		return ErrCaptchaFailed
	}
	return nil
}

// handleNetErr — единая точка для решения «отказывать или пропускать»
// в зависимости от mode.
func (v *YandexVerifier) handleNetErr(stage string, err error) error {
	v.log.Warn("captcha verify network issue", "stage", stage, "err", err, "mode", v.mode)
	if v.mode == FailOpen {
		return nil // пропускаем — для dev
	}
	return ErrCaptchaUnavailable
}

// stubVerifier — заглушка. Всегда возвращает always (по умолчанию nil = пропуск).
type stubVerifier struct{ always error }

func (s stubVerifier) Verify(ctx context.Context, token, userIP string) error {
	return s.always
}
