package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/ramisoul84/assistant-server/internal/domain"
)

type OTPRepository interface {
	Create(ctx context.Context, userID int64, code string, expiresAt time.Time) (*domain.OTPCode, error)
	FindValid(ctx context.Context, userID int64, code string) (*domain.OTPCode, error)
	MarkUsed(ctx context.Context, id int64) error
	DeleteExpired(ctx context.Context) error
}

type otpRepo struct{ db *sqlx.DB }

func NewOTPRepository(db *sqlx.DB) OTPRepository {
	return &otpRepo{db: db}
}

func (r *otpRepo) Create(ctx context.Context, userID int64, code string, expiresAt time.Time) (*domain.OTPCode, error) {
	const q = `
		INSERT INTO otp_codes (user_id, code, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, code, used, expires_at, created_at`
	var otp domain.OTPCode
	if err := r.db.GetContext(ctx, &otp, q, userID, code, expiresAt); err != nil {
		return nil, fmt.Errorf("otpRepo.Create: %w", err)
	}
	return &otp, nil
}

func (r *otpRepo) FindValid(ctx context.Context, userID int64, code string) (*domain.OTPCode, error) {
	const q = `
		SELECT id, user_id, code, used, expires_at, created_at
		FROM otp_codes
		WHERE user_id = $1
		  AND code = $2
		  AND used = FALSE
		  AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1`
	var otp domain.OTPCode
	if err := r.db.GetContext(ctx, &otp, q, userID, code); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("otpRepo.FindValid: %w", err)
	}
	return &otp, nil
}

func (r *otpRepo) MarkUsed(ctx context.Context, id int64) error {
	const q = `UPDATE otp_codes SET used = TRUE WHERE id = $1`
	if _, err := r.db.ExecContext(ctx, q, id); err != nil {
		return fmt.Errorf("otpRepo.MarkUsed: %w", err)
	}
	return nil
}

func (r *otpRepo) DeleteExpired(ctx context.Context) error {
	const q = `DELETE FROM otp_codes WHERE expires_at < NOW() OR used = TRUE`
	if _, err := r.db.ExecContext(ctx, q); err != nil {
		return fmt.Errorf("otpRepo.DeleteExpired: %w", err)
	}
	return nil
}
