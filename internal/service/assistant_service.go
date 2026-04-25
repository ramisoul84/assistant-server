package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ramisoul84/assistant-server/internal/domain"
	"github.com/ramisoul84/assistant-server/internal/repository"
)

type AssistantService struct {
	finance repository.FinanceRepository
	notes   repository.NoteRepository
}

func NewAssistantService(finance repository.FinanceRepository, notes repository.NoteRepository) *AssistantService {
	return &AssistantService{finance: finance, notes: notes}
}

func (s *AssistantService) Save(ctx context.Context, userID int64, result *domain.AIResult) error {
	switch result.Intent {
	case domain.IntentSaveExpense:
		if result.Expense == nil {
			return fmt.Errorf("no expense payload")
		}
		_, err := s.finance.CreateExpense(ctx, &domain.Expense{
			UserID: userID, Amount: result.Expense.Amount,
			Currency:    ifEmpty(result.Expense.Currency, "EUR"),
			Category:    ifEmpty(result.Expense.Category, "other"),
			Description: result.Expense.Description,
			HappenedAt:  ifZero(result.Expense.HappenedAt),
		})
		return err

	case domain.IntentSaveIncome:
		if result.Income == nil {
			return fmt.Errorf("no income payload")
		}
		_, err := s.finance.CreateIncome(ctx, &domain.Income{
			UserID: userID, Amount: result.Income.Amount,
			Currency:    ifEmpty(result.Income.Currency, "EUR"),
			Category:    ifEmpty(result.Income.Category, "other"),
			Description: result.Income.Description,
			HappenedAt:  ifZero(result.Income.HappenedAt),
		})
		return err

	case domain.IntentSaveNote:
		if result.Note == nil {
			return fmt.Errorf("no note payload")
		}
		n := &domain.Note{UserID: userID, Content: result.Note.Content, Tags: result.Note.Tags}
		if result.Note.Datetime != nil && !result.Note.Datetime.IsZero() {
			n.Datetime = sql.NullTime{Time: *result.Note.Datetime, Valid: true}
		}
		if n.Tags == nil {
			n.Tags = []string{}
		}
		_, err := s.notes.Create(ctx, n)
		return err
	}
	return nil
}

func (s *AssistantService) GetUpcomingNotes(ctx context.Context, userID int64, from, to time.Time) ([]domain.Note, error) {
	all, err := s.notes.GetAll(ctx, userID, &from, &to)
	if err != nil {
		return nil, err
	}
	var out []domain.Note
	for _, n := range all {
		if n.Datetime.Valid {
			out = append(out, n)
		}
	}
	return out, nil
}

func ifEmpty(s, fb string) string {
	if s == "" {
		return fb
	}
	return s
}
func ifZero(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t
}
