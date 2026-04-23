package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// New создаёт JSON-slog с указанным уровнем. Используется как единый логгер
// приложения, инжектится в конструкторы сервисов/хэндлеров.
func New(level string, out io.Writer) *slog.Logger {
	if out == nil {
		out = os.Stdout
	}
	var lvl slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{
		Level:     lvl,
		AddSource: false,
	}))
}
