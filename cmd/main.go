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
	// ── Config ────────────────────────────────────────────────────────────────
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}
	cfg := config.Load(env)

	// ── Logger ────────────────────────────────────────────────────────────────
	logger.InitGlobal(cfg)
	defer func() {
		if err := logger.CloseGlobal(); err != nil {
			logger.Error().Err(err).Msg("failed to close logger")
		}
	}()
	log := logger.Get()

	log.Info().
		Str("env", cfg.App.Env).
		Str("version", cfg.App.Version).
		Int("pid", os.Getpid()).
		Msg("starting assistant server")

	// ── Database ──────────────────────────────────────────────────────────────
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to PostgreSQL")
	}
	defer db.Close()
	log.Info().Str("host", cfg.Database.Host).Msg("PostgreSQL connected")

	// ── Repositories ─────────────────────────────────────────────────────────
	userRepo := repository.NewUserRepository(db)
	finRepo := repository.NewFinanceRepository(db)
	noteRepo := repository.NewNoteRepository(db)
	otpRepo := repository.NewOTPRepository(db)
	notifRepo := repository.NewNotificationRepository(db)

	// ── AI client ─────────────────────────────────────────────────────────────
	groqClient := ai.NewGroqClient(cfg.AI)
	aiSvc := service.NewAIService(groqClient, cfg.AI.Model)
	log.Info().Str("model", cfg.AI.Model).Msg("Groq client ready")

	// ── Assistant service (no notifier needed) ────────────────────────────────
	assistantSvc := service.NewAssistantService(finRepo, noteRepo)

	// ── Telegram Bot ──────────────────────────────────────────────────────────
	// Bot is built first because it owns the tgbotapi.BotAPI instance.
	// It returns a *telegram.Notifier which is then injected into services
	// that need to send Telegram messages (auth OTP, budget/appointment alerts).
	telegramBot, notifier, err := bot.New(cfg, userRepo, finRepo, aiSvc, assistantSvc)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialise Telegram bot")
	}

	// ── Services that depend on the notifier ─────────────────────────────────
	// Both are constructed here — after the notifier is available — so there
	// is no partial/nil initialisation and no double construction.
	authSvc := service.NewAuthService(userRepo, otpRepo, notifier, cfg.Auth.JWTSecret, cfg.Auth.JWTExpiry, cfg.Auth.OTPExpiry)
	notifSvc := service.NewNotifService(noteRepo, finRepo, userRepo, notifRepo, notifier)

	// ── HTTP server ───────────────────────────────────────────────────────────
	srv := httpserver.New(cfg)
	srv.RegisterRoutes(
		handler.NewAuthHandler(authSvc),
		handler.NewFinanceHandler(finRepo),
		handler.NewNoteHandler(noteRepo),
	)

	// ── Root context — cancelled on shutdown ──────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ── Start subsystems ──────────────────────────────────────────────────────
	listenErr := make(chan error, 1)

	go func() {
		log.Info().Str("port", cfg.Server.Port).Msg("HTTP server listening")
		listenErr <- srv.App().Listen(":" + cfg.Server.Port)
	}()

	go func() {
		log.Info().Msg("Telegram bot started")
		telegramBot.Start(ctx)
	}()

	// Appointment reminders — every minute
	go runEvery(ctx, time.Minute, func() {
		notifSvc.CheckAppointments(ctx)
	})

	// Budget alerts — immediately on start, then every hour
	go func() {
		notifSvc.CheckBudgets(ctx)
		runEvery(ctx, time.Hour, func() {
			notifSvc.CheckBudgets(ctx)
		})
	}()

	// OTP cleanup — every hour
	go runEvery(ctx, time.Hour, func() {
		if err := otpRepo.DeleteExpired(ctx); err != nil {
			log.Error().Err(err).Msg("OTP cleanup failed")
		}
	})

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	select {
	case err := <-listenErr:
		if err != nil {
			log.Fatal().Err(err).Msg("HTTP server error")
		}
	case sig := <-quit:
		log.Info().Str("signal", sig.String()).Msg("shutdown signal received")
	}

	// Stop background goroutines
	cancel()

	// Give Fiber 30 s to drain in-flight requests
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.App().ShutdownWithContext(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("forced HTTP shutdown")
	}

	log.Info().Msg("server stopped cleanly")
}

// runEvery ticks every interval and calls fn until ctx is cancelled.
func runEvery(ctx context.Context, interval time.Duration, fn func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fn()
		}
	}
}
