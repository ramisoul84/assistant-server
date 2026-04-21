package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ramisoul84/assistant-server/internal/domain"
	"github.com/ramisoul84/assistant-server/internal/repository"
)

type GymService interface {
	StartSession(ctx context.Context, userID int64) (*domain.GymSession, error)
	AddExercise(ctx context.Context, sessionID int64, intent *domain.GymExerciseIntent) (*domain.GymExercise, error)
	CloseSession(ctx context.Context, sessionID int64) error
	DeleteSession(ctx context.Context, sessionID, userID int64) error
	UpdateExercise(ctx context.Context, id, userID int64, name string, sets, reps int, weightKg float64, notes string) (*domain.GymExercise, error)
	DeleteExercise(ctx context.Context, id, userID int64) error
	GetSessions(ctx context.Context, userID int64, from, to *time.Time, limit int) ([]domain.GymSession, error)
}

type gymService struct{ repo repository.GymRepository }

func NewGymService(repo repository.GymRepository) GymService {
	return &gymService{repo: repo}
}

func (s *gymService) StartSession(ctx context.Context, userID int64) (*domain.GymSession, error) {
	return s.repo.CreateSession(ctx, userID, time.Now().UTC())
}

func (s *gymService) AddExercise(ctx context.Context, sessionID int64, intent *domain.GymExerciseIntent) (*domain.GymExercise, error) {
	if intent.Name == "" {
		return nil, fmt.Errorf("%w: exercise name is required", domain.ErrInvalidInput)
	}
	return s.repo.AddExercise(ctx, sessionID, &domain.GymExercise{
		SessionID: sessionID, Name: intent.Name,
		Sets: intent.Sets, Reps: intent.Reps, WeightKg: intent.WeightKg, Notes: intent.Notes,
	})
}

func (s *gymService) CloseSession(ctx context.Context, sessionID int64) error {
	return s.repo.CloseSession(ctx, sessionID, time.Now().UTC())
}

func (s *gymService) DeleteSession(ctx context.Context, sessionID, userID int64) error {
	return s.repo.DeleteSession(ctx, sessionID, userID)
}

func (s *gymService) UpdateExercise(ctx context.Context, id, userID int64, name string, sets, reps int, weightKg float64, notes string) (*domain.GymExercise, error) {
	return s.repo.UpdateExercise(ctx, id, userID, name, sets, reps, weightKg, notes)
}

func (s *gymService) DeleteExercise(ctx context.Context, id, userID int64) error {
	return s.repo.DeleteExercise(ctx, id, userID)
}

func (s *gymService) GetSessions(ctx context.Context, userID int64, from, to *time.Time, limit int) ([]domain.GymSession, error) {
	return s.repo.GetSessionsWithExercises(ctx, userID, from, to, limit)
}
