package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/ramisoul84/assistant-server/pkg/jwt"
)

const UserIDKey = "uid"

func RequireAuth(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		h := c.Get("Authorization")
		if h == "" { return fiber.NewError(fiber.StatusUnauthorized, "missing token") }
		parts := strings.SplitN(h, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid Authorization header")
		}
		claims, err := jwt.Verify(secret, parts[1])
		if err != nil { return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token") }
		c.Locals(UserIDKey, claims.UserID)
		return c.Next()
	}
}

func UID(c *fiber.Ctx) (int64, error) {
	v := c.Locals(UserIDKey)
	if v == nil { return 0, fiber.NewError(fiber.StatusUnauthorized, "not authenticated") }
	id, ok := v.(int64)
	if !ok { return 0, fiber.NewError(fiber.StatusInternalServerError, "bad uid type") }
	return id, nil
}
