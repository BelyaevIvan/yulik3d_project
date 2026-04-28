package model

import (
	"time"

	"github.com/google/uuid"
)

// Role — роль пользователя.
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

// User — entity. Хранит всё, что есть в БД, включая PasswordHash.
// Никогда не сериализуется напрямую в JSON.
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	FullName     string
	Phone        *string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UserDTO — DTO для API. Не содержит PasswordHash.
type UserDTO struct {
	ID        uuid.UUID `json:"id" example:"018f7d3e-4a5b-7c9d-a0b1-c2d3e4f5a6b7"`
	Email     string    `json:"email" example:"user@example.com"`
	FullName  string    `json:"full_name" example:"Иван Петров"`
	Phone     *string   `json:"phone,omitempty" example:"+79991234567"`
	Role      Role      `json:"role" example:"user"`
	CreatedAt time.Time `json:"created_at"`
}

// ToDTO конвертирует entity в DTO, отбрасывая sensitive поля.
func (u *User) ToDTO() UserDTO {
	return UserDTO{
		ID:        u.ID,
		Email:     u.Email,
		FullName:  u.FullName,
		Phone:     u.Phone,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
	}
}

// RegisterRequest — тело POST /auth/register.
type RegisterRequest struct {
	Email    string  `json:"email" example:"user@example.com"`
	Password string  `json:"password" example:"strongpass123"`
	FullName string  `json:"full_name" example:"Иван Петров"`
	Phone    *string `json:"phone,omitempty" example:"+79991234567"`
}

// LoginRequest — тело POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"strongpass123"`
}

// UpdateMeRequest — тело PATCH /me.
// Любое подмножество полей. Для смены пароля — обязательно передать оба:
// old_password (для проверки) и new_password.
type UpdateMeRequest struct {
	FullName    *string `json:"full_name,omitempty" example:"Иван П."`
	Phone       *string `json:"phone,omitempty" example:"+79991234567"`
	OldPassword *string `json:"old_password,omitempty" example:"oldpass123"`
	NewPassword *string `json:"new_password,omitempty" example:"newpass456"`
}

// PasswordResetRequestDTO — тело POST /auth/password/reset-request.
type PasswordResetRequestDTO struct {
	Email string `json:"email" example:"user@example.com"`
}

// PasswordResetConfirmDTO — тело POST /auth/password/reset-confirm.
type PasswordResetConfirmDTO struct {
	Token       string `json:"token" example:"<токен из ссылки в письме>"`
	NewPassword string `json:"new_password" example:"newpass456"`
}

// OKResponse — стандартный ответ {"ok": true} для эндпоинтов без полезной нагрузки.
type OKResponse struct {
	OK bool `json:"ok" example:"true"`
}
