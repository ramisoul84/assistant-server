package main

import (
	"os"

	"github.com/ramisoul84/assistant-server/internal/config"
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
			logger.Error().Err(err).Msg("Failed to close logger output")
		}
	}()
	log := logger.Get()

	log.Info().
		Str("env", cfg.App.Env).
		Str("version", cfg.App.Version).
		Int("pid", os.Getpid()).
		Msg("Starting Assistant Server")
}
