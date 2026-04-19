package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ramisoul84/assistant-server/internal/config"
	"github.com/rs/zerolog"
)

var (
	globalLogger Logger
	globalCloser io.Closer
	once         sync.Once
)

// Logger defines the logging interface used throughout the app.
// Wrapping zerolog.Event keeps callers decoupled from the logging library.
type Logger interface {
	Debug() *zerolog.Event
	Info() *zerolog.Event
	Warn() *zerolog.Event
	Error() *zerolog.Event
	Fatal() *zerolog.Event
	WithFields(fields map[string]any) Logger
	Close() error
}

type logger struct {
	zerolog.Logger
	closer io.Closer
}

type outputTarget struct {
	writer io.Writer
	closer io.Closer
}

// New creates a configured Logger from the app config.
func New(cfg *config.Config) Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	level := parseLevel(cfg.Logging.Level)
	output := buildOutput(cfg)

	zlog := zerolog.New(output.writer).
		Level(level).
		With().
		Str("service", cfg.App.Name).
		Str("version", cfg.App.Version).
		Str("env", cfg.App.Env).
		Timestamp().
		Logger()

	return &logger{Logger: zlog, closer: output.closer}
}

// InitGlobal initialises the singleton logger exactly once.
func InitGlobal(cfg *config.Config) {
	once.Do(func() {
		globalLogger = New(cfg)
		if l, ok := globalLogger.(*logger); ok {
			globalCloser = l.closer
		}
	})
}

// Get returns the global logger. Panics if InitGlobal was never called.
func Get() Logger {
	if globalLogger == nil {
		panic("logger: Get() called before InitGlobal()")
	}
	return globalLogger
}

func CloseGlobal() error {
	if globalCloser == nil {
		return nil
	}
	err := globalCloser.Close()
	globalCloser = nil
	return err
}

// ─── Package-level shortcuts ──────────────────────────────────────────────────
func Debug() *zerolog.Event { return Get().Debug() }
func Info() *zerolog.Event  { return Get().Info() }
func Warn() *zerolog.Event  { return Get().Warn() }
func Error() *zerolog.Event { return Get().Error() }
func Fatal() *zerolog.Event { return Get().Fatal() }

// WithFields returns a child logger with the given fields attached to every
// subsequent log entry. Useful for request-scoped logging.
func (l *logger) WithFields(fields map[string]any) Logger {
	ctx := l.Logger.With()
	for key, value := range fields {
		ctx = ctx.Interface(key, value)
	}
	return &logger{Logger: ctx.Logger(), closer: l.closer}
}

func (l *logger) Close() error {
	if l.closer == nil {
		return nil
	}
	return l.closer.Close()
}

// ─── Output builder ───────────────────────────────────────────────────────────

func buildOutput(cfg *config.Config) outputTarget {
	if cfg.Logging.Output == "file" {
		return buildFileOutput(cfg.Logging.File)
	}

	if cfg.IsDevelopment() {
		return outputTarget{
			writer: zerolog.ConsoleWriter{
				Out:        os.Stdout,
				TimeFormat: time.RFC822,
			},
		}
	}
	return outputTarget{writer: os.Stdout}
}

func buildFileOutput(path string) outputTarget {
	// Ensure parent directory exists before trying to open the file.
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Printf("logger: failed to create log directory %q: %v — falling back to stdout\n", dir, err)
			return outputTarget{writer: os.Stdout}
		}
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Printf("logger: failed to open log file %q: %v — falling back to stdout\n", path, err)
		return outputTarget{writer: os.Stdout}
	}
	return outputTarget{
		writer: file,
		closer: file,
	}
}

// ─── Level parser ─────────────────────────────────────────────────────────────

func parseLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		fmt.Fprintf(os.Stderr, "logger: unknown LOG_LEVEL %q, defaulting to info\n", level)
		return zerolog.InfoLevel
	}
}
