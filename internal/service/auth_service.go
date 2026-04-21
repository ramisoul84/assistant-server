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

// Notifier sends messages via the Telegram bot.
// Defined as interface here so auth service doesn't import the bot package.
type Notifier interface {
	SendMessage(telegramID int64, text string) error
}

type AuthService interface {
	RequestOTP(ctx context.Context, handle string) error
	VerifyOTP(ctx context.Context, handle, code string) (string, error) // returns JWT
}

type authService struct {
	userRepo  repository.UserRepository
	otpRepo   repository.OTPRepository
	notifier  Notifier
	jwtSecret string
	jwtExpiry time.Duration
	otpExpiry time.Duration
	log       logger.Logger
}

func NewAuthService(
	userRepo repository.UserRepository,
	otpRepo repository.OTPRepository,
	notifier Notifier,
	jwtSecret string,
	jwtExpiry time.Duration,
	otpExpiry time.Duration,
) AuthService {
	return &authService{
		userRepo:  userRepo,
		otpRepo:   otpRepo,
		notifier:  notifier,
		jwtSecret: jwtSecret,
		jwtExpiry: jwtExpiry,
		otpExpiry: otpExpiry,
		log:       logger.Get(),
	}
}

func (s *authService) RequestOTP(ctx context.Context, handle string) error {
	// Strip leading @ if user typed it
	if len(handle) > 0 && handle[0] == '@' {
		handle = handle[1:]
	}

	user, err := s.userRepo.FindByHandle(ctx, handle)
	if err != nil {
		// Don't leak whether the user exists — return generic message
		s.log.Warn().Str("handle", handle).Msg("OTP requested for unknown handle")
		return nil
	}

	code, err := generateOTP()
	fmt.Println("Code generated", code)
	if err != nil {
		return fmt.Errorf("authService.RequestOTP: %w", err)
	}

	expiresAt := time.Now().Add(s.otpExpiry)
	if _, err := s.otpRepo.Create(ctx, user.ID, code, expiresAt); err != nil {
		return fmt.Errorf("authService.RequestOTP: %w", err)
	}

	msg := fmt.Sprintf("🔐 Your login code: *%s*\n\nExpires in 5 minutes. Do not share this code.", code)
	if err := s.notifier.SendMessage(user.TelegramID, msg); err != nil {
		return fmt.Errorf("authService.RequestOTP: failed to send OTP: %w", err)
	}

	s.log.Info().Str("handle", handle).Msg("OTP sent")
	return nil
}

func (s *authService) VerifyOTP(ctx context.Context, handle, code string) (string, error) {
	if len(handle) > 0 && handle[0] == '@' {
		handle = handle[1:]
	}

	user, err := s.userRepo.FindByHandle(ctx, handle)
	if err != nil {
		return "", domain.ErrUnauthorized
	}

	otp, err := s.otpRepo.FindValid(ctx, user.ID, code)
	if err != nil {
		return "", domain.ErrUnauthorized
	}

	// Mark used immediately — prevents replay attacks
	if err := s.otpRepo.MarkUsed(ctx, otp.ID); err != nil {
		return "", fmt.Errorf("authService.VerifyOTP: %w", err)
	}

	token, err := jwt.Issue(s.jwtSecret, s.jwtExpiry, domain.AuthClaims{
		UserID:     user.ID,
		TelegramID: user.TelegramID,
		Handle:     user.TelegramHandle,
	})
	if err != nil {
		return "", fmt.Errorf("authService.VerifyOTP: %w", err)
	}

	s.log.Info().Str("handle", handle).Int64("user_id", user.ID).Msg("User authenticated")
	return token, nil
}

// generateOTP produces a cryptographically random 6-digit code.
func generateOTP() (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// Convert to 6-digit number: 000000–999999
	n := (int(b[0])<<16 | int(b[1])<<8 | int(b[2])) % 1_000_000
	return fmt.Sprintf("%06d", n), nil
}
