package bot

import (
	"context"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ramisoul84/assistant-server/internal/bot/commands"
	"github.com/ramisoul84/assistant-server/internal/bot/handlers"
	"github.com/ramisoul84/assistant-server/internal/bot/state"
	"github.com/ramisoul84/assistant-server/internal/config"
	"github.com/ramisoul84/assistant-server/internal/domain"
	"github.com/ramisoul84/assistant-server/internal/repository"
	"github.com/ramisoul84/assistant-server/internal/service"
	"github.com/ramisoul84/assistant-server/pkg/logger"
	"github.com/ramisoul84/assistant-server/pkg/telegram"
)

type Bot struct {
	api      *tgbotapi.BotAPI
	log      logger.Logger
	userRepo repository.UserRepository
	states   *state.Store

	appointment *handlers.AppointmentHandler
	expense     *handlers.ExpenseHandler

	cmdAppointments *commands.AppointmentsCommand
	cmdExpenses     *commands.ExpensesCommand
	cmdGym          *commands.GymCommand
	cmdSummary      *commands.SummaryCommand

	aiParser service.AIParser
	gymSvc   service.GymService
}

func New(
	cfg *config.Config,
	userRepo repository.UserRepository,
	aiParser service.AIParser,
	appointmentSvc service.AppointmentService,
	expenseSvc service.ExpenseService,
	gymSvc service.GymService,
) (*Bot, *telegram.Notifier, error) {
	api, err := tgbotapi.NewBotAPI(cfg.Telegram.Token)
	if err != nil {
		return nil, nil, err
	}
	api.Debug = cfg.Telegram.Debug

	log := logger.Get()
	log.Info().Str("username", api.Self.UserName).Msg("Telegram bot authorized")

	botCommands := tgbotapi.NewSetMyCommands(
		tgbotapi.BotCommand{Command: "start",        Description: "Welcome message"},
		tgbotapi.BotCommand{Command: "help",         Description: "Show all commands"},
		tgbotapi.BotCommand{Command: "appointments", Description: "Upcoming appointments"},
		tgbotapi.BotCommand{Command: "expenses",     Description: "This month's expenses"},
		tgbotapi.BotCommand{Command: "gym",          Description: "Last gym sessions"},
		tgbotapi.BotCommand{Command: "summary",      Description: "Today's overview"},
		tgbotapi.BotCommand{Command: "gym_start",    Description: "Start a gym session"},
		tgbotapi.BotCommand{Command: "gym_end",      Description: "End current gym session"},
		tgbotapi.BotCommand{Command: "cancel",       Description: "Cancel current action"},
	)
	if _, err := api.Request(botCommands); err != nil {
		log.Warn().Err(err).Msg("Failed to set bot commands menu")
	}

	notifier := telegram.NewNotifier(api)

	b := &Bot{
		api:             api,
		log:             log,
		userRepo:        userRepo,
		states:          state.NewStore(),
		aiParser:        aiParser,
		gymSvc:          gymSvc,
		appointment:     handlers.NewAppointmentHandler(appointmentSvc),
		expense:         handlers.NewExpenseHandler(expenseSvc),
		cmdAppointments: commands.NewAppointmentsCommand(appointmentSvc),
		cmdExpenses:     commands.NewExpensesCommand(expenseSvc),
		cmdGym:          commands.NewGymCommand(gymSvc),
		cmdSummary:      commands.NewSummaryCommand(appointmentSvc, expenseSvc, gymSvc),
	}
	return b, notifier, nil
}

func (b *Bot) Start(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.api.GetUpdatesChan(u)
	b.log.Info().Msg("Bot update loop started")

	for {
		select {
		case <-ctx.Done():
			b.log.Info().Msg("Bot update loop stopped")
			b.api.StopReceivingUpdates()
			return
		case update, ok := <-updates:
			if !ok {
				return
			}
			if update.Message == nil {
				continue
			}
			go b.handleMessage(ctx, update.Message)
		}
	}
}

func (b *Bot) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	log := b.log.WithFields(map[string]any{
		"telegram_id": msg.From.ID,
		"chat_id":     msg.Chat.ID,
		"text":        msg.Text,
	})

	user, err := b.userRepo.FindOrCreate(ctx, msg.From.ID, msg.From.UserName, msg.From.FirstName)
	if err != nil {
		log.Error().Err(err).Msg("Failed to resolve user")
		b.reply(msg, "⚠️ Something went wrong\\. Please try again\\.")
		return
	}

	if msg.IsCommand() {
		b.handleCommand(ctx, msg, user)
		return
	}
	if msg.Text == "" {
		b.reply(msg, "I can only process text messages for now\\.")
		return
	}

	conv := b.states.Get(msg.From.ID)
	switch conv.Stage {
	case state.StageGymSessionOpen:
		b.handleGymExercise(ctx, msg, user, conv)
	case state.StageAwaitingCompletion:
		b.handleCompletion(ctx, msg, user, conv)
	default:
		b.handleIdle(ctx, msg, user)
	}
}

