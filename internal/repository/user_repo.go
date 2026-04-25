package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/ramisoul84/assistant-server/internal/domain"
)

type UserRepository interface {
	FindOrCreate(ctx context.Context, telegramID int64, handle, firstName string) (*domain.User, error)
	FindByHandle(ctx context.Context, handle string) (*domain.User, error)
	FindByID(ctx context.Context, id int64) (*domain.User, error)
}

type userRepo struct{ db *sqlx.DB }

func NewUserRepository(db *sqlx.DB) UserRepository { return &userRepo{db: db} }

func (r *userRepo) FindOrCreate(ctx context.Context, telegramID int64, handle, firstName string) (*domain.User, error) {
	const q = `INSERT INTO users (telegram_id,handle,first_name) VALUES ($1,$2,$3)
		ON CONFLICT (telegram_id) DO UPDATE SET handle=$2,first_name=$3
		RETURNING id,telegram_id,handle,first_name,created_at`
	var u domain.User
	if err := r.db.GetContext(ctx, &u, q, telegramID, handle, firstName); err != nil {
		return nil, fmt.Errorf("userRepo.FindOrCreate: %w", err)
	}
	return &u, nil
}

func (r *userRepo) FindByHandle(ctx context.Context, handle string) (*domain.User, error) {
	var u domain.User
	if err := r.db.GetContext(ctx, &u, `SELECT id,telegram_id,handle,first_name,created_at FROM users WHERE handle=$1`, handle); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("userRepo.FindByHandle: %w", err)
	}
	return &u, nil
}

func (r *userRepo) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	var u domain.User
	if err := r.db.GetContext(ctx, &u, `SELECT id,telegram_id,handle,first_name,created_at FROM users WHERE id=$1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("userRepo.FindByID: %w", err)
	}
	return &u, nil
}
