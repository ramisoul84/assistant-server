package domain

import "time"

type OTPCode struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	Code      string    `db:"code"`
	Used      bool      `db:"used"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

type AuthClaims struct {
	UserID     int64  `json:"user_id"`
	TelegramID int64  `json:"telegram_id"`
	Handle     string `json:"handle"`
}
