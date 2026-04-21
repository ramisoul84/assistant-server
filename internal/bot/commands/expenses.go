package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ramisoul84/assistant-server/internal/service"
)

type ExpensesCommand struct {
	svc service.ExpenseService
}

func NewExpensesCommand(svc service.ExpenseService) *ExpensesCommand {
	return &ExpensesCommand{svc: svc}
}

// Handle responds to /expenses [week|month|all]
// Default: this calendar month
func (c *ExpensesCommand) Handle(ctx context.Context, userID int64, args string) (string, error) {
	now := time.Now().UTC()
	var from, to *time.Time

	switch strings.TrimSpace(strings.ToLower(args)) {
	case "week":
		f := now.AddDate(0, 0, -7)
		from = &f
	case "all":
		// no filter
	default:
		// this calendar month
		f := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		from = &f
	}

	list, err := c.svc.GetFiltered(ctx, userID, from, to, 0)
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "💸 No expenses found for that period\\.", nil
	}

	var sb strings.Builder
	var total float64
	currency := list[0].Currency

	sb.WriteString(fmt.Sprintf("💸 *Expenses \\(%d\\)*\n\n", len(list)))
	for _, e := range list {
		sb.WriteString(fmt.Sprintf("• *%.2f %s* — %s\n  🏷️ %s  📅 %s\n\n",
			e.Amount, e.Currency,
			escapeMarkdown(e.Description),
			escapeMarkdown(e.Category),
			e.SpentAt.Format("Jan 2"),
		))
		total += e.Amount
	}
	sb.WriteString(fmt.Sprintf("*Total: %.2f %s*", total, escapeMarkdown(currency)))
	return strings.TrimSpace(sb.String()), nil
}
