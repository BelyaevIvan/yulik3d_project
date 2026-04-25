// Package cookie — helper для установки/очистки сессионного cookie.
package cookie

import (
	"net/http"
	"time"
)

// Options — параметры cookie. Загружаются из конфига один раз.
type Options struct {
	Name     string
	Domain   string
	Secure   bool
	SameSite http.SameSite
	MaxAge   time.Duration
	Path     string
}

// Set ставит сессионный cookie в ResponseWriter.
func Set(w http.ResponseWriter, opts Options, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     opts.Name,
		Value:    sessionID,
		Path:     defPath(opts.Path),
		Domain:   opts.Domain,
		MaxAge:   int(opts.MaxAge.Seconds()),
		Secure:   opts.Secure,
		HttpOnly: true,
		SameSite: defSameSite(opts.SameSite),
	})
}

// Clear сбрасывает сессионный cookie (MaxAge=-1 → удаление в браузере).
func Clear(w http.ResponseWriter, opts Options) {
	http.SetCookie(w, &http.Cookie{
		Name:     opts.Name,
		Value:    "",
		Path:     defPath(opts.Path),
		Domain:   opts.Domain,
		MaxAge:   -1,
		Secure:   opts.Secure,
		HttpOnly: true,
		SameSite: defSameSite(opts.SameSite),
	})
}

// Read возвращает значение cookie или пустую строку, если нет.
func Read(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}

func defPath(p string) string {
	if p == "" {
		return "/"
	}
	return p
}

func defSameSite(s http.SameSite) http.SameSite {
	if s == 0 {
		return http.SameSiteLaxMode
	}
	return s
}
