package domain

import "time"

// NotificationType is the kind of notification sent.
type NotificationType string

const (
	NotifAppointment24h NotificationType = "appointment_24h"
	NotifAppointment1h  NotificationType = "appointment_1h"
	NotifBudget50       NotificationType = "budget_50"
	NotifBudget80       NotificationType = "budget_80"
	NotifBudget100      NotificationType = "budget_100"
)

type Notification struct {
	ID     int64            `db:"id"`
	UserID int64            `db:"user_id"`
	Type   NotificationType `db:"type"`
	RefID  string           `db:"ref_id"` // appointment id OR "YYYY-MM"
	SentAt time.Time        `db:"sent_at"`
}

// BudgetLimit is the monthly spending cap a user sets.
type BudgetLimit struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	Amount    float64   `db:"amount"`
	Currency  string    `db:"currency"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
