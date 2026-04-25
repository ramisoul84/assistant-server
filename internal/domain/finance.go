package domain

import "time"

type Expense struct {
	ID          int64     `db:"id"          json:"id"`
	UserID      int64     `db:"user_id"     json:"user_id"`
	Amount      float64   `db:"amount"      json:"amount"`
	Currency    string    `db:"currency"    json:"currency"`
	Category    string    `db:"category"    json:"category"`
	Description string    `db:"description" json:"description"`
	HappenedAt  time.Time `db:"happened_at" json:"happened_at"`
	CreatedAt   time.Time `db:"created_at"  json:"created_at"`
}

type Income struct {
	ID          int64     `db:"id"          json:"id"`
	UserID      int64     `db:"user_id"     json:"user_id"`
	Amount      float64   `db:"amount"      json:"amount"`
	Currency    string    `db:"currency"    json:"currency"`
	Category    string    `db:"category"    json:"category"`
	Description string    `db:"description" json:"description"`
	HappenedAt  time.Time `db:"happened_at" json:"happened_at"`
	CreatedAt   time.Time `db:"created_at"  json:"created_at"`
}
