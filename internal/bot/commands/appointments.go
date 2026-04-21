package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ramisoul84/assistant-server/internal/service"
)

type AppointmentsCommand struct {
	svc service.AppointmentService
}

func NewAppointmentsCommand(svc service.AppointmentService) *AppointmentsCommand {
	return &AppointmentsCommand{svc: svc}
}

// Handle responds to /appointments [week|month|all]
// Default (no arg): upcoming from now onward
func (c *AppointmentsCommand) Handle(ctx context.Context, userID int64, args string) (string, error) {
	now := time.Now().UTC()
	var from, to *time.Time

	switch strings.TrimSpace(strings.ToLower(args)) {
	case "week":
		f := now
		t := now.AddDate(0, 0, 7)
		from, to = &f, &t
	case "month":
		f := now
		t := now.AddDate(0, 1, 0)
		from, to = &f, &t
	case "all":
		// no filter
	default:
		// upcoming — from now, no upper bound
		f := now
		from = &f
	}

	list, err := c.svc.GetFiltered(ctx, userID, from, to, 0)
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "📅 No appointments found\\.", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📅 *Appointments \\(%d\\)*\n\n", len(list)))
	for _, a := range list {
		sb.WriteString(fmt.Sprintf("• *%s*\n  🕐 %s\n",
			escapeMarkdown(a.Title),
			escapeMarkdown(a.Datetime.Format("Mon, Jan 2 2006 at 15:04")),
		))
		if a.Notes != "" {
			sb.WriteString(fmt.Sprintf("  📝 %s\n", escapeMarkdown(a.Notes)))
		}
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String()), nil
}
