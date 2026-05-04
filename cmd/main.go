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
	httpserver "github.com/ramisoul84/assistant-server/internal/server/http"
	"github.com/ramisoul84/assistant-server/internal/server/http/handler"
	"github.com/ramisoul84/assistant-server/internal/service"
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

	// ── Logger ──────────────────────────────────────────────────────────────
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

	// Repos
	userRepo := repository.NewUserRepository(db)
	finRepo := repository.NewFinanceRepository(db)
	noteRepo := repository.NewNoteRepository(db)
	otpRepo := repository.NewOTPRepository(db)
	notifRepo := repository.NewNotificationRepository(db)

	// AI + services	// AI + services
	groqClient := ai.NewGroqClient(cfg.AI)
	aiSvc := service.NewAIService(groqClient, cfg.AI.Model)
	assistantSvc := service.NewAssistantService(finRepo, noteRepo)
	authSvc := service.NewAuthService(userRepo, otpRepo, nil, cfg.Auth.JWTSecret, cfg.Auth.JWTExpiry, cfg.Auth.OTPExpiry)
	log.Info().Str("model", cfg.AI.Model).Msg("Groq ready")

	// Bot (also gives us the Notifier)
	telegramBot, notifier, err := bot.New(cfg, userRepo, finRepo, aiSvc, assistantSvc)
	if err != nil {
		log.Fatal().Err(err).Msg("Bot init failed")
	}

	// Wire notifier into auth + notif services
	authSvc = service.NewAuthService(userRepo, otpRepo, notifier, cfg.Auth.JWTSecret, cfg.Auth.JWTExpiry, cfg.Auth.OTPExpiry)
	notifSvc := service.NewNotifService(noteRepo, finRepo, userRepo, notifRepo, notifier)

	// HTTP
	srv := httpserver.New(cfg)
	srv.RegisterRoutes(
		handler.NewAuthHandler(authSvc),
		handler.NewFinanceHandler(finRepo),
		handler.NewNoteHandler(noteRepo),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listenErr := make(chan error, 1)
	go func() {
		log.Info().Str("port", cfg.Server.Port).Msg("HTTP listening")
		listenErr <- srv.App().Listen(":" + cfg.Server.Port)
	}()

	go func() { log.Info().Msg("Bot started"); telegramBot.Start(ctx) }()

	// Appointment reminder — every minute
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				notifSvc.CheckAppointments(ctx)
			}
		}
	}()

	// Budget alerts — every hour
	go func() {
		notifSvc.CheckBudgets(ctx)
		t := time.NewTicker(time.Hour)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				notifSvc.CheckBudgets(ctx)
			}
		}
	}()

	// OTP cleanup — every hour
	go func() {
		t := time.NewTicker(time.Hour)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				otpRepo.DeleteExpired(ctx)
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	select {
	case err := <-listenErr:
		if err != nil {
			log.Fatal().Err(err).Msg("HTTP error")
		}
	case sig := <-quit:
		log.Info().Str("signal", sig.String()).Msg("Shutting down")
	}

	cancel()
	sCtx, sCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer sCancel()
	if err := srv.App().ShutdownWithContext(sCtx); err != nil {
		log.Error().Err(err).Msg("Forced shutdown")
	}
	log.Info().Msg("Stopped")

}
