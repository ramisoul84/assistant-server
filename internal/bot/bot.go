package bot

import (
	"context"
	"fmt"
	"strings"
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

func New(
	cfg *config.Config,
	users repository.UserRepository,
	finance repository.FinanceRepository,
	ai service.AIService,
	assistant *service.AssistantService,
) (*Bot, *telegram.Notifier, error) {
	api, err := tgbotapi.NewBotAPI(cfg.Telegram.Token)
	if err != nil {
		return nil, nil, err
	}
	api.Debug = cfg.Telegram.Debug

	log := logger.Get()
	log.Info().Str("username", api.Self.UserName).Msg("Bot authorized")

	_, _ = api.Request(tgbotapi.NewSetMyCommands(
		tgbotapi.BotCommand{Command: "today",  Description: "Today's summary"},
		tgbotapi.BotCommand{Command: "budget", Description: "View or set monthly budget limit"},
		tgbotapi.BotCommand{Command: "help",   Description: "How to use this assistant"},
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
	user, err := b.users.FindOrCreate(ctx, msg.From.ID, msg.From.UserName, msg.From.FirstName, msg.From.LanguageCode)
	if err != nil {
		b.send(msg.Chat.ID, "Something went wrong. Please try again.")
		return
	}

	// ── Silent timezone detection ─────────────────────────────────────────────
	// If the user has no timezone yet, try to detect it from this message
	// without blocking the user or asking any question.
	if user.Timezone == "" || user.Timezone == "UTC" {
		go b.tryDetectTimezone(ctx, user, msg.Text)
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

	// Use the user's local time so relative expressions ("at 16", "tomorrow")
	// are resolved in their timezone, not UTC.
	localNow := user.NowLocal()
	result, err := b.ai.Parse(ctx, msg.Text, localNow)
	if err != nil {
		b.log.Error().Err(err).Msg("AI parse failed")
		b.send(msg.Chat.ID, "Sorry, I couldn't understand that. Try again?")
		return
	}

	if result.Intent == domain.IntentUnknown {
		b.send(msg.Chat.ID,
			"I didn't quite get that. Try:\n"+
				"• *Spent 45€ on groceries*\n"+
				"• *Got paid 2000 EUR*\n"+
				"• *Dentist tomorrow at 3pm*\n"+
				"• *Remember to call mom*")
		return
	}

	if err := b.assistant.Save(ctx, user.ID, result); err != nil {
		b.log.Error().Err(err).Msg("Save failed")
		b.send(msg.Chat.ID, "Something went wrong saving. Please try again.")
		return
	}

	b.send(msg.Chat.ID, result.Reply)
}

// tryDetectTimezone runs in a goroutine — never blocks the message flow.
// It asks the AI to infer a timezone from the message text, validates it,
// and saves it silently. The user never sees this happening.
func (b *Bot) tryDetectTimezone(ctx context.Context, user *domain.User, text string) {
	if text == "" {
		return
	}

	// Fallback: try to guess from Telegram language_code first (free, no API call)
	// This covers the majority of cases without spending tokens.
	if tz := timezoneFromLanguage(user.LanguageCode); tz != "" {
		if err := b.users.SetTimezone(ctx, user.ID, tz); err == nil {
			b.log.Info().
				Int64("user_id", user.ID).
				Str("timezone", tz).
				Str("source", "language_code").
				Msg("timezone detected")
			return
		}
	}

	// Second: use AI to detect from message content (costs ~10 tokens)
	tz := b.ai.DetectTimezone(ctx, text)
	if tz == "" {
		return
	}
	if err := b.users.SetTimezone(ctx, user.ID, tz); err != nil {
		b.log.Warn().Err(err).Str("timezone", tz).Msg("failed to save detected timezone")
		return
	}
	b.log.Info().
		Int64("user_id", user.ID).
		Str("timezone", tz).
		Str("source", "ai").
		Msg("timezone detected from message")
}

// timezoneFromLanguage maps Telegram's language_code (BCP-47) to a primary IANA timezone.
// This covers the most common cases instantly, without an AI call.
// Not exhaustive — for rare codes the AI fallback handles it.
func timezoneFromLanguage(code string) string {
	if code == "" {
		return ""
	}
	// Normalize: "ru-RU" → "ru", "en-US" → "en-US"
	parts := strings.SplitN(strings.ToLower(code), "-", 2)
	lang := parts[0]
	region := ""
	if len(parts) > 1 {
		region = strings.ToUpper(parts[1])
	}

	// Region-specific overrides first (more precise)
	regionMap := map[string]string{
		"en-US": "America/New_York",
		"en-GB": "Europe/London",
		"en-AU": "Australia/Sydney",
		"en-CA": "America/Toronto",
		"en-IN": "Asia/Kolkata",
		"en-AE": "Asia/Dubai",
		"fr-CA": "America/Toronto",
		"zh-TW": "Asia/Taipei",
		"zh-HK": "Asia/Hong_Kong",
		"pt-BR": "America/Sao_Paulo",
		"es-MX": "America/Mexico_City",
		"es-AR": "America/Argentina/Buenos_Aires",
	}
	if region != "" {
		if tz, ok := regionMap[lang+"-"+region]; ok {
			return tz
		}
	}

	// Language-level defaults
	langMap := map[string]string{
		"ru": "Europe/Moscow",
		"uk": "Europe/Kiev",
		"ar": "Asia/Riyadh",
		"tr": "Europe/Istanbul",
		"de": "Europe/Berlin",
		"fr": "Europe/Paris",
		"es": "Europe/Madrid",
		"it": "Europe/Rome",
		"pl": "Europe/Warsaw",
		"nl": "Europe/Amsterdam",
		"pt": "Europe/Lisbon",
		"sv": "Europe/Stockholm",
		"nb": "Europe/Oslo",
		"da": "Europe/Copenhagen",
		"fi": "Europe/Helsinki",
		"cs": "Europe/Prague",
		"sk": "Europe/Bratislava",
		"hu": "Europe/Budapest",
		"ro": "Europe/Bucharest",
		"bg": "Europe/Sofia",
		"hr": "Europe/Zagreb",
		"sr": "Europe/Belgrade",
		"sl": "Europe/Ljubljana",
		"lt": "Europe/Vilnius",
		"lv": "Europe/Riga",
		"et": "Europe/Tallinn",
		"el": "Europe/Athens",
		"he": "Asia/Jerusalem",
		"fa": "Asia/Tehran",
		"hi": "Asia/Kolkata",
		"bn": "Asia/Dhaka",
		"ur": "Asia/Karachi",
		"vi": "Asia/Ho_Chi_Minh",
		"th": "Asia/Bangkok",
		"id": "Asia/Jakarta",
		"ms": "Asia/Kuala_Lumpur",
		"zh": "Asia/Shanghai",
		"ja": "Asia/Tokyo",
		"ko": "Asia/Seoul",
		"ka": "Asia/Tbilisi",
		"az": "Asia/Baku",
		"kk": "Asia/Almaty",
		"uz": "Asia/Tashkent",
	}
	return langMap[lang]
}

func (b *Bot) handleCommand(ctx context.Context, msg *tgbotapi.Message, user *domain.User) {
	switch msg.Command() {

	case "start":
		b.send(msg.Chat.ID, fmt.Sprintf(
			"👋 Hi *%s*!\n\n"+
				"Just write to me naturally:\n\n"+
				"💸 \"Spent 45€ on groceries\"\n"+
				"💰 \"Got paid 3000 EUR salary\"\n"+
				"📅 \"Dentist next Monday 3pm\"\n"+
				"📝 \"Remember to call the bank\"\n\n"+
				"/help for full guide.",
			user.FirstName))

	case "today":
		b.handleToday(ctx, msg, user)

	case "budget":
		b.handleBudget(ctx, msg, user)

	case "help":
		b.send(msg.Chat.ID,
			"📖 *How to use me*\n\n"+
				"*Expenses:*\n"+
				"→ \"Spent 45 on groceries\"\n"+
				"→ \"Coffee 3.50\"\n\n"+
				"*Income:*\n"+
				"→ \"Got paid 3000 EUR\"\n\n"+
				"*Appointments:*\n"+
				"→ \"Dentist Monday at 3pm\"\n"+
				"→ \"Meeting Thursday 10am\"\n\n"+
				"*Notes:*\n"+
				"→ \"Remember to call the bank\"\n\n"+
				"*Commands:*\n"+
				"/today — daily summary\n"+
				"/budget 1500 — set monthly limit\n"+
				"/help — this guide")

	default:
		b.send(msg.Chat.ID, "Type /help to see available commands.")
	}
}

func (b *Bot) handleToday(ctx context.Context, msg *tgbotapi.Message, user *domain.User) {
	loc := user.Location()
	now := time.Now().In(loc)

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	in7 := now.AddDate(0, 0, 7)

	expenses, _ := b.finance.GetExpenses(ctx, user.ID, &monthStart, nil)
	incomes, _  := b.finance.GetIncomes(ctx, user.ID, &monthStart, nil)
	budget, _   := b.finance.GetBudget(ctx, user.ID)
	upNotes, _  := b.assistant.GetUpcomingNotes(ctx, user.ID, now, in7)

	var spent, earned float64
	for _, e := range expenses { spent += e.Amount }
	for _, i := range incomes  { earned += i.Amount }

	txt := fmt.Sprintf("📊 *%s*\n\n", now.Format("Mon, Jan 2"))
	txt += "💰 *This month*\n"
	txt += fmt.Sprintf("  Income:    +%.2f EUR\n", earned)
	txt += fmt.Sprintf("  Expenses:  −%.2f EUR\n", spent)
	txt += fmt.Sprintf("  Net:        %.2f EUR\n", earned-spent)
	if budget != nil {
		pct := (spent / budget.Amount) * 100
		txt += fmt.Sprintf("  Budget:    %s %.0f%%\n", progressBar(pct), pct)
	}

	if len(upNotes) > 0 {
		txt += "\n📅 *Upcoming (7 days)*\n"
		for _, n := range upNotes {
			localTime := n.Datetime.Time.In(loc)
			diff := time.Until(n.Datetime.Time)
			var when string
			if diff < 24*time.Hour {
				when = "today " + localTime.Format("15:04")
			} else {
				when = localTime.Format("Mon Jan 2 · 15:04")
			}
			txt += fmt.Sprintf("  • %s — _%s_\n", n.Content, when)
		}
	}

	b.send(msg.Chat.ID, txt)
}

func (b *Bot) handleBudget(ctx context.Context, msg *tgbotapi.Message, user *domain.User) {
	args := strings.TrimSpace(msg.CommandArguments())
	loc  := user.Location()
	now  := time.Now().In(loc)

	if args == "" {
		bud, err := b.finance.GetBudget(ctx, user.ID)
		if err != nil {
			b.send(msg.Chat.ID, "No budget set.\nUse /budget 1500 to set a monthly limit.")
			return
		}
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		expenses, _ := b.finance.GetExpenses(ctx, user.ID, &monthStart, nil)
		var spent float64
		for _, e := range expenses { spent += e.Amount }
		pct := (spent / bud.Amount) * 100
		b.send(msg.Chat.ID, fmt.Sprintf(
			"💰 *Monthly budget:* %.0f %s\n💸 *Spent:* %.2f (%.0f%%)\n✅ *Remaining:* %.2f",
			bud.Amount, bud.Currency, spent, pct, bud.Amount-spent))
		return
	}

	var amount float64
	currency := "EUR"
	fmt.Sscanf(args, "%f %s", &amount, &currency)
	if amount <= 0 {
		b.send(msg.Chat.ID, "Usage: /budget 1500 or /budget 2000 USD")
		return
	}
	b.finance.UpsertBudget(ctx, user.ID, amount, currency)
	b.send(msg.Chat.ID, fmt.Sprintf(
		"✅ Budget set: *%.0f %s/month*\n\nI'll alert you at 50%%, 80%%, and 100%%.",
		amount, currency))
}

func progressBar(pct float64) string {
	filled := int(pct / 10)
	if filled > 10 { filled = 10 }
	bar := ""
	for i := 0; i < 10; i++ {
		if i < filled { bar += "█" } else { bar += "░" }
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
