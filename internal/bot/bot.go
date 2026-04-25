package bot

import (
	"context"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ramisoul84/assistant-server/internal/config"
	"github.com/ramisoul84/assistant-server/internal/domain"
	"github.com/ramisoul84/assistant-server/internal/repository"
	"github.com/ramisoul84/assistant-server/internal/service"
	"github.com/ramisoul84/assistant-server/pkg/logger"
	"github.com/ramisoul84/assistant-server/pkg/telegram"
)

type Bot struct {
	api       *tgbotapi.BotAPI
	log       logger.Logger
	users     repository.UserRepository
	finance   repository.FinanceRepository
	ai        service.AIService
	assistant *service.AssistantService
}

func New(cfg *config.Config, users repository.UserRepository, finance repository.FinanceRepository, ai service.AIService, assistant *service.AssistantService) (*Bot, *telegram.Notifier, error) {
	api, err := tgbotapi.NewBotAPI(cfg.Telegram.Token)
	if err != nil {
		return nil, nil, err
	}
	api.Debug = cfg.Telegram.Debug

	log := logger.Get()
	log.Info().Str("username", api.Self.UserName).Msg("Bot authorized")

	_, _ = api.Request(tgbotapi.NewSetMyCommands(
		tgbotapi.BotCommand{Command: "today", Description: "Today's summary"},
		tgbotapi.BotCommand{Command: "budget", Description: "View or set monthly budget limit"},
		tgbotapi.BotCommand{Command: "help", Description: "How to use this assistant"},
	))

	return &Bot{api: api, log: log, users: users, finance: finance, ai: ai, assistant: assistant},
		telegram.NewNotifier(api), nil
}

func (b *Bot) Start(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.api.GetUpdatesChan(u)
	b.log.Info().Msg("Bot update loop started")

	for {
		select {
		case <-ctx.Done():
			b.api.StopReceivingUpdates()
			return
		case upd, ok := <-updates:
			if !ok {
				return
			}
			if upd.Message == nil {
				continue
			}
			go b.handle(ctx, upd.Message)
		}
	}
}

func (b *Bot) handle(ctx context.Context, msg *tgbotapi.Message) {
	user, err := b.users.FindOrCreate(ctx, msg.From.ID, msg.From.UserName, msg.From.FirstName)
	if err != nil {
		b.send(msg.Chat.ID, "Something went wrong. Please try again.")
		return
	}

	if msg.IsCommand() {
		b.handleCommand(ctx, msg, user)
		return
	}
	if msg.Text == "" {
		b.send(msg.Chat.ID, "Send me text and I'll save it.")
		return
	}

	b.api.Send(tgbotapi.NewChatAction(msg.Chat.ID, tgbotapi.ChatTyping))

	result, err := b.ai.Parse(ctx, msg.Text, time.Now().UTC())
	if err != nil {
		b.log.Error().Err(err).Msg("AI parse failed")
		b.send(msg.Chat.ID, "Sorry, I couldn't understand that. Try again?")
		return
	}

	if result.Intent == domain.IntentUnknown {
		b.send(msg.Chat.ID, "I didn't quite get that. Try:\n• *Spent 45€ on groceries*\n• *Got paid 2000 EUR*\n• *Dentist tomorrow at 3pm*\n• *Remember to call mom*")
		return
	}

	if err := b.assistant.Save(ctx, user.ID, result); err != nil {
		b.log.Error().Err(err).Msg("Save failed")
		b.send(msg.Chat.ID, "Something went wrong saving. Please try again.")
		return
	}

	b.send(msg.Chat.ID, result.Reply)
}

