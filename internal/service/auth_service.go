package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/ramisoul84/assistant-server/internal/domain"
	"github.com/ramisoul84/assistant-server/internal/repository"
	"github.com/ramisoul84/assistant-server/pkg/jwt"
	"github.com/ramisoul84/assistant-server/pkg/logger"
)

type Notifier interface {
	SendMessage(telegramID int64, text string) error
}

type AuthService interface {
	RequestOTP(ctx context.Context, handle string) error
	VerifyOTP(ctx context.Context, handle, code string) (string, error)
}

type authService struct {
	users     repository.UserRepository
	otp       repository.OTPRepository
	notifier  Notifier
	secret    string
	jwtExpiry time.Duration
	otpExpiry time.Duration
	log       logger.Logger
}

func NewAuthService(users repository.UserRepository, otp repository.OTPRepository, notifier Notifier, secret string, jwtExpiry, otpExpiry time.Duration) AuthService {
	return &authService{users: users, otp: otp, notifier: notifier, secret: secret, jwtExpiry: jwtExpiry, otpExpiry: otpExpiry, log: logger.Get()}
}

func (s *authService) RequestOTP(ctx context.Context, handle string) error {
	if len(handle) > 0 && handle[0] == '@' {
		handle = handle[1:]
	}
	user, err := s.users.FindByHandle(ctx, handle)
	if err != nil {
		s.log.Warn().Str("handle", handle).Msg("OTP requested for unknown handle")
		return nil // don't leak existence
	}
	b := make([]byte, 3)
	rand.Read(b)
	code := fmt.Sprintf("%06d", (int(b[0])<<16|int(b[1])<<8|int(b[2]))%1_000_000)
	if _, err := s.otp.Create(ctx, user.ID, code, time.Now().Add(s.otpExpiry)); err != nil {
		return err
	}
	return s.notifier.SendMessage(user.TelegramID, fmt.Sprintf("🔐 Your login code: *%s*\n\nExpires in 5 minutes.", code))
}

func (s *authService) VerifyOTP(ctx context.Context, handle, code string) (string, error) {
	if len(handle) > 0 && handle[0] == '@' {
		handle = handle[1:]
	}
	user, err := s.users.FindByHandle(ctx, handle)
	if err != nil {
		return "", domain.ErrUnauthorized
	}
	rec, err := s.otp.FindValid(ctx, user.ID, code)
	if err != nil {
		return "", domain.ErrUnauthorized
	}
	_ = s.otp.MarkUsed(ctx, rec.ID)
	return jwt.Issue(s.secret, s.jwtExpiry, domain.AuthClaims{UserID: user.ID, TelegramID: user.TelegramID, Handle: user.Handle})
}
