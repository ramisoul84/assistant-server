package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ramisoul84/assistant-server/internal/service"
)

type SummaryCommand struct {
	appointmentSvc service.AppointmentService
	expenseSvc     service.ExpenseService
	gymSvc         service.GymService
}

func NewSummaryCommand(a service.AppointmentService, e service.ExpenseService, g service.GymService) *SummaryCommand {
	return &SummaryCommand{appointmentSvc: a, expenseSvc: e, gymSvc: g}
}

func (c *SummaryCommand) Handle(ctx context.Context, userID int64) (string, error) {
	now := time.Now().UTC()

	apptFrom := now
	apptTo := now.AddDate(0, 0, 7)
	appointments, err := c.appointmentSvc.GetFiltered(ctx, userID, &apptFrom, &apptTo, 0)
	if err != nil {
		return "", err
	}

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	expenses, err := c.expenseSvc.GetFiltered(ctx, userID, &monthStart, nil, 0)
	if err != nil {
		return "", err
	}

	sessions, err := c.gymSvc.GetSessions(ctx, userID, nil, nil, 1)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📊 *Summary — %s*\n\n", escapeMarkdown(now.Format("Mon Jan 2"))))

	sb.WriteString("📅 *Upcoming \\(7 days\\)*\n")
	if len(appointments) == 0 {
		sb.WriteString("  Nothing scheduled\n")
	} else {
		for _, a := range appointments {
			sb.WriteString(fmt.Sprintf("  • %s — %s\n",
				escapeMarkdown(a.Title),
				escapeMarkdown(a.Datetime.Format("Mon Jan 2 at 15:04")),
			))
		}
	}
	sb.WriteString("\n")

	var total float64
	currency := "EUR"
	for _, e := range expenses {
		total += e.Amount
		currency = e.Currency
	}
	sb.WriteString("💸 *Spending this month*\n")
	sb.WriteString(fmt.Sprintf("  %d expense\\(s\\) — *%.2f %s*\n\n",
		len(expenses), total, escapeMarkdown(currency),
	))

	sb.WriteString("💪 *Last gym session*\n")
	if len(sessions) == 0 {
		sb.WriteString("  No sessions yet — /gym\\_start to log one\n")
	} else {
		s := sessions[0]
		sb.WriteString(fmt.Sprintf("  %s \\(%d exercise\\(s\\)\\)\n",
			escapeMarkdown(s.StartedAt.Format("Mon Jan 2")),
			len(s.Exercises),
		))
	}

	return strings.TrimSpace(sb.String()), nil
}
