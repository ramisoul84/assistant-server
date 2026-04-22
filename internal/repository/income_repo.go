package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/ramisoul84/assistant-server/internal/domain"
)

type IncomeRepository interface {
	Create(ctx context.Context, i *domain.Income) (*domain.Income, error)
	Update(ctx context.Context, id, userID int64, amount float64, currency, category, description string, receivedAt time.Time) (*domain.Income, error)
	Delete(ctx context.Context, id, userID int64) error
	GetFiltered(ctx context.Context, userID int64, from, to *time.Time, limit int) ([]domain.Income, error)
}

type incomeRepo struct{ db *sqlx.DB }

func NewIncomeRepository(db *sqlx.DB) IncomeRepository { return &incomeRepo{db: db} }

func (r *incomeRepo) Create(ctx context.Context, i *domain.Income) (*domain.Income, error) {
	const q = `
		INSERT INTO incomes (user_id, amount, currency, category, description, received_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, user_id, amount, currency, category, description, received_at, created_at`
	if err := r.db.GetContext(ctx, i, q, i.UserID, i.Amount, i.Currency, i.Category, i.Description, i.ReceivedAt); err != nil {
		return nil, fmt.Errorf("incomeRepo.Create: %w", err)
	}
	return i, nil
}

func (r *incomeRepo) Update(ctx context.Context, id, userID int64, amount float64, currency, category, description string, receivedAt time.Time) (*domain.Income, error) {
	const q = `
		UPDATE incomes SET amount=$1,currency=$2,category=$3,description=$4,received_at=$5
		WHERE id=$6 AND user_id=$7
		RETURNING id,user_id,amount,currency,category,description,received_at,created_at`
	var i domain.Income
	if err := r.db.GetContext(ctx, &i, q, amount, currency, category, description, receivedAt, id, userID); err != nil {
		return nil, fmt.Errorf("incomeRepo.Update: %w", err)
	}
	return &i, nil
}

func (r *incomeRepo) Delete(ctx context.Context, id, userID int64) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM incomes WHERE id=$1 AND user_id=$2`, id, userID); err != nil {
		return fmt.Errorf("incomeRepo.Delete: %w", err)
	}
	return nil
}

func (r *incomeRepo) GetFiltered(ctx context.Context, userID int64, from, to *time.Time, limit int) ([]domain.Income, error) {
	q := `SELECT id,user_id,amount,currency,category,description,received_at,created_at FROM incomes WHERE user_id=$1`
	args := []any{userID}
	i := 2
	if from != nil { q += fmt.Sprintf(" AND received_at>=$%d", i); args = append(args, *from); i++ }
	if to   != nil { q += fmt.Sprintf(" AND received_at<=$%d", i); args = append(args, *to);   i++ }
	q += " ORDER BY received_at DESC"
	if limit > 0 { q += fmt.Sprintf(" LIMIT $%d", i); args = append(args, limit) }
	var list []domain.Income
	if err := r.db.SelectContext(ctx, &list, q, args...); err != nil {
		return nil, fmt.Errorf("incomeRepo.GetFiltered: %w", err)
	}
	return list, nil
}
