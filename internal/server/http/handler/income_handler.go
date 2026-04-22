package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/ramisoul84/assistant-server/internal/server/http/middleware"
	"github.com/ramisoul84/assistant-server/internal/service"
)

type IncomeHandler struct{ svc service.IncomeService }

func NewIncomeHandler(svc service.IncomeService) *IncomeHandler { return &IncomeHandler{svc: svc} }

func (h *IncomeHandler) List(c *fiber.Ctx) error {
	userID, err := middleware.AuthUserID(c)
	if err != nil { return err }
	var from, to *time.Time
	if f := c.Query("from"); f != "" { t, _ := time.Parse("2006-01-02", f); from = &t }
	if t := c.Query("to");   t != "" { p, _ := time.Parse("2006-01-02", t); to   = &p }
	list, err := h.svc.GetFiltered(c.UserContext(), userID, from, to, 0)
	if err != nil { return fiber.NewError(fiber.StatusInternalServerError, err.Error()) }
	return c.JSON(fiber.Map{"data": list})
}

func (h *IncomeHandler) Create(c *fiber.Ctx) error {
	userID, err := middleware.AuthUserID(c)
	if err != nil { return err }
	var body struct {
		Amount      float64   `json:"amount"`
		Currency    string    `json:"currency"`
		Category    string    `json:"category"`
		Description string    `json:"description"`
		ReceivedAt  time.Time `json:"received_at"`
	}
	if err := c.BodyParser(&body); err != nil { return fiber.NewError(fiber.StatusBadRequest, "invalid body") }
	saved, err := h.svc.Create(c.UserContext(), userID, &service.IncomeInput{
		Amount: body.Amount, Currency: body.Currency, Category: body.Category,
		Description: body.Description, ReceivedAt: body.ReceivedAt,
	})
	if err != nil { return fiber.NewError(fiber.StatusBadRequest, err.Error()) }
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": saved})
}

func (h *IncomeHandler) Update(c *fiber.Ctx) error {
	userID, err := middleware.AuthUserID(c)
	if err != nil { return err }
	id, err := c.ParamsInt("id")
	if err != nil { return fiber.NewError(fiber.StatusBadRequest, "invalid id") }
	var body struct {
		Amount      float64   `json:"amount"`
		Currency    string    `json:"currency"`
		Category    string    `json:"category"`
		Description string    `json:"description"`
		ReceivedAt  time.Time `json:"received_at"`
	}
	if err := c.BodyParser(&body); err != nil { return fiber.NewError(fiber.StatusBadRequest, "invalid body") }
	updated, err := h.svc.Update(c.UserContext(), int64(id), userID, &service.IncomeInput{
		Amount: body.Amount, Currency: body.Currency, Category: body.Category,
		Description: body.Description, ReceivedAt: body.ReceivedAt,
	})
	if err != nil { return fiber.NewError(fiber.StatusBadRequest, err.Error()) }
	return c.JSON(fiber.Map{"data": updated})
}

func (h *IncomeHandler) Delete(c *fiber.Ctx) error {
	userID, err := middleware.AuthUserID(c)
	if err != nil { return err }
	id, err := c.ParamsInt("id")
	if err != nil { return fiber.NewError(fiber.StatusBadRequest, "invalid id") }
	if err := h.svc.Delete(c.UserContext(), int64(id), userID); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}
