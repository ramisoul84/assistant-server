package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/ramisoul84/assistant-server/internal/domain"
)

type BudgetRepository interface {
	Upsert(ctx context.Context, userID int64, amount float64, currency string) (*domain.BudgetLimit, error)
	GetByUserID(ctx context.Context, userID int64) (*domain.BudgetLimit, error)
	GetAll(ctx context.Context) ([]domain.BudgetLimit, error)
}

type budgetRepo struct{ db *sqlx.DB }

func NewBudgetRepository(db *sqlx.DB) BudgetRepository {
	return &budgetRepo{db: db}
}

func (r *budgetRepo) Upsert(ctx context.Context, userID int64, amount float64, currency string) (*domain.BudgetLimit, error) {
	const q = `
		INSERT INTO budget_limits (user_id, amount, currency, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id) DO UPDATE
			SET amount     = EXCLUDED.amount,
			    currency   = EXCLUDED.currency,
			    updated_at = NOW()
		RETURNING id, user_id, amount, currency, created_at, updated_at`
	var b domain.BudgetLimit
	if err := r.db.GetContext(ctx, &b, q, userID, amount, currency); err != nil {
		return nil, fmt.Errorf("budgetRepo.Upsert: %w", err)
	}
	return &b, nil
}

func (r *budgetRepo) GetByUserID(ctx context.Context, userID int64) (*domain.BudgetLimit, error) {
	const q = `SELECT id, user_id, amount, currency, created_at, updated_at FROM budget_limits WHERE user_id=$1`
	var b domain.BudgetLimit
	if err := r.db.GetContext(ctx, &b, q, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("budgetRepo.GetByUserID: %w", err)
	}
	return &b, nil
}

func (r *budgetRepo) GetAll(ctx context.Context) ([]domain.BudgetLimit, error) {
	const q = `SELECT id, user_id, amount, currency, created_at, updated_at FROM budget_limits`
	var list []domain.BudgetLimit
	if err := r.db.SelectContext(ctx, &list, q); err != nil {
		return nil, fmt.Errorf("budgetRepo.GetAll: %w", err)
	}
	return list, nil
}
