package config

import "time"

type Config struct {
	App      AppConfig
	Logging  LoggingConfig
	Server   ServerConfig
	Database PostgresConfig
	Telegram TelegramConfig
	AI       AIConfig
	Auth     AuthConfig
}

type AppConfig struct {
	Env     string
	Name    string
	Version string
}

type LoggingConfig struct {
	Level  string
	Output string
	File   string
}

type ServerConfig struct {
	Port           string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	AllowedOrigins string
	BodyLimitMB    int
}

type PostgresConfig struct {
	Host         string
	Port         string
	User         string
	Password     string
	Name         string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
	ConnLifetime time.Duration
	ConnIdleTime time.Duration
}

type TelegramConfig struct {
	Token   string
	Debug   bool
	Timeout int
}

type AIConfig struct {
	APIKey string
	Model  string
}

type AuthConfig struct {
	JWTSecret     string
	JWTExpiry     time.Duration
	OTPExpiry     time.Duration
}

func Load(env string) *Config {
	loadDotenv(env)
	return &Config{
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
		Server: ServerConfig{
			Port:           getEnv("PORT", "8080"),
			ReadTimeout:    getEnvDuration("READ_TIMEOUT", 15*time.Second),
			WriteTimeout:   getEnvDuration("WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:    getEnvDuration("IDLE_TIMEOUT", 120*time.Second),
			AllowedOrigins: getEnv("ALLOWED_ORIGINS", "http://localhost:4200"),
			BodyLimitMB:    getEnvInt("BODY_LIMIT_MB", 10),
		},
		Database: PostgresConfig{
			Host:         getEnv("DB_HOST", "localhost"),
			Port:         getEnv("DB_PORT", "5432"),
			User:         mustGetEnv("DB_USER"),
			Password:     mustGetEnv("DB_PASSWORD"),
			Name:         mustGetEnv("DB_NAME"),
			SSLMode:      getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns: getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 10),
			ConnLifetime: getEnvDuration("DB_CONN_LIFETIME", 5*time.Minute),
			ConnIdleTime: getEnvDuration("DB_CONN_IDLE_TIME", 1*time.Minute),
		},
		Telegram: TelegramConfig{
			Token:   mustGetEnv("BOT_TOKEN"),
			Debug:   getEnvBool("BOT_DEBUG", false),
			Timeout: getEnvInt("BOT_TIMEOUT", 60),
		},
		AI: AIConfig{
			APIKey: mustGetEnv("GROQ_API_KEY"),
			Model:  getEnv("GROQ_MODEL", "llama-3.3-70b-versatile"),
		},
		Auth: AuthConfig{
			JWTSecret: mustGetEnv("JWT_SECRET"),
			JWTExpiry: getEnvDuration("JWT_EXPIRY", 24*time.Hour*7), // 7 days
			OTPExpiry: getEnvDuration("OTP_EXPIRY", 5*time.Minute),
		},
	}
}

func (c *Config) IsDevelopment() bool { return c.App.Env == "development" }
func (c *Config) IsProduction() bool  { return c.App.Env == "production" }