func (b *Bot) handleIdle(ctx context.Context, msg *tgbotapi.Message, user *domain.User) {
	b.sendTyping(msg.Chat.ID)
	parsed, err := b.aiParser.Parse(ctx, msg.Text, time.Now().UTC())
	if err != nil {
		b.log.Error().Err(err).Msg("AI parser failed")
		b.reply(msg, "🤖 I had trouble understanding that\\. Could you rephrase?")
		return
	}

	switch parsed.Intent {
	case domain.IntentAddAppointment:
		replyText, err := b.appointment.Handle(ctx, user.ID, parsed)
		if err != nil {
			b.reply(msg, "❌ Failed to save appointment\\. Please try again\\.")
			return
		}
		b.reply(msg, escMD(replyText))

	case domain.IntentAddExpense:
		replyText, err := b.expense.Handle(ctx, user.ID, parsed)
		if err != nil {
			b.reply(msg, "❌ Failed to save expense\\. Please try again\\.")
			return
		}
		b.reply(msg, escMD(replyText))

	case domain.IntentIncomplete:
		b.states.Set(msg.From.ID, &state.Conversation{
			Stage:         state.StageAwaitingCompletion,
			IncompleteFor: state.IncompleteIntent(parsed.Incomplete),
		})
		b.reply(msg, escMD(parsed.Reply))

	default:
		b.reply(msg, "To add data just type naturally:\n"+
			"• \"Dentist appointment tomorrow at 3pm\"\n"+
			"• \"Spent 40€ on groceries\"\n\n"+
			"To view data use commands — type /help")
	}
}

func (b *Bot) handleCompletion(ctx context.Context, msg *tgbotapi.Message, user *domain.User, conv *state.Conversation) {
	b.sendTyping(msg.Chat.ID)
	category := string(conv.IncompleteFor)
	parsed, err := b.aiParser.ParseCompletion(ctx, category, msg.Text, time.Now().UTC())
	if err != nil {
		b.reply(msg, "🤖 I couldn't understand that\\. Please try again\\.")
		return
	}

	var replyText string
	switch parsed.Intent {
	case domain.IntentAddAppointment:
		replyText, err = b.appointment.Handle(ctx, user.ID, parsed)
	case domain.IntentAddExpense:
		replyText, err = b.expense.Handle(ctx, user.ID, parsed)
	default:
		b.reply(msg, escMD(parsed.Reply))
		return
	}
	if err != nil {
		b.reply(msg, "❌ Failed to save\\. Please try again\\.")
		return
	}
	b.states.Reset(msg.From.ID)
	b.reply(msg, escMD(replyText))
}

func (b *Bot) handleGymExercise(ctx context.Context, msg *tgbotapi.Message, user *domain.User, conv *state.Conversation) {
	text := msg.Text
	if text == "done" || text == "Done" || text == "finish" || text == "end" {
		b.closeGymSession(ctx, msg, conv)
		return
	}
	b.sendTyping(msg.Chat.ID)
	parsed, err := b.aiParser.ParseExercise(ctx, text)
	if err != nil {
		b.reply(msg, "🤖 Couldn't parse that\\. Try: 'bench press 4x10 80kg'")
		return
	}
	if parsed.Intent != domain.IntentAddGymExercise || parsed.GymExercise == nil {
		b.reply(msg, escMD(parsed.Reply))
		return
	}
	ex, err := b.gymSvc.AddExercise(ctx, conv.GymSessionID, parsed.GymExercise)
	if err != nil {
		b.reply(msg, "❌ Failed to save exercise\\. Try again\\.")
		return
	}
	conv.GymExercises = append(conv.GymExercises, state.GymExercise{
		Name: ex.Name, Sets: ex.Sets, Reps: ex.Reps, WeightKg: ex.WeightKg,
	})
	b.states.Set(msg.From.ID, conv)

	line := fmt.Sprintf("✅ *%s* logged", escMD(ex.Name))
	if ex.Sets > 0 {
		line += fmt.Sprintf(" — %d×%d @ %.1fkg", ex.Sets, ex.Reps, ex.WeightKg)
	}
	line += fmt.Sprintf("\n\nNext exercise? Or /gym\\_end to finish \\(%d logged\\)", len(conv.GymExercises))
	b.reply(msg, line)
}

func (b *Bot) closeGymSession(ctx context.Context, msg *tgbotapi.Message, conv *state.Conversation) {
	if len(conv.GymExercises) == 0 {
		b.states.Reset(msg.From.ID)
		b.reply(msg, "Session cancelled — no exercises logged\\.")
		return
	}
	if err := b.gymSvc.CloseSession(ctx, conv.GymSessionID); err != nil {
		b.reply(msg, "❌ Failed to close session\\.")
		return
	}
	duration := time.Since(conv.GymStartedAt).Round(time.Minute)
	summary := fmt.Sprintf("💪 *Session complete\\!* \\(%s\\)\n\n", escMD(duration.String()))
	for _, ex := range conv.GymExercises {
		line := fmt.Sprintf("• *%s*", escMD(ex.Name))
		if ex.Sets > 0 {
			line += fmt.Sprintf(" — %d×%d @ %.1fkg", ex.Sets, ex.Reps, ex.WeightKg)
		}
		summary += line + "\n"
	}
	summary += fmt.Sprintf("\n_%d exercise\\(s\\) saved_ 🏋️", len(conv.GymExercises))
	b.states.Reset(msg.From.ID)
	b.reply(msg, summary)
}

