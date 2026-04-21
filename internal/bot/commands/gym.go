package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ramisoul84/assistant-server/internal/service"
)

type GymCommand struct {
	svc service.GymService
}

func NewGymCommand(svc service.GymService) *GymCommand {
	return &GymCommand{svc: svc}
}

// Handle responds to /gym [N] — last N sessions (default 5)
func (c *GymCommand) Handle(ctx context.Context, userID int64, args string) (string, error) {
	limit := 5
	if args != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(args)); err == nil && n > 0 {
			limit = n
		}
	}

	sessions, err := c.svc.GetSessions(ctx, userID, nil, nil, limit)
	if err != nil {
		return "", err
	}
	if len(sessions) == 0 {
		return "💪 No gym sessions recorded yet\\. Start one with /gym\\_start", nil
	}

	label := fmt.Sprintf("💪 *Last %d gym session", limit)
	if limit > 1 {
		label += "s"
	}
	label += fmt.Sprintf(" \\(%d\\)*\n\n", len(sessions))

	var sb strings.Builder
	sb.WriteString(label)
	for _, s := range sessions {
		sb.WriteString(fmt.Sprintf("📅 *%s*\n", s.StartedAt.Format("Mon Jan 2, 15:04")))
		if len(s.Exercises) == 0 {
			sb.WriteString("  \\(no exercises logged\\)\n")
		}
		for _, ex := range s.Exercises {
			line := fmt.Sprintf("  • *%s*", escapeMarkdown(ex.Name))
			if ex.Sets > 0 {
				line += fmt.Sprintf(" — %d×%d @ %.1fkg", ex.Sets, ex.Reps, ex.WeightKg)
			}
			sb.WriteString(line + "\n")
		}
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String()), nil
}
