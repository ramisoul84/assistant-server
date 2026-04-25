package domain

import "time"

type Intent string

const (
	IntentSaveExpense Intent = "save_expense"
	IntentSaveIncome  Intent = "save_income"
	IntentSaveNote    Intent = "save_note"
	IntentUnknown     Intent = "unknown"
)

type AIResult struct {
	Intent  Intent     `json:"intent"`
	Expense *ExpenseAI `json:"expense,omitempty"`
	Income  *IncomeAI  `json:"income,omitempty"`
	Note    *NoteAI    `json:"note,omitempty"`
	Reply   string     `json:"reply"`
}

type ExpenseAI struct {
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	HappenedAt  time.Time `json:"happened_at"`
}

type IncomeAI struct {
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	HappenedAt  time.Time `json:"happened_at"`
}

type NoteAI struct {
	Content  string     `json:"content"`
	Datetime *time.Time `json:"datetime,omitempty"`
	Tags     []string   `json:"tags,omitempty"`
}
