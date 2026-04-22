package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/ramisoul84/assistant-server/internal/domain"
)

type AppointmentRepository interface {
	Create(ctx context.Context, a *domain.Appointment) (*domain.Appointment, error)
	Update(ctx context.Context, id, userID int64, title string, datetime time.Time, notes string) (*domain.Appointment, error)
	Delete(ctx context.Context, id, userID int64) error
	GetFiltered(ctx context.Context, userID int64, from, to *time.Time, limit int) ([]domain.Appointment, error)
	// GetInWindow returns ALL users' appointments in a time window — used by the notification service.
	GetInWindow(ctx context.Context, from, to time.Time) ([]domain.Appointment, error)
}

type appointmentRepo struct{ db *sqlx.DB }

func NewAppointmentRepository(db *sqlx.DB) AppointmentRepository {
	return &appointmentRepo{db: db}
}

func (r *appointmentRepo) Create(ctx context.Context, a *domain.Appointment) (*domain.Appointment, error) {
	const q = `
		INSERT INTO appointments (user_id, title, datetime, notes)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, title, datetime, notes, created_at`
	if err := r.db.GetContext(ctx, a, q, a.UserID, a.Title, a.Datetime, a.Notes); err != nil {
		return nil, fmt.Errorf("appointmentRepo.Create: %w", err)
	}
	return a, nil
}

func (r *appointmentRepo) Update(ctx context.Context, id, userID int64, title string, datetime time.Time, notes string) (*domain.Appointment, error) {
	const q = `
		UPDATE appointments SET title=$1, datetime=$2, notes=$3
		WHERE id=$4 AND user_id=$5
		RETURNING id, user_id, title, datetime, notes, created_at`
	var a domain.Appointment
	if err := r.db.GetContext(ctx, &a, q, title, datetime, notes, id, userID); err != nil {
		return nil, fmt.Errorf("appointmentRepo.Update: %w", err)
	}
	return &a, nil
}

func (r *appointmentRepo) Delete(ctx context.Context, id, userID int64) error {
	const q = `DELETE FROM appointments WHERE id=$1 AND user_id=$2`
	if _, err := r.db.ExecContext(ctx, q, id, userID); err != nil {
		return fmt.Errorf("appointmentRepo.Delete: %w", err)
	}
	return nil
}

func (r *appointmentRepo) GetFiltered(ctx context.Context, userID int64, from, to *time.Time, limit int) ([]domain.Appointment, error) {
	q := `SELECT id, user_id, title, datetime, notes, created_at FROM appointments WHERE user_id=$1`
	args := []any{userID}
	i := 2
	if from != nil {
		q += fmt.Sprintf(" AND datetime >= $%d", i); args = append(args, *from); i++
	}
	if to != nil {
		q += fmt.Sprintf(" AND datetime <= $%d", i); args = append(args, *to); i++
	}
	q += " ORDER BY datetime ASC"
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT $%d", i); args = append(args, limit)
	}
	var list []domain.Appointment
	if err := r.db.SelectContext(ctx, &list, q, args...); err != nil {
		return nil, fmt.Errorf("appointmentRepo.GetFiltered: %w", err)
	}
	return list, nil
}

func (r *appointmentRepo) GetInWindow(ctx context.Context, from, to time.Time) ([]domain.Appointment, error) {
	const q = `
		SELECT id, user_id, title, datetime, notes, created_at
		FROM appointments
		WHERE datetime >= $1 AND datetime <= $2
		ORDER BY datetime ASC`
	var list []domain.Appointment
	if err := r.db.SelectContext(ctx, &list, q, from, to); err != nil {
		return nil, fmt.Errorf("appointmentRepo.GetInWindow: %w", err)
	}
	return list, nil
}
