package handlers

import (
	"context"
	"fmt"

	"github.com/ramisoul84/assistant-server/internal/domain"
	"github.com/ramisoul84/assistant-server/internal/service"
)

type AppointmentHandler struct {
	svc service.AppointmentService
}

func NewAppointmentHandler(svc service.AppointmentService) *AppointmentHandler {
	return &AppointmentHandler{svc: svc}
}

func (h *AppointmentHandler) Handle(ctx context.Context, userID int64, parsed *domain.AIResponse) (string, error) {
	if parsed.Appointment == nil {
		return "", fmt.Errorf("appointment handler: no appointment payload")
	}
	saved, err := h.svc.Create(ctx, userID, &service.AppointmentInput{
		Title:    parsed.Appointment.Title,
		Datetime: parsed.Appointment.Datetime,
		Notes:    parsed.Appointment.Notes,
	})
	if err != nil {
		return "", err
	}
	if parsed.Reply != "" {
		return "✅ " + parsed.Reply, nil
	}
	return fmt.Sprintf("✅ *%s* saved for %s",
		saved.Title,
		saved.Datetime.Format("Mon, Jan 2 at 15:04"),
	), nil
}
