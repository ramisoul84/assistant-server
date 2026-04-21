package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/ramisoul84/assistant-server/internal/domain"
	"github.com/ramisoul84/assistant-server/internal/service"
)

type AuthHandler struct {
	svc service.AuthService
}

func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// POST /api/v1/auth/request-otp
// Body: { "handle": "ramisoul" }
func (h *AuthHandler) RequestOTP(c *fiber.Ctx) error {
	var body struct {
		Handle string `json:"handle"`
	}
	if err := c.BodyParser(&body); err != nil || body.Handle == "" {
		return fiber.NewError(fiber.StatusBadRequest, "handle is required")
	}

	if err := h.svc.RequestOTP(c.UserContext(), body.Handle); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to send OTP")
	}

	// Always return 200 — don't leak whether handle exists
	return c.JSON(fiber.Map{
		"message": "If this account exists, an OTP has been sent to your Telegram.",
	})
}

// POST /api/v1/auth/verify-otp
// Body: { "handle": "ramisoul", "code": "483921" }
func (h *AuthHandler) VerifyOTP(c *fiber.Ctx) error {
	var body struct {
		Handle string `json:"handle"`
		Code   string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil || body.Handle == "" || body.Code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "handle and code are required")
	}

	token, err := h.svc.VerifyOTP(c.UserContext(), body.Handle, body.Code)
	if err != nil {
		if err == domain.ErrUnauthorized {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired code")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "verification failed")
	}

	return c.JSON(fiber.Map{"token": token})
}
