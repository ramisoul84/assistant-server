package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/ramisoul84/assistant-server/internal/repository"
	"github.com/ramisoul84/assistant-server/internal/server/http/middleware"
)

type BudgetHandler struct{ repo repository.BudgetRepository }

func NewBudgetHandler(repo repository.BudgetRepository) *BudgetHandler { return &BudgetHandler{repo: repo} }

// GET /api/v1/budget
func (h *BudgetHandler) Get(c *fiber.Ctx) error {
	userID, err := middleware.AuthUserID(c)
	if err != nil { return err }
	limit, err := h.repo.GetByUserID(c.UserContext(), userID)
	if err != nil { return c.JSON(fiber.Map{"data": nil}) }
	return c.JSON(fiber.Map{"data": limit})
}

// PUT /api/v1/budget
func (h *BudgetHandler) Upsert(c *fiber.Ctx) error {
	userID, err := middleware.AuthUserID(c)
	if err != nil { return err }
	var body struct {
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
	}
	if err := c.BodyParser(&body); err != nil || body.Amount <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "amount required")
	}
	if body.Currency == "" { body.Currency = "EUR" }
	saved, err := h.repo.Upsert(c.UserContext(), userID, body.Amount, body.Currency)
	if err != nil { return fiber.NewError(fiber.StatusInternalServerError, err.Error()) }
	return c.JSON(fiber.Map{"data": saved})
}
