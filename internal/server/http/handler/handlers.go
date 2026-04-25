package handler

import (
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/ramisoul84/assistant-server/internal/domain"
	"github.com/ramisoul84/assistant-server/internal/repository"
	"github.com/ramisoul84/assistant-server/internal/server/http/middleware"
	"github.com/ramisoul84/assistant-server/internal/service"
)

// ── Auth ──────────────────────────────────────────────────────────────────────

type AuthHandler struct{ svc service.AuthService }

func NewAuthHandler(svc service.AuthService) *AuthHandler { return &AuthHandler{svc} }

func (h *AuthHandler) RequestOTP(c *fiber.Ctx) error {
	var b struct {
		Handle string `json:"handle"`
	}
	if err := c.BodyParser(&b); err != nil || b.Handle == "" {
		return fiber.NewError(fiber.StatusBadRequest, "handle required")
	}
	_ = h.svc.RequestOTP(c.UserContext(), b.Handle)
	return c.JSON(fiber.Map{"message": "If this account exists, a code was sent to your Telegram."})
}

func (h *AuthHandler) VerifyOTP(c *fiber.Ctx) error {
	var b struct {
		Handle string `json:"handle"`
		Code   string `json:"code"`
	}
	if err := c.BodyParser(&b); err != nil || b.Handle == "" || b.Code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "handle and code required")
	}
	token, err := h.svc.VerifyOTP(c.UserContext(), b.Handle, b.Code)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired code")
	}
	return c.JSON(fiber.Map{"token": token})
}

// ── Finance ───────────────────────────────────────────────────────────────────

type FinanceHandler struct{ repo repository.FinanceRepository }

func NewFinanceHandler(repo repository.FinanceRepository) *FinanceHandler {
	return &FinanceHandler{repo}
}

func parseDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}

func (h *FinanceHandler) ListExpenses(c *fiber.Ctx) error {
	uid, err := middleware.UID(c)
	if err != nil {
		return err
	}
	list, err := h.repo.GetExpenses(c.UserContext(), uid, parseDate(c.Query("from")), parseDate(c.Query("to")))
	if err != nil {
		return fiber.NewError(500, err.Error())
	}
	if list == nil {
		list = []domain.Expense{}
	}
	return c.JSON(fiber.Map{"data": list})
}

func (h *FinanceHandler) CreateExpense(c *fiber.Ctx) error {
	uid, err := middleware.UID(c)
	if err != nil {
		return err
	}
	var b struct {
		Amount      float64   `json:"amount"`
		Currency    string    `json:"currency"`
		Category    string    `json:"category"`
		Description string    `json:"description"`
		HappenedAt  time.Time `json:"happened_at"`
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(400, "invalid body")
	}
	if b.Currency == "" {
		b.Currency = "EUR"
	}
	if b.Category == "" {
		b.Category = "other"
	}
	if b.HappenedAt.IsZero() {
		b.HappenedAt = time.Now().UTC()
	}
	saved, err := h.repo.CreateExpense(c.UserContext(), &domain.Expense{
		UserID: uid, Amount: b.Amount, Currency: b.Currency,
		Category: b.Category, Description: b.Description, HappenedAt: b.HappenedAt,
	})
	if err != nil {
		return fiber.NewError(400, err.Error())
	}
	return c.Status(201).JSON(fiber.Map{"data": saved})
}

func (h *FinanceHandler) UpdateExpense(c *fiber.Ctx) error {
	uid, err := middleware.UID(c)
	if err != nil {
		return err
	}
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(400, "invalid id")
	}
	var b struct {
		Amount      float64   `json:"amount"`
		Currency    string    `json:"currency"`
		Category    string    `json:"category"`
		Description string    `json:"description"`
		HappenedAt  time.Time `json:"happened_at"`
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(400, "invalid body")
	}
	saved, err := h.repo.UpdateExpense(c.UserContext(), int64(id), uid, b.Amount, b.Currency, b.Category, b.Description, b.HappenedAt)
	if err != nil {
		return fiber.NewError(400, err.Error())
	}
	return c.JSON(fiber.Map{"data": saved})
}

func (h *FinanceHandler) DeleteExpense(c *fiber.Ctx) error {
	uid, err := middleware.UID(c)
	if err != nil {
		return err
	}
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(400, "invalid id")
	}
	if err := h.repo.DeleteExpense(c.UserContext(), int64(id), uid); err != nil {
		return fiber.NewError(400, err.Error())
	}
	return c.SendStatus(204)
}

func (h *FinanceHandler) ListIncomes(c *fiber.Ctx) error {
	uid, err := middleware.UID(c)
	if err != nil {
		return err
	}
	list, err := h.repo.GetIncomes(c.UserContext(), uid, parseDate(c.Query("from")), parseDate(c.Query("to")))
	if err != nil {
		return fiber.NewError(500, err.Error())
	}
	if list == nil {
		list = []domain.Income{}
	}
	return c.JSON(fiber.Map{"data": list})
}

