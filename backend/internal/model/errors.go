// Package model — доменные типы, DTO и ошибки. Не зависит от других слоёв.
package model

import (
	"errors"
	"fmt"
)

// Базовые классы ошибок. Все ожидаемые ошибки сервисного слоя — производные
// от них через errors.Is. Middleware мапит класс в HTTP-статус.
var (
	ErrNotFound         = errors.New("not found")
	ErrConflict         = errors.New("conflict")
	ErrInvalidInput     = errors.New("invalid input")
	ErrUnauthenticated  = errors.New("unauthenticated")
	ErrForbidden        = errors.New("forbidden")
	ErrRateLimited      = errors.New("rate limited")
	ErrUnavailable      = errors.New("service unavailable")
)

// AppError — ошибка с сообщением для клиента. Оборачивает базовую ErrXxx.
type AppError struct {
	Kind    error  // ErrNotFound, ErrConflict и т.п.
	Message string // сообщение для клиента
	Cause   error  // исходная ошибка (может быть nil)
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Kind }

// Обёртки для удобного создания.

func NewNotFound(msg string) error {
	return &AppError{Kind: ErrNotFound, Message: msg}
}

func NewConflict(msg string) error {
	return &AppError{Kind: ErrConflict, Message: msg}
}

func NewInvalidInput(msg string) error {
	return &AppError{Kind: ErrInvalidInput, Message: msg}
}

func NewUnauthenticated(msg string) error {
	return &AppError{Kind: ErrUnauthenticated, Message: msg}
}

func NewForbidden(msg string) error {
	return &AppError{Kind: ErrForbidden, Message: msg}
}

func NewRateLimited(msg string) error {
	return &AppError{Kind: ErrRateLimited, Message: msg}
}

// WithCause добавляет исходную ошибку к AppError (для логов).
func WithCause(err, cause error) error {
	if ae, ok := err.(*AppError); ok {
		ae.Cause = cause
		return ae
	}
	return fmt.Errorf("%w: %v", err, cause)
}

// MessageOf извлекает клиентское сообщение из ошибки.
// Для AppError → e.Message, иначе — универсальное.
func MessageOf(err error) string {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae.Message
	}
	return "Внутренняя ошибка сервера"
}
