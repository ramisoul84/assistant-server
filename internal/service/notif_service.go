package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ramisoul84/assistant-server/internal/domain"
	"github.com/ramisoul84/assistant-server/internal/repository"
	"github.com/ramisoul84/assistant-server/pkg/logger"
)

type NotifService struct {
	notes    repository.NoteRepository
	finance  repository.FinanceRepository
	users    repository.UserRepository
	notifs   repository.NotificationRepository
	notifier Notifier
	log      logger.Logger
}

func NewNotifService(notes repository.NoteRepository, finance repository.FinanceRepository, users repository.UserRepository, notifs repository.NotificationRepository, notifier Notifier) *NotifService {
	return &NotifService{notes: notes, finance: finance, users: users, notifs: notifs, notifier: notifier, log: logger.Get()}
}

// CheckAppointments runs every minute — sends reminders 24h and 1h before.
func (s *NotifService) CheckAppointments(ctx context.Context) {
	now := time.Now().UTC()
	windows := []struct {
		from, to time.Duration
		typ      domain.NotificationType
		msg      string
	}{
		{23*time.Hour + 55*time.Minute, 24*time.Hour + 5*time.Minute, domain.NotifAppointment24h, "📅 *Tomorrow:* _%s_\n🕐 %s"},
		{55 * time.Minute, 65 * time.Minute, domain.NotifAppointment1h, "⏰ *In 1 hour:* _%s_\n🕐 %s"},
	}

	for _, w := range windows {
		upcoming, err := s.notes.GetUpcoming(ctx, now.Add(w.from), now.Add(w.to))
		if err != nil {
			s.log.Error().Err(err).Msg("notif: get upcoming failed")
			continue
		}

		for _, note := range upcoming {
			refID := fmt.Sprintf("%d", note.ID)
			sent, _ := s.notifs.WasSent(ctx, note.UserID, w.typ, refID)
			if sent {
				continue
			}

			user, err := s.users.FindByID(ctx, note.UserID)
			if err != nil {
				continue
			}

			msg := fmt.Sprintf(w.msg, note.Content, note.Datetime.Time.Format("15:04"))
			if err := s.notifier.SendMessage(user.TelegramID, msg); err != nil {
				s.log.Error().Err(err).Msg("notif: send failed")
				continue
			}
			_ = s.notifs.MarkSent(ctx, note.UserID, w.typ, refID)
			s.log.Info().Int64("note_id", note.ID).Str("type", string(w.typ)).Msg("appointment reminder sent")
		}
	}
}

// CheckBudgets runs every hour — alerts at 50%, 80%, 100%.
func (s *NotifService) CheckBudgets(ctx context.Context) {
	budgets, err := s.finance.GetAllBudgets(ctx)
	if err != nil || len(budgets) == 0 {
		return
	}

	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthKey := now.Format("2006-01")

	thresholds := []struct {
		pct float64
		typ domain.NotificationType
	}{{50, domain.NotifBudget50}, {80, domain.NotifBudget80}, {100, domain.NotifBudget100}}

	for _, b := range budgets {
		expenses, err := s.finance.GetExpenses(ctx, b.UserID, &monthStart, nil)
		if err != nil {
			continue
		}

		var spent float64
		for _, e := range expenses {
			spent += e.Amount
		}
		pct := (spent / b.Amount) * 100

		for _, t := range thresholds {
			if pct < t.pct {
				continue
			}
			sent, _ := s.notifs.WasSent(ctx, b.UserID, t.typ, monthKey)
			if sent {
				continue
			}

			user, err := s.users.FindByID(ctx, b.UserID)
			if err != nil {
				continue
			}

			var msg string
			switch t.typ {
			case domain.NotifBudget50:
				msg = fmt.Sprintf("💛 You've used *50%%* of your monthly budget.\n%.2f / %.2f %s", spent, b.Amount, b.Currency)
			case domain.NotifBudget80:
				msg = fmt.Sprintf("🟠 *80%% of budget used!*\n%.2f / %.2f %s — only %.2f left.", spent, b.Amount, b.Currency, b.Amount-spent)
			default:
				msg = fmt.Sprintf("🔴 *Budget exceeded!*\n%.2f / %.2f %s spent this month.", spent, b.Amount, b.Currency)
			}

			if s.notifier.SendMessage(user.TelegramID, msg) == nil {
				_ = s.notifs.MarkSent(ctx, b.UserID, t.typ, monthKey)
			}
		}
	}
}
