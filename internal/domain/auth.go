package domain

import "time"

// OTPCode is a one-time password record stored in the DB.
type OTPCode struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	Code      string    `db:"code"`
	Used      bool      `db:"used"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

// AuthClaims is embedded in the JWT token.
type AuthClaims struct {
	UserID     int64  `json:"user_id"`
	TelegramID int64  `json:"telegram_id"`
	Handle     string `json:"handle"`
}
