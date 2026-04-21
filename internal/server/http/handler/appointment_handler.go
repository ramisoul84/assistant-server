package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/ramisoul84/assistant-server/internal/server/http/middleware"
	"github.com/ramisoul84/assistant-server/internal/service"
)

type AppointmentHandler struct {
	svc service.AppointmentService
}

func NewAppointmentHandler(svc service.AppointmentService) *AppointmentHandler {
	return &AppointmentHandler{svc: svc}
}

// GET /api/v1/appointments
func (h *AppointmentHandler) List(c *fiber.Ctx) error {
	userID, err := middleware.AuthUserID(c)
	if err != nil {
		return err
	}

	// Optional date filters: ?from=2026-01-01&to=2026-12-31
	var from, to *time.Time
	if f := c.Query("from"); f != "" {
		t, err := time.Parse("2006-01-02", f)
		if err == nil {
			from = &t
		}
	}
	if t := c.Query("to"); t != "" {
		parsed, err := time.Parse("2006-01-02", t)
		if err == nil {
			to = &parsed
		}
	}

	list, err := h.svc.GetFiltered(c.UserContext(), userID, from, to, 0)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"data": list})
}

// POST /api/v1/appointments
func (h *AppointmentHandler) Create(c *fiber.Ctx) error {
	userID, err := middleware.AuthUserID(c)
	if err != nil {
		return err
	}
	var body struct {
		Title    string    `json:"title"`
		Datetime time.Time `json:"datetime"`
		Notes    string    `json:"notes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}

	saved, err := h.svc.Create(c.UserContext(), userID, &service.AppointmentInput{
		Title:    body.Title,
		Datetime: body.Datetime,
		Notes:    body.Notes,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": saved})
}

// PUT /api/v1/appointments/:id
func (h *AppointmentHandler) Update(c *fiber.Ctx) error {
	userID, err := middleware.AuthUserID(c)
	if err != nil {
		return err
	}
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	var body struct {
		Title    string    `json:"title"`
		Datetime time.Time `json:"datetime"`
		Notes    string    `json:"notes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}

	updated, err := h.svc.Update(c.UserContext(), int64(id), userID, &service.AppointmentInput{
		Title:    body.Title,
		Datetime: body.Datetime,
		Notes:    body.Notes,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"data": updated})
}

// DELETE /api/v1/appointments/:id
func (h *AppointmentHandler) Delete(c *fiber.Ctx) error {
	userID, err := middleware.AuthUserID(c)
	if err != nil {
		return err
	}
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	if err := h.svc.Delete(c.UserContext(), int64(id), userID); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}
