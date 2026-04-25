package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/ramisoul84/assistant-server/internal/domain"
)

type FinanceRepository interface {
	CreateExpense(ctx context.Context, e *domain.Expense) (*domain.Expense, error)
	UpdateExpense(ctx context.Context, id, userID int64, amount float64, currency, category, description string, happenedAt time.Time) (*domain.Expense, error)
	DeleteExpense(ctx context.Context, id, userID int64) error
	GetExpenses(ctx context.Context, userID int64, from, to *time.Time) ([]domain.Expense, error)

	CreateIncome(ctx context.Context, i *domain.Income) (*domain.Income, error)
	UpdateIncome(ctx context.Context, id, userID int64, amount float64, currency, category, description string, happenedAt time.Time) (*domain.Income, error)
	DeleteIncome(ctx context.Context, id, userID int64) error
	GetIncomes(ctx context.Context, userID int64, from, to *time.Time) ([]domain.Income, error)

	UpsertBudget(ctx context.Context, userID int64, amount float64, currency string) (*domain.BudgetLimit, error)
	GetBudget(ctx context.Context, userID int64) (*domain.BudgetLimit, error)
	GetAllBudgets(ctx context.Context) ([]domain.BudgetLimit, error)
}

type financeRepo struct{ db *sqlx.DB }

func NewFinanceRepository(db *sqlx.DB) FinanceRepository { return &financeRepo{db: db} }

func (r *financeRepo) CreateExpense(ctx context.Context, e *domain.Expense) (*domain.Expense, error) {
	const q = `INSERT INTO expenses (user_id,amount,currency,category,description,happened_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id,user_id,amount,currency,category,description,happened_at,created_at`
	if err := r.db.GetContext(ctx, e, q, e.UserID, e.Amount, e.Currency, e.Category, e.Description, e.HappenedAt); err != nil {
		return nil, fmt.Errorf("financeRepo.CreateExpense: %w", err)
	}
	return e, nil
}

func (r *financeRepo) UpdateExpense(ctx context.Context, id, userID int64, amount float64, currency, category, description string, happenedAt time.Time) (*domain.Expense, error) {
	const q = `UPDATE expenses SET amount=$1,currency=$2,category=$3,description=$4,happened_at=$5
		WHERE id=$6 AND user_id=$7
		RETURNING id,user_id,amount,currency,category,description,happened_at,created_at`
	var e domain.Expense
	if err := r.db.GetContext(ctx, &e, q, amount, currency, category, description, happenedAt, id, userID); err != nil {
		return nil, fmt.Errorf("financeRepo.UpdateExpense: %w", err)
	}
	return &e, nil
}

func (r *financeRepo) DeleteExpense(ctx context.Context, id, userID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM expenses WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

func (r *financeRepo) GetExpenses(ctx context.Context, userID int64, from, to *time.Time) ([]domain.Expense, error) {
	q := `SELECT id,user_id,amount,currency,category,description,happened_at,created_at FROM expenses WHERE user_id=$1`
	args := []any{userID}
	i := 2
	if from != nil {
		q += fmt.Sprintf(" AND happened_at>=$%d", i)
		args = append(args, *from)
		i++
	}
	if to != nil {
		q += fmt.Sprintf(" AND happened_at<=$%d", i)
		args = append(args, *to)
	}
	q += " ORDER BY happened_at DESC"
	var list []domain.Expense
	if err := r.db.SelectContext(ctx, &list, q, args...); err != nil {
		return nil, fmt.Errorf("financeRepo.GetExpenses: %w", err)
	}
	return list, nil
}

func (r *financeRepo) CreateIncome(ctx context.Context, i *domain.Income) (*domain.Income, error) {
	const q = `INSERT INTO incomes (user_id,amount,currency,category,description,happened_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id,user_id,amount,currency,category,description,happened_at,created_at`
	if err := r.db.GetContext(ctx, i, q, i.UserID, i.Amount, i.Currency, i.Category, i.Description, i.HappenedAt); err != nil {
		return nil, fmt.Errorf("financeRepo.CreateIncome: %w", err)
	}
	return i, nil
}

func (r *financeRepo) UpdateIncome(ctx context.Context, id, userID int64, amount float64, currency, category, description string, happenedAt time.Time) (*domain.Income, error) {
	const q = `UPDATE incomes SET amount=$1,currency=$2,category=$3,description=$4,happened_at=$5
		WHERE id=$6 AND user_id=$7
		RETURNING id,user_id,amount,currency,category,description,happened_at,created_at`
	var i domain.Income
	if err := r.db.GetContext(ctx, &i, q, amount, currency, category, description, happenedAt, id, userID); err != nil {
		return nil, fmt.Errorf("financeRepo.UpdateIncome: %w", err)
	}
	return &i, nil
}

func (r *financeRepo) DeleteIncome(ctx context.Context, id, userID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM incomes WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

func (r *financeRepo) GetIncomes(ctx context.Context, userID int64, from, to *time.Time) ([]domain.Income, error) {
	q := `SELECT id,user_id,amount,currency,category,description,happened_at,created_at FROM incomes WHERE user_id=$1`
	args := []any{userID}
	i := 2
	if from != nil {
		q += fmt.Sprintf(" AND happened_at>=$%d", i)
		args = append(args, *from)
		i++
	}
	if to != nil {
		q += fmt.Sprintf(" AND happened_at<=$%d", i)
		args = append(args, *to)
	}
	q += " ORDER BY happened_at DESC"
	var list []domain.Income
	if err := r.db.SelectContext(ctx, &list, q, args...); err != nil {
		return nil, fmt.Errorf("financeRepo.GetIncomes: %w", err)
	}
	return list, nil
}

func (r *financeRepo) UpsertBudget(ctx context.Context, userID int64, amount float64, currency string) (*domain.BudgetLimit, error) {
	const q = `INSERT INTO budget_limits (user_id,amount,currency) VALUES ($1,$2,$3)
		ON CONFLICT (user_id) DO UPDATE SET amount=$2,currency=$3,updated_at=NOW()
		RETURNING id,user_id,amount,currency,updated_at`
	var b domain.BudgetLimit
	if err := r.db.GetContext(ctx, &b, q, userID, amount, currency); err != nil {
		return nil, fmt.Errorf("financeRepo.UpsertBudget: %w", err)
	}
	return &b, nil
}

func (r *financeRepo) GetBudget(ctx context.Context, userID int64) (*domain.BudgetLimit, error) {
	var b domain.BudgetLimit
	if err := r.db.GetContext(ctx, &b, `SELECT id,user_id,amount,currency,updated_at FROM budget_limits WHERE user_id=$1`, userID); err != nil {
		return nil, domain.ErrNotFound
	}
	return &b, nil
}

func (r *financeRepo) GetAllBudgets(ctx context.Context) ([]domain.BudgetLimit, error) {
	var list []domain.BudgetLimit
	if err := r.db.SelectContext(ctx, &list, `SELECT id,user_id,amount,currency,updated_at FROM budget_limits`); err != nil {
		return nil, err
	}
	return list, nil
}
