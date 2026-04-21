package domain

import "time"

// GymSession is one visit to the gym — contains multiple exercises.
type GymSession struct {
	ID        int64         `db:"id"         json:"id"`
	UserID    int64         `db:"user_id"    json:"user_id"`
	Notes     string        `db:"notes"      json:"notes"`
	StartedAt time.Time     `db:"started_at" json:"started_at"`
	EndedAt   time.Time     `db:"ended_at"   json:"ended_at"`
	CreatedAt time.Time     `db:"created_at" json:"created_at"`
	Exercises []GymExercise `db:"-"          json:"exercises"`
}

// GymExercise is a single exercise within a session.
// Each set can have different weight, so we store per-exercise averages
// for simplicity — good enough for a personal tracker.
type GymExercise struct {
	ID        int64     `db:"id"          json:"id"`
	SessionID int64     `db:"session_id"  json:"session_id"`
	Name      string    `db:"name"        json:"name"`
	Sets      int       `db:"sets"        json:"sets"`
	Reps      int       `db:"reps"        json:"reps"`
	WeightKg  float64   `db:"weight_kg"   json:"weight_kg"`
	Notes     string    `db:"notes"       json:"notes"`
	CreatedAt time.Time `db:"created_at"  json:"created_at"`
}
