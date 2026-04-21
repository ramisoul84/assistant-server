package domain

import "time"

type Appointment struct {
	ID        int64     `db:"id"         json:"id"`
	UserID    int64     `db:"user_id"    json:"user_id"`
	Title     string    `db:"title"      json:"title"`
	Datetime  time.Time `db:"datetime"   json:"datetime"`
	Notes     string    `db:"notes"      json:"notes"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
