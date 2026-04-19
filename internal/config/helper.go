package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// loadDotenv loads the appropriate .env file for the given environment.
func loadDotenv(env string) {
	file := ".env.dev"
	if strings.ToLower(env) == "production" {
		file = ".env.prod"
	}

	if err := godotenv.Load(file); err != nil {
		if strings.ToLower(env) == "production" {
			return
		}
		log.Printf("WARNING: %s not found — falling back to system environment variables\n", file)
		log.Printf("         Create %s to configure your local environment\n", file)
	}
}

// getEnv returns the env var value or the fallback if not set.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
