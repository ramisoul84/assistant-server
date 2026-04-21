package domain

import "time"

// User maps a Telegram user to an internal account.
type User struct {
	ID             int64     `db:"id"          json:"id"`
	TelegramID     int64     `db:"telegram_id" json:"telegram_id"`
	TelegramHandle string    `db:"handle"      json:"handle"`
	FirstName      string    `db:"first_name"  json:"first_name"`
	CreatedAt      time.Time `db:"created_at"  json:"created_at"`
}
