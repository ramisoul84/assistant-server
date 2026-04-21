package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ramisoul84/assistant-server/internal/domain"
	"github.com/ramisoul84/assistant-server/internal/repository"
)

type AppointmentInput struct {
	Title    string
	Datetime time.Time
	Notes    string
}

type AppointmentService interface {
	Create(ctx context.Context, userID int64, input *AppointmentInput) (*domain.Appointment, error)
	Update(ctx context.Context, id, userID int64, input *AppointmentInput) (*domain.Appointment, error)
	Delete(ctx context.Context, id, userID int64) error
	GetFiltered(ctx context.Context, userID int64, from, to *time.Time, limit int) ([]domain.Appointment, error)
}

type appointmentService struct{ repo repository.AppointmentRepository }

func NewAppointmentService(repo repository.AppointmentRepository) AppointmentService {
	return &appointmentService{repo: repo}
}

func (s *appointmentService) Create(ctx context.Context, userID int64, input *AppointmentInput) (*domain.Appointment, error) {
	if input.Datetime.IsZero() {
		return nil, fmt.Errorf("%w: datetime is required", domain.ErrInvalidInput)
	}
	title := input.Title
	if title == "" {
		title = "Appointment on " + input.Datetime.Format("Jan 2 at 15:04")
	}
	return s.repo.Create(ctx, &domain.Appointment{
		UserID: userID, Title: title, Datetime: input.Datetime, Notes: input.Notes,
	})
}

func (s *appointmentService) Update(ctx context.Context, id, userID int64, input *AppointmentInput) (*domain.Appointment, error) {
	if input.Datetime.IsZero() {
		return nil, fmt.Errorf("%w: datetime is required", domain.ErrInvalidInput)
	}
	return s.repo.Update(ctx, id, userID, input.Title, input.Datetime, input.Notes)
}

func (s *appointmentService) Delete(ctx context.Context, id, userID int64) error {
	return s.repo.Delete(ctx, id, userID)
}

func (s *appointmentService) GetFiltered(ctx context.Context, userID int64, from, to *time.Time, limit int) ([]domain.Appointment, error) {
	return s.repo.GetFiltered(ctx, userID, from, to, limit)
}
