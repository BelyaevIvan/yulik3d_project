package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
)

// RedisOpt — параметры подключения к Redis (DB для очереди).
type RedisOpt struct {
	Addr     string
	Password string
	DB       int
}

func (o RedisOpt) toAsynq() asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Addr:     o.Addr,
		Password: o.Password,
		DB:       o.DB,
	}
}

// Client — публичная точка постановки задач из сервисов.
type Client struct {
	asynq *asynq.Client
	log   *slog.Logger
}

// NewClient — создать клиент очереди (для Enqueue).
func NewClient(opt RedisOpt, log *slog.Logger) *Client {
	return &Client{
		asynq: asynq.NewClient(opt.toAsynq()),
		log:   log,
	}
}

// Close — закрыть соединение с Redis.
func (c *Client) Close() error { return c.asynq.Close() }

// EnqueueEmail — поставить email-задачу в очередь.
// Не блокирующая: ошибка лишь логируется, основной флоу не страдает.
// Возвращает ошибку для случаев, когда вызывающему важно знать (тесты).
func (c *Client) EnqueueEmail(ctx context.Context, taskType string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		c.log.Error("queue: marshal payload", "type", taskType, "err", err)
		return fmt.Errorf("marshal: %w", err)
	}
	t := asynq.NewTask(taskType, data,
		asynq.MaxRetry(5),
		asynq.Timeout(45*time.Second),
		asynq.Retention(24*time.Hour), // хранить успешные задачи сутки для отладки
	)
	if _, err := c.asynq.EnqueueContext(ctx, t); err != nil {
		c.log.Error("queue: enqueue", "type", taskType, "err", err)
		return fmt.Errorf("enqueue: %w", err)
	}
	return nil
}

// Server — обёртка над asynq.Server, чтобы скрыть подробности от main.go.
type Server struct {
	srv *asynq.Server
	mux *asynq.ServeMux
	log *slog.Logger
}

// NewServer — создать сервер-воркер. Регистрация хендлеров через RegisterHandlers.
func NewServer(opt RedisOpt, log *slog.Logger) *Server {
	srv := asynq.NewServer(opt.toAsynq(), asynq.Config{
		Concurrency: 5,
		Logger:      newAsynqLogger(log),
		// Экспоненциальный backoff: 30s, 1m, 4m, 16m, 1h.
		RetryDelayFunc: func(n int, _ error, _ *asynq.Task) time.Duration {
			delays := []time.Duration{30 * time.Second, 1 * time.Minute, 4 * time.Minute, 16 * time.Minute, 1 * time.Hour}
			if n >= len(delays) {
				return delays[len(delays)-1]
			}
			return delays[n]
		},
	})
	return &Server{srv: srv, mux: asynq.NewServeMux(), log: log}
}

// Start — запускает воркер в фоне. Block=false: вызывающий контролирует жизненный цикл.
func (s *Server) Start() error {
	if err := s.srv.Start(s.mux); err != nil {
		return fmt.Errorf("asynq start: %w", err)
	}
	s.log.Info("asynq worker started")
	return nil
}

// Shutdown — graceful stop с ожиданием активных задач.
func (s *Server) Shutdown() {
	s.srv.Shutdown()
	s.log.Info("asynq worker stopped")
}

// asynqLogger — адаптер slog → asynq.Logger.
type asynqLogger struct{ log *slog.Logger }

func newAsynqLogger(log *slog.Logger) *asynqLogger { return &asynqLogger{log: log} }

func (l *asynqLogger) Debug(args ...any) { l.log.Debug(fmtArgs(args)) }
func (l *asynqLogger) Info(args ...any)  { l.log.Info(fmtArgs(args)) }
func (l *asynqLogger) Warn(args ...any)  { l.log.Warn(fmtArgs(args)) }
func (l *asynqLogger) Error(args ...any) { l.log.Error(fmtArgs(args)) }
func (l *asynqLogger) Fatal(args ...any) { l.log.Error("FATAL: " + fmtArgs(args)) }

func fmtArgs(args []any) string {
	return fmt.Sprint(args...)
}
