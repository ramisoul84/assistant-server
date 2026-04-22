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
	appointmentRepo := repository.NewAppointmentRepository(db)
	expenseRepo := repository.NewExpenseRepository(db)
	incomeRepo := repository.NewIncomeRepository(db)
	gymRepo := repository.NewGymRepository(db)
	otpRepo := repository.NewOTPRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	budgetRepo := repository.NewBudgetRepository(db)

	// Services
	groqClient := ai.NewGroqClient(cfg.AI)
	aiParser := service.NewAIParserService(groqClient, cfg.AI.Model)
	appointmentSvc := service.NewAppointmentService(appointmentRepo)
	expenseSvc := service.NewExpenseService(expenseRepo)
	incomeSvc := service.NewIncomeService(incomeRepo)
	gymSvc := service.NewGymService(gymRepo)
	log.Info().Str("model", cfg.AI.Model).Msg("Groq client ready")

	// Bot
	telegramBot, notifier, err := bot.New(cfg, userRepo, aiParser, appointmentSvc, expenseSvc, gymSvc, budgetRepo)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Telegram bot")
	}

	// Auth + notifications
	authSvc := service.NewAuthService(userRepo, otpRepo, notifier, cfg.Auth.JWTSecret, cfg.Auth.JWTExpiry, cfg.Auth.OTPExpiry)
	notifSvc := service.NewNotificationService(appointmentRepo, expenseRepo, userRepo, budgetRepo, notifRepo, notifier)

	// HTTP
	srv := httpserver.New(cfg)
	srv.RegisterRoutes(
		handler.NewAuthHandler(authSvc),
		handler.NewAppointmentHandler(appointmentSvc),
		handler.NewExpenseHandler(expenseSvc),
		handler.NewIncomeHandler(incomeSvc),
		handler.NewGymHandler(gymSvc),
		handler.NewBudgetHandler(budgetRepo),
	)

	listenErrCh := make(chan error, 1)
	go func() {
		log.Info().Str("port", cfg.Server.Port).Msg("HTTP server listening")
		listenErrCh <- srv.App().Listen(":" + cfg.Server.Port)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { log.Info().Msg("Telegram bot started"); telegramBot.Start(ctx) }()

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				notifSvc.CheckAppointments(ctx)
			}
		}
	}()

	go func() {
		notifSvc.CheckBudgets(ctx)
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				notifSvc.CheckBudgets(ctx)
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = otpRepo.DeleteExpired(ctx)
			}
		}
	}()

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
