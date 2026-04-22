package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ramisoul84/assistant-server/internal/domain"
	"github.com/ramisoul84/assistant-server/internal/repository"
	"github.com/ramisoul84/assistant-server/pkg/logger"
)

type NotificationService struct {
	apptRepo    repository.AppointmentRepository
	expenseRepo repository.ExpenseRepository
	userRepo    repository.UserRepository
	budgetRepo  repository.BudgetRepository
	notifRepo   repository.NotificationRepository
	notifier    Notifier
	log         logger.Logger
}

func NewNotificationService(
	apptRepo repository.AppointmentRepository,
	expenseRepo repository.ExpenseRepository,
	userRepo repository.UserRepository,
	budgetRepo repository.BudgetRepository,
	notifRepo repository.NotificationRepository,
	notifier Notifier,
) *NotificationService {
	return &NotificationService{
		apptRepo:    apptRepo,
		expenseRepo: expenseRepo,
		userRepo:    userRepo,
		budgetRepo:  budgetRepo,
		notifRepo:   notifRepo,
		notifier:    notifier,
		log:         logger.Get(),
	}
}

// ── Appointment reminders — call every minute ─────────────────────────────

func (s *NotificationService) CheckAppointments(ctx context.Context) {
	now := time.Now().UTC()

	windows := []struct {
		label     string
		from, to  time.Duration
		notifType domain.NotificationType
		msgFmt    string
	}{
		{
			label:     "24h",
			from:      23*time.Hour + 55*time.Minute,
			to:        24*time.Hour + 5*time.Minute,
			notifType: domain.NotifAppointment24h,
			msgFmt:    "📅 *Reminder:* _%s_ is *tomorrow* at *%s*",
		},
		{
			label:     "1h",
			from:      55 * time.Minute,
			to:        65 * time.Minute,
			notifType: domain.NotifAppointment1h,
			msgFmt:    "⏰ *Heads up:* _%s_ starts in about *1 hour* \\(%s\\)",
		},
	}

	for _, w := range windows {
		appts, err := s.apptRepo.GetInWindow(ctx, now.Add(w.from), now.Add(w.to))
		if err != nil {
			s.log.Error().Err(err).Str("window", w.label).Msg("Failed to fetch appointments for notification")
			continue
		}

		for _, appt := range appts {
			refID := fmt.Sprintf("%d", appt.ID)
			already, err := s.notifRepo.WasSent(ctx, appt.UserID, w.notifType, refID)
			if err != nil || already {
				continue
			}

			user, err := s.userRepo.FindByID(ctx, appt.UserID)
			if err != nil {
				s.log.Error().Err(err).Int64("user_id", appt.UserID).Msg("User not found for notification")
				continue
			}

			msg := fmt.Sprintf(w.msgFmt, appt.Title, appt.Datetime.Format("15:04"))
			if appt.Notes != "" {
				msg += fmt.Sprintf("\n📝 _%s_", appt.Notes)
			}

			if err := s.notifier.SendMessage(user.TelegramID, msg); err != nil {
				s.log.Error().Err(err).Int64("appt_id", appt.ID).Msg("Failed to send appointment reminder")
				continue
			}

			_ = s.notifRepo.MarkSent(ctx, appt.UserID, w.notifType, refID)
			s.log.Info().Int64("appt_id", appt.ID).Str("window", w.label).Str("title", appt.Title).Msg("Appointment reminder sent")
		}
	}
}

// ── Budget alerts — call every hour ──────────────────────────────────────

func (s *NotificationService) CheckBudgets(ctx context.Context) {
	limits, err := s.budgetRepo.GetAll(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to fetch budget limits")
		return
	}
	if len(limits) == 0 {
		return
	}

	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthKey := now.Format("2006-01")

	thresholds := []struct {
		pct       float64
		notifType domain.NotificationType
	}{
		{50, domain.NotifBudget50},
		{80, domain.NotifBudget80},
		{100, domain.NotifBudget100},
	}

	for _, limit := range limits {
		expenses, err := s.expenseRepo.GetFiltered(ctx, limit.UserID, &monthStart, nil, 0)
		if err != nil {
			s.log.Error().Err(err).Int64("user_id", limit.UserID).Msg("Failed to fetch expenses for budget check")
			continue
		}

		var spent float64
		for _, e := range expenses {
			spent += e.Amount
		}
		pct := (spent / limit.Amount) * 100

		for _, t := range thresholds {
			if pct < t.pct {
				continue
			}
			already, err := s.notifRepo.WasSent(ctx, limit.UserID, t.notifType, monthKey)
			if err != nil || already {
				continue
			}

			user, err := s.userRepo.FindByID(ctx, limit.UserID)
			if err != nil {
				continue
			}

			msg := s.budgetMsg(t.notifType, spent, limit.Amount, limit.Currency)
			if err := s.notifier.SendMessage(user.TelegramID, msg); err != nil {
				s.log.Error().Err(err).Int64("user_id", limit.UserID).Msg("Failed to send budget alert")
				continue
			}

			_ = s.notifRepo.MarkSent(ctx, limit.UserID, t.notifType, monthKey)
			s.log.Info().Int64("user_id", limit.UserID).Float64("pct", t.pct).Float64("spent", spent).Msg("Budget alert sent")
		}
	}
}

func (s *NotificationService) budgetMsg(t domain.NotificationType, spent, limit float64, currency string) string {
	switch t {
	case domain.NotifBudget50:
		return fmt.Sprintf(
			"💛 *Budget: 50%% used*\n%.2f / %.2f %s spent this month\\.",
			spent, limit, currency)
	case domain.NotifBudget80:
		return fmt.Sprintf(
			"🟠 *Budget warning: 80%% used*\n%.2f / %.2f %s spent — only *%.2f %s* remaining\\.",
			spent, limit, currency, limit-spent, currency)
	default: // 100%
		return fmt.Sprintf(
			"🔴 *Budget limit reached\\!*\n%.2f / %.2f %s spent — *%.2f %s over budget*\\.",
			spent, limit, currency, spent-limit, currency)
	}
}
