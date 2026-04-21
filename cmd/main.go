package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ramisoul84/assistant-server/internal/bot"
	"github.com/ramisoul84/assistant-server/internal/config"
	"github.com/ramisoul84/assistant-server/internal/repository"
	"github.com/ramisoul84/assistant-server/internal/service"
	httpserver "github.com/ramisoul84/assistant-server/internal/server/http"
	"github.com/ramisoul84/assistant-server/internal/server/http/handler"
	"github.com/ramisoul84/assistant-server/pkg/ai"
	"github.com/ramisoul84/assistant-server/pkg/database"
	"github.com/ramisoul84/assistant-server/pkg/logger"
)

func main() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}
	cfg := config.Load(env)

	logger.InitGlobal(cfg)
	defer func() {
		if err := logger.CloseGlobal(); err != nil {
			logger.Error().Err(err).Msg("Failed to close logger")
		}
	}()
	log := logger.Get()

	log.Info().
		Str("env", cfg.App.Env).
		Str("version", cfg.App.Version).
		Int("pid", os.Getpid()).
		Msg("Starting Assistant Server")

	// ── Database ──────────────────────────────────────────────────────────────
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to PostgreSQL")
	}
	defer db.Close()
	log.Info().Str("host", cfg.Database.Host).Msg("PostgreSQL connected")

	// ── Repositories ─────────────────────────────────────────────────────────
	userRepo        := repository.NewUserRepository(db)
	appointmentRepo := repository.NewAppointmentRepository(db)
	expenseRepo     := repository.NewExpenseRepository(db)
	gymRepo         := repository.NewGymRepository(db)
	otpRepo         := repository.NewOTPRepository(db)

	// ── AI client ─────────────────────────────────────────────────────────────
	groqClient := ai.NewGroqClient(cfg.AI)
	log.Info().Str("model", cfg.AI.Model).Msg("Groq client ready")

	// ── Services ─────────────────────────────────────────────────────────────
	aiParser       := service.NewAIParserService(groqClient, cfg.AI.Model)
	appointmentSvc := service.NewAppointmentService(appointmentRepo)
	expenseSvc     := service.NewExpenseService(expenseRepo)
	gymSvc         := service.NewGymService(gymRepo)

	// ── Telegram Bot (also returns Notifier for OTP sending) ──────────────────
	telegramBot, notifier, err := bot.New(cfg, userRepo, aiParser, appointmentSvc, expenseSvc, gymSvc)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Telegram bot")
	}

	// ── Auth service (depends on notifier) ────────────────────────────────────
	authSvc := service.NewAuthService(
		userRepo, otpRepo, notifier,
		cfg.Auth.JWTSecret, cfg.Auth.JWTExpiry, cfg.Auth.OTPExpiry,
	)

	// ── HTTP handlers ─────────────────────────────────────────────────────────
	authHandler    := handler.NewAuthHandler(authSvc)
	apptHandler    := handler.NewAppointmentHandler(appointmentSvc)
	expenseHandler := handler.NewExpenseHandler(expenseSvc)
	gymHandler     := handler.NewGymHandler(gymSvc)

	// ── HTTP server ───────────────────────────────────────────────────────────
	srv := httpserver.New(cfg)
	srv.RegisterRoutes(authHandler, apptHandler, expenseHandler, gymHandler)

	listenErrCh := make(chan error, 1)
	go func() {
		log.Info().Str("port", cfg.Server.Port).Msg("HTTP server listening")
		listenErrCh <- srv.App().Listen(":" + cfg.Server.Port)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Info().Msg("Telegram bot started")
		telegramBot.Start(ctx)
	}()

	// ── OTP cleanup — runs every hour ─────────────────────────────────────────
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := otpRepo.DeleteExpired(ctx); err != nil {
					log.Error().Err(err).Msg("OTP cleanup failed")
				}
			}
		}
	}()

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	select {
	case err := <-listenErrCh:
		if err != nil {
			log.Fatal().Err(err).Msg("HTTP server error")
		}
	case sig := <-quit:
		log.Info().Str("signal", sig.String()).Msg("Shutdown signal received")
	}

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.App().ShutdownWithContext(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP forced shutdown")
	}
	log.Info().Msg("Server stopped gracefully")
}
