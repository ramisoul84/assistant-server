package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ramisoul84/assistant-server/internal/domain"
	"github.com/ramisoul84/assistant-server/internal/repository"
)

type IncomeInput struct {
	Amount      float64
	Currency    string
	Category    string
	Description string
	ReceivedAt  time.Time
}

type IncomeService interface {
	Create(ctx context.Context, userID int64, input *IncomeInput) (*domain.Income, error)
	Update(ctx context.Context, id, userID int64, input *IncomeInput) (*domain.Income, error)
	Delete(ctx context.Context, id, userID int64) error
	GetFiltered(ctx context.Context, userID int64, from, to *time.Time, limit int) ([]domain.Income, error)
}

type incomeService struct{ repo repository.IncomeRepository }

func NewIncomeService(repo repository.IncomeRepository) IncomeService {
	return &incomeService{repo: repo}
}

func (s *incomeService) Create(ctx context.Context, userID int64, input *IncomeInput) (*domain.Income, error) {
	if input.Amount <= 0 {
		return nil, fmt.Errorf("%w: amount must be positive", domain.ErrInvalidInput)
	}
	currency := input.Currency
	if currency == "" { currency = "EUR" }
	receivedAt := input.ReceivedAt
	if receivedAt.IsZero() { receivedAt = time.Now().UTC() }
	return s.repo.Create(ctx, &domain.Income{
		UserID: userID, Amount: input.Amount, Currency: currency,
		Category: input.Category, Description: input.Description, ReceivedAt: receivedAt,
	})
}

func (s *incomeService) Update(ctx context.Context, id, userID int64, input *IncomeInput) (*domain.Income, error) {
	if input.Amount <= 0 {
		return nil, fmt.Errorf("%w: amount must be positive", domain.ErrInvalidInput)
	}
	return s.repo.Update(ctx, id, userID, input.Amount, input.Currency, input.Category, input.Description, input.ReceivedAt)
}

func (s *incomeService) Delete(ctx context.Context, id, userID int64) error {
	return s.repo.Delete(ctx, id, userID)
}

func (s *incomeService) GetFiltered(ctx context.Context, userID int64, from, to *time.Time, limit int) ([]domain.Income, error) {
	return s.repo.GetFiltered(ctx, userID, from, to, limit)
}
