package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/ramisoul84/assistant-server/pkg/jwt"
)

const UserIDKey = "auth_user_id"

// RequireAuth validates the Bearer JWT and injects user_id into Fiber locals.
func RequireAuth(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "missing Authorization header")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			return fiber.NewError(fiber.StatusUnauthorized, "expected: Bearer <token>")
		}

		claims, err := jwt.Verify(jwtSecret, parts[1])
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
		}

		c.Locals(UserIDKey, claims.UserID)
		return c.Next()
	}
}

// AuthUserID extracts the authenticated user ID from Fiber locals.
func AuthUserID(c *fiber.Ctx) (int64, error) {
	val := c.Locals(UserIDKey)
	if val == nil {
		return 0, fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
	}
	id, ok := val.(int64)
	if !ok {
		return 0, fiber.NewError(fiber.StatusInternalServerError, "bad user_id type in context")
	}
	return id, nil
}
