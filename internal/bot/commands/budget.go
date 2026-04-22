package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ramisoul84/assistant-server/internal/domain"
	"github.com/ramisoul84/assistant-server/internal/repository"
)

type BudgetCommand struct {
	budgetRepo repository.BudgetRepository
}

func NewBudgetCommand(repo repository.BudgetRepository) *BudgetCommand {
	return &BudgetCommand{budgetRepo: repo}
}

// Handle responds to /budget [amount] [currency]
// Examples:
//   /budget           → show current limit
//   /budget 1500      → set 1500 EUR limit
//   /budget 2000 USD  → set 2000 USD limit
func (c *BudgetCommand) Handle(ctx context.Context, userID int64, args string) (string, error) {
	args = strings.TrimSpace(args)

	// No args — show current limit
	if args == "" {
		limit, err := c.budgetRepo.GetByUserID(ctx, userID)
		if err != nil {
			if err == domain.ErrNotFound {
				return "💰 No budget limit set\\. Use:\n`/budget 1500` to set a monthly limit of 1500 EUR", nil
			}
			return "", err
		}
		return fmt.Sprintf("💰 *Monthly budget limit:* %.2f %s\n\nUse `/budget <amount>` to update it\\.",
			limit.Amount, escapeMarkdown(limit.Currency)), nil
	}

	// Parse args: amount [currency]
	parts := strings.Fields(args)
	amount, err := strconv.ParseFloat(parts[0], 64)
	if err != nil || amount <= 0 {
		return "❌ Invalid amount\\. Example: `/budget 1500` or `/budget 2000 USD`", nil
	}

	currency := "EUR"
	if len(parts) >= 2 {
		currency = strings.ToUpper(parts[1])
	}

	limit, err := c.budgetRepo.Upsert(ctx, userID, amount, currency)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("✅ Monthly budget set to *%.2f %s*\n\nI'll notify you at 50%%, 80%%, and 100%%\\.",
		limit.Amount, escapeMarkdown(limit.Currency)), nil
}