func (b *Bot) handleCommand(ctx context.Context, msg *tgbotapi.Message, user *domain.User) {
	switch msg.Command() {
	case "start":
		b.send(msg.Chat.ID, fmt.Sprintf(
			"👋 Hi *%s*!\n\nJust write to me naturally:\n\n💸 \"Spent 45€ on groceries\"\n💰 \"Got paid 3000 EUR salary\"\n📅 \"Dentist next Monday 3pm\"\n📝 \"Remember to call the bank\"\n\nI'll figure out the rest and save it. /help for details.", user.FirstName))

	case "today":
		now := time.Now().UTC()
		ms := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		in7 := now.AddDate(0, 0, 7)

		expenses, _ := b.finance.GetExpenses(ctx, user.ID, &ms, nil)
		incomes, _ := b.finance.GetIncomes(ctx, user.ID, &ms, nil)
		budget, _ := b.finance.GetBudget(ctx, user.ID)
		upNotes, _ := b.assistant.GetUpcomingNotes(ctx, user.ID, now, in7)

		var spent, earned float64
		for _, e := range expenses {
			spent += e.Amount
		}
		for _, i := range incomes {
			earned += i.Amount
		}

		txt := fmt.Sprintf("📊 *%s*\n\n", now.Format("Mon, Jan 2"))
		txt += "💰 *This month*\n"
		txt += fmt.Sprintf("  Income:    +%.2f EUR\n", earned)
		txt += fmt.Sprintf("  Expenses:  −%.2f EUR\n", spent)
		txt += fmt.Sprintf("  Net:        %.2f EUR\n", earned-spent)
		if budget != nil {
			pct := (spent / budget.Amount) * 100
			bar := progressBar(pct)
			txt += fmt.Sprintf("  Budget:    %s %.0f%%\n", bar, pct)
		}

		if len(upNotes) > 0 {
			txt += "\n📅 *Upcoming (7 days)*\n"
			for _, n := range upNotes {
				diff := time.Until(n.Datetime.Time)
				var when string
				if diff < 24*time.Hour {
					when = "today " + n.Datetime.Time.Format("15:04")
				} else {
					when = n.Datetime.Time.Format("Mon Jan 2 · 15:04")
				}
				txt += fmt.Sprintf("  • %s — _%s_\n", n.Content, when)
			}
		}

		b.send(msg.Chat.ID, txt)

	case "budget":
		args := msg.CommandArguments()
		if args == "" {
			bud, err := b.finance.GetBudget(ctx, user.ID)
			if err != nil {
				b.send(msg.Chat.ID, "No budget set.\nUse /budget 1500 to set a monthly limit.")
			} else {
				now := time.Now().UTC()
				ms := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
				expenses, _ := b.finance.GetExpenses(ctx, user.ID, &ms, nil)
				var spent float64
				for _, e := range expenses {
					spent += e.Amount
				}
				pct := (spent / bud.Amount) * 100
				b.send(msg.Chat.ID, fmt.Sprintf("💰 *Monthly budget:* %.0f %s\n💸 *Spent:* %.2f (%.0f%%)\n✅ *Remaining:* %.2f",
					bud.Amount, bud.Currency, spent, pct, bud.Amount-spent))
			}
		} else {
			var amount float64
			currency := "EUR"
			fmt.Sscanf(args, "%f %s", &amount, &currency)
			if amount <= 0 {
				b.send(msg.Chat.ID, "Usage: /budget 1500 or /budget 2000 USD")
				return
			}
			b.finance.UpsertBudget(ctx, user.ID, amount, currency)
			b.send(msg.Chat.ID, fmt.Sprintf("✅ Budget set: *%.0f %s/month*\n\nI'll alert you at 50%%, 80%%, and 100%%.", amount, currency))
		}

	case "help":
		b.send(msg.Chat.ID, "📖 *How to use me*\n\n*Expenses:*\n→ \"Spent 45 on groceries\"\n→ \"Coffee 3.50\"\n→ \"Paid rent 800€\"\n\n*Income:*\n→ \"Got paid 3000 EUR\"\n→ \"500 freelance project\"\n\n*Appointments:*\n→ \"Dentist Monday at 3pm\"\n→ \"Meeting Thursday 10am\"\n\n*Notes:*\n→ \"Remember to call the bank\"\n→ \"Buy milk\"\n\n*Commands:*\n/today — daily summary\n/budget 1500 — set monthly budget limit\n/help — this guide")

	default:
		b.send(msg.Chat.ID, "Type /help to see available commands.")
	}
}

func progressBar(pct float64) string {
	filled := int(pct / 10)
	if filled > 10 {
		filled = 10
	}
	bar := ""
	for i := 0; i < 10; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	return bar
}

func (b *Bot) send(chatID int64, text string) {
	m := tgbotapi.NewMessage(chatID, text)
	m.ParseMode = tgbotapi.ModeMarkdown
	if _, err := b.api.Send(m); err != nil {
		m.ParseMode = ""
		b.api.Send(m)
	}
}