func (b *Bot) handleCommand(ctx context.Context, msg *tgbotapi.Message, user *domain.User) {
	var replyText string
	var err error
	switch msg.Command() {
	case "start":
		replyText = b.cmdStart(user)
	case "help":
		replyText = cmdHelp()
	case "appointments", "a":
		replyText, err = b.cmdAppointments.Handle(ctx, user.ID, msg.CommandArguments())
	case "expenses", "e":
		replyText, err = b.cmdExpenses.Handle(ctx, user.ID, msg.CommandArguments())
	case "gym", "g":
		replyText, err = b.cmdGym.Handle(ctx, user.ID, msg.CommandArguments())
	case "summary", "s":
		replyText, err = b.cmdSummary.Handle(ctx, user.ID)
	case "gym_start":
		b.startGymSession(ctx, msg, user)
		return
	case "gym_end":
		conv := b.states.Get(msg.From.ID)
		if conv.Stage != state.StageGymSessionOpen {
			b.reply(msg, "No active gym session\\. Start one with /gym\\_start")
			return
		}
		b.closeGymSession(ctx, msg, conv)
		return
	case "cancel":
		b.states.Reset(msg.From.ID)
		b.reply(msg, "✅ Cancelled\\. Back to normal\\.")
		return
	default:
		replyText = "Unknown command\\. Type /help"
	}
	if err != nil {
		b.log.Error().Err(err).Str("command", msg.Command()).Msg("Command failed")
		b.reply(msg, "❌ Failed to retrieve data\\. Please try again\\.")
		return
	}
	b.reply(msg, replyText)
}

func (b *Bot) startGymSession(ctx context.Context, msg *tgbotapi.Message, user *domain.User) {
	b.states.Reset(msg.From.ID)
	session, err := b.gymSvc.StartSession(ctx, user.ID)
	if err != nil {
		b.reply(msg, "❌ Failed to start session\\.")
		return
	}
	b.states.Set(msg.From.ID, &state.Conversation{
		Stage:        state.StageGymSessionOpen,
		GymSessionID: session.ID,
		GymStartedAt: session.StartedAt,
	})
	b.reply(msg, "💪 *Gym session started\\!*\n\n"+
		"Send each exercise:\n"+
		"• `bench press 4x10 80kg`\n"+
		"• `squat 5x5 100kg`\n"+
		"• `pull ups 3x12` \\(bodyweight\\)\n\n"+
		"Type /gym\\_end or 'done' to finish\\.")
}

func (b *Bot) cmdStart(user *domain.User) string {
	return fmt.Sprintf("👋 Hi *%s*\\!\n\n"+
		"*Add data:*\n"+
		"• \"Dentist tomorrow at 3pm\"\n"+
		"• \"Spent 40€ on groceries\"\n\n"+
		"*Gym:* /gym\\_start → log exercises → /gym\\_end\n\n"+
		"*View:* /appointments · /expenses · /gym · /summary", escMD(user.FirstName))
}

func cmdHelp() string {
	return "📋 *Commands*\n\n" +
		"*/appointments* \\[week|month|all\\]\n" +
		"*/expenses* \\[week|month|all\\]\n" +
		"*/gym* \\[N\\] — last N sessions\n" +
		"*/summary* — today's overview\n" +
		"*/gym\\_start* — begin session\n" +
		"*/gym\\_end* — finish session\n" +
		"*/cancel* — cancel current action"
}

func (b *Bot) reply(msg *tgbotapi.Message, text string) {
	m := tgbotapi.NewMessage(msg.Chat.ID, text)
	m.ParseMode = tgbotapi.ModeMarkdownV2
	if _, err := b.api.Send(m); err != nil {
		m.ParseMode = ""
		if _, err2 := b.api.Send(m); err2 != nil {
			b.log.Error().Err(err2).Msg("Failed to send reply")
		}
	}
}

func (b *Bot) sendTyping(chatID int64) {
	_, _ = b.api.Send(tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping))
}

func escMD(s string) string {
	replacer := []struct{ from, to string }{
		{"_", "\\_"}, {"*", "\\*"}, {"[", "\\["}, {"]", "\\]"},
		{"(", "\\("}, {")", "\\)"}, {"~", "\\~"}, {"`", "\\`"},
		{">", "\\>"}, {"#", "\\#"}, {"+", "\\+"}, {"-", "\\-"},
		{"=", "\\="}, {"|", "\\|"}, {"{", "\\{"}, {"}", "\\}"},
		{".", "\\."}, {"!", "\\!"},
	}
	for _, r := range replacer {
		result := ""
		for i := 0; i < len(s); {
			if i+len(r.from) <= len(s) && s[i:i+len(r.from)] == r.from {
				result += r.to
				i += len(r.from)
			} else {
				result += string(s[i])
				i++
			}
		}
		s = result
	}
	return s
}
