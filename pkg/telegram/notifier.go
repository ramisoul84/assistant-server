package telegram

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Notifier sends messages to a specific Telegram chat.
// It implements service.Notifier so the auth service can send OTPs
// without depending on the full bot package.
type Notifier struct {
	api *tgbotapi.BotAPI
}

func NewNotifier(api *tgbotapi.BotAPI) *Notifier {
	return &Notifier{api: api}
}

func (n *Notifier) SendMessage(telegramID int64, text string) error {
	msg := tgbotapi.NewMessage(telegramID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	if _, err := n.api.Send(msg); err != nil {
		return fmt.Errorf("telegram.Notifier: %w", err)
	}
	return nil
}
