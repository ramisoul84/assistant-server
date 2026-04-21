package domain

import "time"

type IntentType string

const (
	IntentAddAppointment IntentType = "add_appointment"
	IntentAddExpense     IntentType = "add_expense"
	IntentAddGymExercise IntentType = "add_gym_exercise" // single exercise line during open session
	IntentIncomplete     IntentType = "incomplete"       // AI detected intent but missing required fields
	IntentUnknown        IntentType = "unknown"
)

type AIResponse struct {
	Intent      IntentType         `json:"intent"`
	Incomplete  string             `json:"incomplete,omitempty"` // "appointment" | "expense"
	Appointment *AppointmentIntent `json:"appointment,omitempty"`
	Expense     *ExpenseIntent     `json:"expense,omitempty"`
	GymExercise *GymExerciseIntent `json:"gym_exercise,omitempty"`
	Reply       string             `json:"reply"`
}

type AppointmentIntent struct {
	Title    string    `json:"title"`
	Datetime time.Time `json:"datetime"`
	Notes    string    `json:"notes"`
}

type ExpenseIntent struct {
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	SpentAt     time.Time `json:"spent_at"`
}

// GymExerciseIntent is one exercise line parsed during an open session.
type GymExerciseIntent struct {
	Name     string  `json:"name"`
	Sets     int     `json:"sets"`
	Reps     int     `json:"reps"`
	WeightKg float64 `json:"weight_kg"`
	Notes    string  `json:"notes"`
}

type contextKey string

const RequestIDKey contextKey = "request_id"
