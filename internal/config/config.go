package config

// Config holds all configuration for App.
type Config struct {
	App     AppConfig
	Logging LoggingConfig
}

// AppConfig holds application metadata.
type AppConfig struct {
	Env     string
	Name    string
	Version string
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	Level  string
	Output string
	File   string
}

// ─── Load ─────────────────────────────────────────────────────────────────────

func Load(env string) *Config {
	loadDotenv(env)

	cfg := &Config{
		App: AppConfig{
			Env:     getEnv("APP_ENV", "development"),
			Name:    getEnv("APP_NAME", "assistant-server"),
			Version: getEnv("APP_VERSION", "1.0.0"),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Output: getEnv("LOG_OUTPUT", "stdout"),
			File:   getEnv("LOG_FILE", "./logs/assistant-server.log"),
		},
	}

	return cfg

}

// ─── Helper methods ───────────────────────────────────────────────────────────

func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}