func (h *FinanceHandler) CreateIncome(c *fiber.Ctx) error {
	uid, err := middleware.UID(c)
	if err != nil {
		return err
	}
	var b struct {
		Amount      float64   `json:"amount"`
		Currency    string    `json:"currency"`
		Category    string    `json:"category"`
		Description string    `json:"description"`
		HappenedAt  time.Time `json:"happened_at"`
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(400, "invalid body")
	}
	if b.Currency == "" {
		b.Currency = "EUR"
	}
	if b.Category == "" {
		b.Category = "other"
	}
	if b.HappenedAt.IsZero() {
		b.HappenedAt = time.Now().UTC()
	}
	saved, err := h.repo.CreateIncome(c.UserContext(), &domain.Income{
		UserID: uid, Amount: b.Amount, Currency: b.Currency,
		Category: b.Category, Description: b.Description, HappenedAt: b.HappenedAt,
	})
	if err != nil {
		return fiber.NewError(400, err.Error())
	}
	return c.Status(201).JSON(fiber.Map{"data": saved})
}

func (h *FinanceHandler) UpdateIncome(c *fiber.Ctx) error {
	uid, err := middleware.UID(c)
	if err != nil {
		return err
	}
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(400, "invalid id")
	}
	var b struct {
		Amount      float64   `json:"amount"`
		Currency    string    `json:"currency"`
		Category    string    `json:"category"`
		Description string    `json:"description"`
		HappenedAt  time.Time `json:"happened_at"`
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(400, "invalid body")
	}
	saved, err := h.repo.UpdateIncome(c.UserContext(), int64(id), uid, b.Amount, b.Currency, b.Category, b.Description, b.HappenedAt)
	if err != nil {
		return fiber.NewError(400, err.Error())
	}
	return c.JSON(fiber.Map{"data": saved})
}

func (h *FinanceHandler) DeleteIncome(c *fiber.Ctx) error {
	uid, err := middleware.UID(c)
	if err != nil {
		return err
	}
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(400, "invalid id")
	}
	if err := h.repo.DeleteIncome(c.UserContext(), int64(id), uid); err != nil {
		return fiber.NewError(400, err.Error())
	}
	return c.SendStatus(204)
}

func (h *FinanceHandler) GetBudget(c *fiber.Ctx) error {
	uid, err := middleware.UID(c)
	if err != nil {
		return err
	}
	b, err := h.repo.GetBudget(c.UserContext(), uid)
	if err != nil {
		return c.JSON(fiber.Map{"data": nil})
	}
	return c.JSON(fiber.Map{"data": b})
}

func (h *FinanceHandler) UpsertBudget(c *fiber.Ctx) error {
	uid, err := middleware.UID(c)
	if err != nil {
		return err
	}
	var b struct {
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
	}
	if err := c.BodyParser(&b); err != nil || b.Amount <= 0 {
		return fiber.NewError(400, "amount required")
	}
	if b.Currency == "" {
		b.Currency = "EUR"
	}
	saved, err := h.repo.UpsertBudget(c.UserContext(), uid, b.Amount, b.Currency)
	if err != nil {
		return fiber.NewError(500, err.Error())
	}
	return c.JSON(fiber.Map{"data": saved})
}

// ── Notes ─────────────────────────────────────────────────────────────────────

type NoteHandler struct{ repo repository.NoteRepository }

func NewNoteHandler(repo repository.NoteRepository) *NoteHandler { return &NoteHandler{repo} }

func (h *NoteHandler) List(c *fiber.Ctx) error {
	uid, err := middleware.UID(c)
	if err != nil {
		return err
	}
	list, err := h.repo.GetAll(c.UserContext(), uid, parseDate(c.Query("from")), parseDate(c.Query("to")))
	if err != nil {
		return fiber.NewError(500, err.Error())
	}
	if list == nil {
		list = []domain.Note{}
	}
	return c.JSON(fiber.Map{"data": list})
}

func (h *NoteHandler) Create(c *fiber.Ctx) error {
	uid, err := middleware.UID(c)
	if err != nil {
		return err
	}
	var b struct {
		Content  string     `json:"content"`
		Datetime *time.Time `json:"datetime"`
		Tags     []string   `json:"tags"`
	}
	if err := c.BodyParser(&b); err != nil || b.Content == "" {
		return fiber.NewError(400, "content required")
	}
	if b.Tags == nil {
		b.Tags = []string{}
	}
	n := &domain.Note{UserID: uid, Content: b.Content, Tags: b.Tags}
	if b.Datetime != nil {
		n.Datetime = sql.NullTime{Time: *b.Datetime, Valid: true}
	}
	saved, err := h.repo.Create(c.UserContext(), n)
	if err != nil {
		return fiber.NewError(500, err.Error())
	}
	return c.Status(201).JSON(fiber.Map{"data": saved})
}

func (h *NoteHandler) Update(c *fiber.Ctx) error {
	uid, err := middleware.UID(c)
	if err != nil {
		return err
	}
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(400, "invalid id")
	}
	var b struct {
		Content  string     `json:"content"`
		Datetime *time.Time `json:"datetime"`
		Tags     []string   `json:"tags"`
	}
	if err := c.BodyParser(&b); err != nil {
		return fiber.NewError(400, "invalid body")
	}
	if b.Tags == nil {
		b.Tags = []string{}
	}
	saved, err := h.repo.Update(c.UserContext(), int64(id), uid, b.Content, b.Datetime, b.Tags)
	if err != nil {
		return fiber.NewError(400, err.Error())
	}
	return c.JSON(fiber.Map{"data": saved})
}

func (h *NoteHandler) Delete(c *fiber.Ctx) error {
	uid, err := middleware.UID(c)
	if err != nil {
		return err
	}
	id, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(400, "invalid id")
	}
	if err := h.repo.Delete(c.UserContext(), int64(id), uid); err != nil {
		return fiber.NewError(400, err.Error())
	}
	return c.SendStatus(204)
}
