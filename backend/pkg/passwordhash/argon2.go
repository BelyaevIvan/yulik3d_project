// Package passwordhash реализует хэширование паролей через argon2id.
// Формат хранения — PHC-совместимая строка.
package passwordhash

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	saltLen = 16
	keyLen  = 32
)

// Params — параметры argon2id. Подбирай под свою инфру. Значения ниже —
// разумный production-дефолт: 64 MiB RAM, 3 итерации, 2 параллели.
type Params struct {
	Memory      uint32 // KiB
	Iterations  uint32
	Parallelism uint8
}

// DefaultParams — production-дефолт.
var DefaultParams = Params{
	Memory:      64 * 1024,
	Iterations:  3,
	Parallelism: 2,
}

// ErrMismatch — пароль не совпал с хэшем.
var ErrMismatch = errors.New("password does not match")

// Hash строит PHC-строку для пароля.
func Hash(password string, p Params) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, keyLen)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		p.Memory, p.Iterations, p.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// Verify сверяет пароль с PHC-строкой. Возвращает ErrMismatch при несовпадении,
// другую ошибку при некорректном формате хэша.
func Verify(password, encoded string) error {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return errors.New("invalid hash format")
	}
	if parts[1] != "argon2id" {
		return errors.New("unsupported algorithm")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return fmt.Errorf("version: %w", err)
	}
	if version != argon2.Version {
		return errors.New("incompatible argon2 version")
	}
	var p Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Iterations, &p.Parallelism); err != nil {
		return fmt.Errorf("params: %w", err)
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return fmt.Errorf("salt: %w", err)
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return fmt.Errorf("key: %w", err)
	}
	actual := argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, uint32(len(expected)))
	if subtle.ConstantTimeCompare(expected, actual) != 1 {
		return ErrMismatch
	}
	return nil
}
