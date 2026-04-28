package domain

import "time"

type User struct {
	ID         int64     `db:"id"          json:"id"`
	TelegramID int64     `db:"telegram_id" json:"telegram_id"`
	Handle     string    `db:"handle"      json:"handle"`
	FirstName  string    `db:"first_name"  json:"first_name"`
	Timezone   string    `db:"timezone"    json:"timezone"` // IANA e.g. "Europe/Moscow"
	CreatedAt  time.Time `db:"created_at"  json:"created_at"`
}

// Location parses the user's IANA timezone string into a *time.Location.
// Falls back to UTC if the timezone is invalid or empty.
func (u *User) Location() *time.Location {
	if u.Timezone == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

// NowLocal returns the current time in the user's timezone.
func (u *User) NowLocal() time.Time {
	return time.Now().In(u.Location())
}
