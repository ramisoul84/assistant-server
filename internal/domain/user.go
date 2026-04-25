package domain

import "time"

type User struct {
	ID         int64     `db:"id"          json:"id"`
	TelegramID int64     `db:"telegram_id" json:"telegram_id"`
	Handle     string    `db:"handle"      json:"handle"`
	FirstName  string    `db:"first_name"  json:"first_name"`
	CreatedAt  time.Time `db:"created_at"  json:"created_at"`
}
