// Package config загружает конфиг из env. Не использует глобалов.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App           App
	HTTP          HTTP
	Postgres      Postgres
	Redis         Redis
	Session       Session
	RateLimit     RateLimit
	Argon2        Argon2
	MinIO         MinIO
	Uploads       Uploads
	SMTP          SMTP
	Mail          Mail
	PasswordReset PasswordReset
	EmailVerify   EmailVerify
}

type App struct {
	Env               string // development | production
	LogLevel          string
	PublicBackendURL  string
	PublicFrontendURL string
}

type HTTP struct {
	Host           string
	Port           int
	AllowedOrigins []string
}

type Postgres struct {
	Host     string
	Port     int
	DB       string
	User     string
	Password string
	SSLMode  string
}

func (p Postgres) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		p.User, p.Password, p.Host, p.Port, p.DB, p.SSLMode)
}

type Redis struct {
	Host     string
	Port     int
	Password string
	DB       int
	AsynqDB  int // отдельная DB для очереди писем — изолирует от сессий/rate-limit
}

func (r Redis) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type SMTP struct {
	Host     string
	Port     int
	User     string
	Password string
	UseSSL   bool // true для порта 465 (implicit TLS), false для 587 (STARTTLS)
}

type Mail struct {
	FromName    string
	FromEmail   string
	AdminNotify string // куда слать админ-уведомления
	SupportContact string // email для футера писем («По вопросам пишите: ...»)
}

type PasswordReset struct {
	TokenTTL time.Duration
	Throttle time.Duration
}

type EmailVerify struct {
	TokenTTL time.Duration
	Throttle time.Duration
}

type Session struct {
	TTL          time.Duration
	CookieName   string
	CookieDomain string
	CookieSecure bool
}

type RateLimit struct {
	AuthAttempts int
	AuthWindow   time.Duration
}

type Argon2 struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
}

type MinIO struct {
	Host       string
	Port       int
	UseSSL     bool
	RootUser   string
	RootPass   string
	Bucket     string
	PublicURL  string
}

func (m MinIO) Endpoint() string {
	return fmt.Sprintf("%s:%d", m.Host, m.Port)
}

type Uploads struct {
	MaxBytes int64
}

// Load читает конфиг из env. Ошибки собираются в одну для удобства сообщения.
func Load() (Config, error) {
	var errs []string

	getStr := func(key, def string, required bool) string {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			if required {
				errs = append(errs, key+" is required")
			}
			return def
		}
		return v
	}
	getInt := func(key string, def int) int {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			return def
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			errs = append(errs, key+": not a number")
			return def
		}
		return n
	}
	getInt64 := func(key string, def int64) int64 {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			return def
		}
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			errs = append(errs, key+": not a number")
			return def
		}
		return n
	}
	getBool := func(key string, def bool) bool {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			return def
		}
		b, err := strconv.ParseBool(v)
		if err != nil {
			errs = append(errs, key+": not a bool")
			return def
		}
		return b
	}

	cfg := Config{
		App: App{
			Env:               getStr("APP_ENV", "development", false),
			LogLevel:          getStr("LOG_LEVEL", "info", false),
			PublicBackendURL:  getStr("PUBLIC_BACKEND_URL", "http://localhost:8080", false),
			PublicFrontendURL: strings.TrimRight(getStr("PUBLIC_FRONTEND_URL", "http://localhost:5173", false), "/"),
		},
		HTTP: HTTP{
			Host:           getStr("BACKEND_HOST", "0.0.0.0", false),
			Port:           getInt("BACKEND_PORT", 8080),
			AllowedOrigins: splitCSV(getStr("CORS_ALLOWED_ORIGINS", "http://localhost:5173", false)),
		},
		Postgres: Postgres{
			Host:     getStr("POSTGRES_HOST", "", true),
			Port:     getInt("POSTGRES_PORT", 5432),
			DB:       getStr("POSTGRES_DB", "", true),
			User:     getStr("POSTGRES_USER", "", true),
			Password: getStr("POSTGRES_PASSWORD", "", true),
			SSLMode:  getStr("POSTGRES_SSLMODE", "disable", false),
		},
		Redis: Redis{
			Host:     getStr("REDIS_HOST", "", true),
			Port:     getInt("REDIS_PORT", 6379),
			Password: getStr("REDIS_PASSWORD", "", false),
			DB:       getInt("REDIS_DB", 0),
			AsynqDB:  getInt("REDIS_ASYNQ_DB", 1),
		},
		Session: Session{
			TTL:          time.Duration(getInt("SESSION_TTL_SECONDS", 2592000)) * time.Second,
			CookieName:   getStr("COOKIE_NAME", "session", false),
			CookieDomain: os.Getenv("COOKIE_DOMAIN"),
			CookieSecure: getBool("COOKIE_SECURE", false),
		},
		RateLimit: RateLimit{
			AuthAttempts: getInt("RATE_LIMIT_AUTH_ATTEMPTS", 5),
			AuthWindow:   time.Duration(getInt("RATE_LIMIT_AUTH_WINDOW_SECONDS", 900)) * time.Second,
		},
		Argon2: Argon2{
			MemoryKiB:   uint32(getInt("ARGON2_MEMORY_KIB", 65536)),
			Iterations:  uint32(getInt("ARGON2_ITERATIONS", 3)),
			Parallelism: uint8(getInt("ARGON2_PARALLELISM", 2)),
		},
		MinIO: MinIO{
			Host:      getStr("MINIO_HOST", "", true),
			Port:      getInt("MINIO_PORT", 9000),
			UseSSL:    getBool("MINIO_USE_SSL", false),
			RootUser:  getStr("MINIO_ROOT_USER", "", true),
			RootPass:  getStr("MINIO_ROOT_PASSWORD", "", true),
			Bucket:    getStr("MINIO_BUCKET", "", true),
			PublicURL: strings.TrimRight(getStr("MINIO_PUBLIC_URL", "", true), "/"),
		},
		Uploads: Uploads{
			MaxBytes: getInt64("MAX_UPLOAD_BYTES", 10*1024*1024),
		},
		SMTP: SMTP{
			Host:     getStr("SMTP_HOST", "", false),
			Port:     getInt("SMTP_PORT", 465),
			User:     getStr("SMTP_USER", "", false),
			Password: getStr("SMTP_PASS", "", false),
			UseSSL:   getBool("SMTP_USE_SSL", true),
		},
		Mail: Mail{
			FromName:       getStr("SMTP_FROM_NAME", "Yulik3D", false),
			FromEmail:      getStr("SMTP_FROM_EMAIL", "", false),
			AdminNotify:    getStr("ADMIN_NOTIFY_EMAIL", "", false),
			SupportContact: getStr("MAIL_SUPPORT_CONTACT", "", false),
		},
		PasswordReset: PasswordReset{
			TokenTTL: time.Duration(getInt("PWRESET_TOKEN_TTL_SECONDS", 3600)) * time.Second,
			Throttle: time.Duration(getInt("PWRESET_THROTTLE_SECONDS", 60)) * time.Second,
		},
		EmailVerify: EmailVerify{
			TokenTTL: time.Duration(getInt("EMAIL_VERIFY_TOKEN_TTL_SECONDS", 86400)) * time.Second,
			Throttle: time.Duration(getInt("EMAIL_VERIFY_THROTTLE_SECONDS", 60)) * time.Second,
		},
	}

	if len(errs) > 0 {
		return Config{}, errors.New("config: " + strings.Join(errs, "; "))
	}
	return cfg, nil
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
