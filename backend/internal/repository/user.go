package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"yulik3d/internal/model"
)

type UserRepo struct {
	db *DB
}

func NewUserRepo(db *DB) *UserRepo {
	return &UserRepo{db: db}
}

const userCols = `id, email, password_hash, full_name, phone, role, email_verified, created_at, updated_at`

func scanUser(row pgx.Row) (model.User, error) {
	var u model.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Phone, &u.Role, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

// Create вставляет пользователя. ID сгенерирован на стороне вызывающего.
func (r *UserRepo) Create(ctx context.Context, u *model.User) error {
	const q = `
		INSERT INTO "user" (id, email, password_hash, full_name, phone, role)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at`
	return r.db.QueryRow(ctx, q, u.ID, u.Email, u.PasswordHash, u.FullName, u.Phone, u.Role).
		Scan(&u.CreatedAt, &u.UpdatedAt)
}

// GetByEmail — для логина. Возвращает ErrNoRows если нет.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (model.User, error) {
	const q = `SELECT ` + userCols + ` FROM "user" WHERE email = $1`
	return scanUser(r.db.QueryRow(ctx, q, email))
}

// GetByID — для /me и админских лукапов.
func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (model.User, error) {
	const q = `SELECT ` + userCols + ` FROM "user" WHERE id = $1`
	return scanUser(r.db.QueryRow(ctx, q, id))
}

// EmailExists — быстрая проверка занятости email.
func (r *UserRepo) EmailExists(ctx context.Context, email string) (bool, error) {
	var x int
	err := r.db.QueryRow(ctx, `SELECT 1 FROM "user" WHERE email = $1`, email).Scan(&x)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// SetEmailVerified — пометить email пользователя как подтверждённый.
// Используется после успешного перехода по ссылке из письма.
func (r *UserRepo) SetEmailVerified(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE "user" SET email_verified = TRUE, updated_at = NOW() WHERE id = $1`, id)
	return err
}

// UpdateProfile — обновление full_name / phone / password_hash.
// Передаются только ненулевые указатели.
func (r *UserRepo) UpdateProfile(ctx context.Context, id uuid.UUID, fullName *string, phone *string, passwordHash *string) (model.User, error) {
	const q = `
		UPDATE "user"
		SET full_name     = COALESCE($2, full_name),
		    phone         = COALESCE($3, phone),
		    password_hash = COALESCE($4, password_hash),
		    updated_at    = NOW()
		WHERE id = $1
		RETURNING ` + userCols
	return scanUser(r.db.QueryRow(ctx, q, id, fullName, phone, passwordHash))
}
