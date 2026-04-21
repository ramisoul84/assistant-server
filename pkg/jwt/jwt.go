package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ramisoul84/assistant-server/internal/domain"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID     int64  `json:"user_id"`
	TelegramID int64  `json:"telegram_id"`
	Handle     string `json:"handle"`
}

// Issue creates a signed JWT for the given user.
func Issue(secret string, expiry time.Duration, claims domain.AuthClaims) (string, error) {
	c := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID:     claims.UserID,
		TelegramID: claims.TelegramID,
		Handle:     claims.Handle,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("jwt.Issue: %w", err)
	}
	return signed, nil
}

// Verify parses and validates a JWT, returning the embedded claims.
func Verify(secret, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("jwt.Verify: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("jwt.Verify: invalid token")
	}
	return claims, nil
}
