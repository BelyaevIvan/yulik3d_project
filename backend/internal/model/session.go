package model

import (
	"time"

	"github.com/google/uuid"
)

// Session — JSON-значение, хранимое в Redis по ключу session:<id>.
type Session struct {
	UserID    uuid.UUID `json:"user_id"`
	Role      Role      `json:"role"`
	FullName  string    `json:"full_name"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	UserAgent string    `json:"user_agent,omitempty"`
	IP        string    `json:"ip,omitempty"`
}
