package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/ramisoul84/assistant-server/internal/server/http/middleware"
	"github.com/ramisoul84/assistant-server/internal/service"
)

type ExpenseHandler struct {
	svc service.ExpenseService
}

func NewExpenseHandler(svc service.ExpenseService) *ExpenseHandler {
	return &ExpenseHandler{svc: svc}
}

// GET /api/v1/expenses?from=&to=&category=
func (h *ExpenseHandler) List(c *fiber.Ctx) error {
	userID, err := middleware.AuthUserID(c)
	if err != nil {
		return err
	}

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

	// Optional category filter (client-side would also work, but doing it here is cleaner)
	if cat := c.Query("category"); cat != "" {
		filtered := list[:0]
		for _, e := range list {
			if e.Category == cat {
				filtered = append(filtered, e)
			}
		}
		list = filtered
	}

	return c.JSON(fiber.Map{"data": list})
}

// POST /api/v1/expenses
func (h *ExpenseHandler) Create(c *fiber.Ctx) error {
	userID, err := middleware.AuthUserID(c)
	if err != nil {
		return err
	}
	var body struct {
		Amount      float64   `json:"amount"`
		Currency    string    `json:"currency"`
		Category    string    `json:"category"`
		Description string    `json:"description"`
		SpentAt     time.Time `json:"spent_at"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}

	saved, err := h.svc.Create(c.UserContext(), userID, &service.ExpenseInput{
		Amount: body.Amount, Currency: body.Currency,
		Category: body.Category, Description: body.Description, SpentAt: body.SpentAt,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": saved})
}

// PUT /api/v1/expenses/:id
func (h *ExpenseHandler) Update(c *fiber.Ctx) error {
	userID, err := middleware.AuthUserID(c)
	if err != nil {
		return err
	}
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	var body struct {
		Amount      float64   `json:"amount"`
		Currency    string    `json:"currency"`
		Category    string    `json:"category"`
		Description string    `json:"description"`
		SpentAt     time.Time `json:"spent_at"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}

	updated, err := h.svc.Update(c.UserContext(), int64(id), userID, &service.ExpenseInput{
		Amount: body.Amount, Currency: body.Currency,
		Category: body.Category, Description: body.Description, SpentAt: body.SpentAt,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"data": updated})
}

// DELETE /api/v1/expenses/:id
func (h *ExpenseHandler) Delete(c *fiber.Ctx) error {
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
