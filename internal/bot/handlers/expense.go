package handlers

import (
	"context"
	"fmt"

	"github.com/ramisoul84/assistant-server/internal/domain"
	"github.com/ramisoul84/assistant-server/internal/service"
)

type ExpenseHandler struct {
	svc service.ExpenseService
}

func NewExpenseHandler(svc service.ExpenseService) *ExpenseHandler {
	return &ExpenseHandler{svc: svc}
}

func (h *ExpenseHandler) Handle(ctx context.Context, userID int64, parsed *domain.AIResponse) (string, error) {
	if parsed.Expense == nil {
		return "", fmt.Errorf("expense handler: no expense payload")
	}
	saved, err := h.svc.Create(ctx, userID, &service.ExpenseInput{
		Amount:      parsed.Expense.Amount,
		Currency:    parsed.Expense.Currency,
		Category:    parsed.Expense.Category,
		Description: parsed.Expense.Description,
		SpentAt:     parsed.Expense.SpentAt,
	})
	if err != nil {
		return "", err
	}
	if parsed.Reply != "" {
		return "💸 " + parsed.Reply, nil
	}
	return fmt.Sprintf("💸 *%.2f %s* saved — %s",
		saved.Amount, saved.Currency, saved.Description,
	), nil
}
