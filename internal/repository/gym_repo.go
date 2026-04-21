package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/ramisoul84/assistant-server/internal/domain"
)

type GymRepository interface {
	CreateSession(ctx context.Context, userID int64, startedAt time.Time) (*domain.GymSession, error)
	CloseSession(ctx context.Context, sessionID int64, endedAt time.Time) error
	DeleteSession(ctx context.Context, sessionID, userID int64) error
	AddExercise(ctx context.Context, sessionID int64, ex *domain.GymExercise) (*domain.GymExercise, error)
	UpdateExercise(ctx context.Context, id, userID int64, name string, sets, reps int, weightKg float64, notes string) (*domain.GymExercise, error)
	DeleteExercise(ctx context.Context, id, userID int64) error
	GetSessionsWithExercises(ctx context.Context, userID int64, from, to *time.Time, limit int) ([]domain.GymSession, error)
}

type gymRepo struct{ db *sqlx.DB }

func NewGymRepository(db *sqlx.DB) GymRepository {
	return &gymRepo{db: db}
}

func (r *gymRepo) CreateSession(ctx context.Context, userID int64, startedAt time.Time) (*domain.GymSession, error) {
	const q = `
		INSERT INTO gym_sessions (user_id, started_at, ended_at)
		VALUES ($1, $2, $2)
		RETURNING id, user_id, notes, started_at, ended_at, created_at`
	var s domain.GymSession
	if err := r.db.GetContext(ctx, &s, q, userID, startedAt); err != nil {
		return nil, fmt.Errorf("gymRepo.CreateSession: %w", err)
	}
	return &s, nil
}

func (r *gymRepo) CloseSession(ctx context.Context, sessionID int64, endedAt time.Time) error {
	const q = `UPDATE gym_sessions SET ended_at=$1 WHERE id=$2`
	if _, err := r.db.ExecContext(ctx, q, endedAt, sessionID); err != nil {
		return fmt.Errorf("gymRepo.CloseSession: %w", err)
	}
	return nil
}

func (r *gymRepo) DeleteSession(ctx context.Context, sessionID, userID int64) error {
	const q = `DELETE FROM gym_sessions WHERE id=$1 AND user_id=$2`
	if _, err := r.db.ExecContext(ctx, q, sessionID, userID); err != nil {
		return fmt.Errorf("gymRepo.DeleteSession: %w", err)
	}
	return nil
}

func (r *gymRepo) AddExercise(ctx context.Context, sessionID int64, ex *domain.GymExercise) (*domain.GymExercise, error) {
	const q = `
		INSERT INTO gym_exercises (session_id, name, sets, reps, weight_kg, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, session_id, name, sets, reps, weight_kg, notes, created_at`
	if err := r.db.GetContext(ctx, ex, q,
		sessionID, ex.Name, ex.Sets, ex.Reps, ex.WeightKg, ex.Notes,
	); err != nil {
		return nil, fmt.Errorf("gymRepo.AddExercise: %w", err)
	}
	return ex, nil
}

func (r *gymRepo) UpdateExercise(ctx context.Context, id, userID int64, name string, sets, reps int, weightKg float64, notes string) (*domain.GymExercise, error) {
	// userID check via JOIN to ensure ownership
	const q = `
		UPDATE gym_exercises ge
		SET name=$1, sets=$2, reps=$3, weight_kg=$4, notes=$5
		FROM gym_sessions gs
		WHERE ge.id=$6 AND ge.session_id=gs.id AND gs.user_id=$7
		RETURNING ge.id, ge.session_id, ge.name, ge.sets, ge.reps, ge.weight_kg, ge.notes, ge.created_at`
	var ex domain.GymExercise
	if err := r.db.GetContext(ctx, &ex, q, name, sets, reps, weightKg, notes, id, userID); err != nil {
		return nil, fmt.Errorf("gymRepo.UpdateExercise: %w", err)
	}
	return &ex, nil
}

func (r *gymRepo) DeleteExercise(ctx context.Context, id, userID int64) error {
	const q = `
		DELETE FROM gym_exercises ge
		USING gym_sessions gs
		WHERE ge.id=$1 AND ge.session_id=gs.id AND gs.user_id=$2`
	if _, err := r.db.ExecContext(ctx, q, id, userID); err != nil {
		return fmt.Errorf("gymRepo.DeleteExercise: %w", err)
	}
	return nil
}

func (r *gymRepo) GetSessionsWithExercises(ctx context.Context, userID int64, from, to *time.Time, limit int) ([]domain.GymSession, error) {
	q := `SELECT id, user_id, notes, started_at, ended_at, created_at
		  FROM gym_sessions WHERE user_id=$1`
	args := []any{userID}
	i := 2
	if from != nil {
		q += fmt.Sprintf(" AND started_at >= $%d", i); args = append(args, *from); i++
	}
	if to != nil {
		q += fmt.Sprintf(" AND started_at <= $%d", i); args = append(args, *to); i++
	}
	q += " ORDER BY started_at DESC"
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT $%d", i); args = append(args, limit)
	}

	var sessions []domain.GymSession
	if err := r.db.SelectContext(ctx, &sessions, q, args...); err != nil {
		return nil, fmt.Errorf("gymRepo.GetSessions: %w", err)
	}
	if len(sessions) == 0 {
		return sessions, nil
	}

	sessionIDs := make([]int64, len(sessions))
	for i, s := range sessions {
		sessionIDs[i] = s.ID
	}

	exQuery, exArgs, err := sqlx.In(
		`SELECT id, session_id, name, sets, reps, weight_kg, notes, created_at
		 FROM gym_exercises WHERE session_id IN (?) ORDER BY created_at ASC`,
		sessionIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("gymRepo.buildExQuery: %w", err)
	}
	exQuery = r.db.Rebind(exQuery)

	var exercises []domain.GymExercise
	if err := r.db.SelectContext(ctx, &exercises, exQuery, exArgs...); err != nil {
		return nil, fmt.Errorf("gymRepo.GetExercises: %w", err)
	}

	exMap := make(map[int64][]domain.GymExercise)
	for _, ex := range exercises {
		exMap[ex.SessionID] = append(exMap[ex.SessionID], ex)
	}
	for i := range sessions {
		sessions[i].Exercises = exMap[sessions[i].ID]
	}
	return sessions, nil
}
