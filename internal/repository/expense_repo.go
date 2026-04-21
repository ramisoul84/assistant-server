package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/ramisoul84/assistant-server/internal/domain"
)

type ExpenseRepository interface {
	Create(ctx context.Context, e *domain.Expense) (*domain.Expense, error)
	Update(ctx context.Context, id, userID int64, amount float64, currency, category, description string, spentAt time.Time) (*domain.Expense, error)
	Delete(ctx context.Context, id, userID int64) error
	GetFiltered(ctx context.Context, userID int64, from, to *time.Time, limit int) ([]domain.Expense, error)
}

type expenseRepo struct{ db *sqlx.DB }

func NewExpenseRepository(db *sqlx.DB) ExpenseRepository {
	return &expenseRepo{db: db}
}

func (r *expenseRepo) Create(ctx context.Context, e *domain.Expense) (*domain.Expense, error) {
	const q = `
		INSERT INTO expenses (user_id, amount, currency, category, description, spent_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, amount, currency, category, description, spent_at, created_at`
	if err := r.db.GetContext(ctx, e, q,
		e.UserID, e.Amount, e.Currency, e.Category, e.Description, e.SpentAt,
	); err != nil {
		return nil, fmt.Errorf("expenseRepo.Create: %w", err)
	}
	return e, nil
}

func (r *expenseRepo) Update(ctx context.Context, id, userID int64, amount float64, currency, category, description string, spentAt time.Time) (*domain.Expense, error) {
	const q = `
		UPDATE expenses
		SET amount=$1, currency=$2, category=$3, description=$4, spent_at=$5
		WHERE id=$6 AND user_id=$7
		RETURNING id, user_id, amount, currency, category, description, spent_at, created_at`
	var e domain.Expense
	if err := r.db.GetContext(ctx, &e, q,
		amount, currency, category, description, spentAt, id, userID,
	); err != nil {
		return nil, fmt.Errorf("expenseRepo.Update: %w", err)
	}
	return &e, nil
}

func (r *expenseRepo) Delete(ctx context.Context, id, userID int64) error {
	const q = `DELETE FROM expenses WHERE id=$1 AND user_id=$2`
	if _, err := r.db.ExecContext(ctx, q, id, userID); err != nil {
		return fmt.Errorf("expenseRepo.Delete: %w", err)
	}
	return nil
}

func (r *expenseRepo) GetFiltered(ctx context.Context, userID int64, from, to *time.Time, limit int) ([]domain.Expense, error) {
	q := `SELECT id, user_id, amount, currency, category, description, spent_at, created_at
		  FROM expenses WHERE user_id = $1`
	args := []any{userID}
	i := 2
	if from != nil {
		q += fmt.Sprintf(" AND spent_at >= $%d", i); args = append(args, *from); i++
	}
	if to != nil {
		q += fmt.Sprintf(" AND spent_at <= $%d", i); args = append(args, *to); i++
	}
	q += " ORDER BY spent_at DESC"
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT $%d", i); args = append(args, limit)
	}
	var list []domain.Expense
	if err := r.db.SelectContext(ctx, &list, q, args...); err != nil {
		return nil, fmt.Errorf("expenseRepo.GetFiltered: %w", err)
	}
	return list, nil
}
