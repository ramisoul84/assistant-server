package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ramisoul84/assistant-server/internal/domain"
	"github.com/ramisoul84/assistant-server/internal/repository"
)

type ExpenseInput struct {
	Amount      float64
	Currency    string
	Category    string
	Description string
	SpentAt     time.Time
}

type ExpenseService interface {
	Create(ctx context.Context, userID int64, input *ExpenseInput) (*domain.Expense, error)
	Update(ctx context.Context, id, userID int64, input *ExpenseInput) (*domain.Expense, error)
	Delete(ctx context.Context, id, userID int64) error
	GetFiltered(ctx context.Context, userID int64, from, to *time.Time, limit int) ([]domain.Expense, error)
}

type expenseService struct{ repo repository.ExpenseRepository }

func NewExpenseService(repo repository.ExpenseRepository) ExpenseService {
	return &expenseService{repo: repo}
}

func (s *expenseService) Create(ctx context.Context, userID int64, input *ExpenseInput) (*domain.Expense, error) {
	if input.Amount <= 0 {
		return nil, fmt.Errorf("%w: amount must be positive", domain.ErrInvalidInput)
	}
	currency := input.Currency
	if currency == "" {
		currency = "EUR"
	}
	spentAt := input.SpentAt
	if spentAt.IsZero() {
		spentAt = time.Now().UTC()
	}
	return s.repo.Create(ctx, &domain.Expense{
		UserID: userID, Amount: input.Amount, Currency: currency,
		Category: input.Category, Description: input.Description, SpentAt: spentAt,
	})
}

func (s *expenseService) Update(ctx context.Context, id, userID int64, input *ExpenseInput) (*domain.Expense, error) {
	if input.Amount <= 0 {
		return nil, fmt.Errorf("%w: amount must be positive", domain.ErrInvalidInput)
	}
	return s.repo.Update(ctx, id, userID, input.Amount, input.Currency, input.Category, input.Description, input.SpentAt)
}

func (s *expenseService) Delete(ctx context.Context, id, userID int64) error {
	return s.repo.Delete(ctx, id, userID)
}

func (s *expenseService) GetFiltered(ctx context.Context, userID int64, from, to *time.Time, limit int) ([]domain.Expense, error) {
	return s.repo.GetFiltered(ctx, userID, from, to, limit)
}
