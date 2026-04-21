package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/ramisoul84/assistant-server/internal/server/http/middleware"
	"github.com/ramisoul84/assistant-server/internal/service"
)

type GymHandler struct {
	svc service.GymService
}

func NewGymHandler(svc service.GymService) *GymHandler {
	return &GymHandler{svc: svc}
}

// GET /api/v1/gym/sessions
func (h *GymHandler) ListSessions(c *fiber.Ctx) error {
	userID, err := middleware.AuthUserID(c)
	if err != nil {
		return err
	}
	sessions, err := h.svc.GetSessions(c.UserContext(), userID, nil, nil, 0)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"data": sessions})
}

// DELETE /api/v1/gym/sessions/:id
func (h *GymHandler) DeleteSession(c *fiber.Ctx) error {
	userID, err := middleware.AuthUserID(c)
	if err != nil {
		return err
	}
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	if err := h.svc.DeleteSession(c.UserContext(), int64(id), userID); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// PUT /api/v1/gym/exercises/:id
func (h *GymHandler) UpdateExercise(c *fiber.Ctx) error {
	userID, err := middleware.AuthUserID(c)
	if err != nil {
		return err
	}
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	var body struct {
		Name     string  `json:"name"`
		Sets     int     `json:"sets"`
		Reps     int     `json:"reps"`
		WeightKg float64 `json:"weight_kg"`
		Notes    string  `json:"notes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	updated, err := h.svc.UpdateExercise(c.UserContext(), int64(id), userID, body.Name, body.Sets, body.Reps, body.WeightKg, body.Notes)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"data": updated})
}

// DELETE /api/v1/gym/exercises/:id
func (h *GymHandler) DeleteExercise(c *fiber.Ctx) error {
	userID, err := middleware.AuthUserID(c)
	if err != nil {
		return err
	}
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	if err := h.svc.DeleteExercise(c.UserContext(), int64(id), userID); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}
