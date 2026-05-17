package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	global *slog.Logger
	once   sync.Once
)

// Config holds logger configuration.
type Config struct {
	Level      string `yaml:"level" json:"level"`
	Format     string `yaml:"format" json:"format"`         // "text" or "json"
	File       string `yaml:"file" json:"file"`             // empty = stdout only
	MaxSize    int    `yaml:"max_size_mb" json:"max_size_mb"`
	MaxBackups int    `yaml:"max_backups" json:"max_backups"`
	MaxAge     int    `yaml:"max_age_days" json:"max_age_days"`
	ToStdout   bool   `yaml:"to_stdout" json:"to_stdout"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Level:      "info",
		Format:     "text",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     30,
		ToStdout:   true,
	}
}

// Init initializes the global logger. Safe to call multiple times.
func Init(cfg Config) {
	once.Do(func() {
		global = New(cfg)
	})
}

// L returns the global logger.
func L() *slog.Logger {
	if global == nil {
		global = New(DefaultConfig())
	}
	return global
}

// New creates a logger from config.
func New(cfg Config) *slog.Logger {
	level := parseLevel(cfg.Level)
	opts := &slog.HandlerOptions{Level: level}

	var writers []io.Writer
	if cfg.ToStdout || cfg.File == "" {
		writers = append(writers, os.Stdout)
	}
	if cfg.File != "" {
		_ = os.MkdirAll(filepath.Dir(cfg.File), 0o755)
		writers = append(writers, &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   true,
		})
	}

	var handler slog.Handler
	w := io.MultiWriter(writers...)
	if strings.ToLower(cfg.Format) == "json" {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	return slog.New(handler)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// SetDefault sets the Go default logger to use our global.
func SetDefault() {
	slog.SetDefault(L())
}

// With returns a child logger with fields.
func With(args ...any) *slog.Logger {
	return L().With(args...)
}

// Debug logs at debug level.
func Debug(msg string, args ...any) { L().Debug(msg, args...) }

// Info logs at info level.
func Info(msg string, args ...any) { L().Info(msg, args...) }

// Warn logs at warn level.
func Warn(msg string, args ...any) { L().Warn(msg, args...) }

// Error logs at error level.
func Error(msg string, args ...any) { L().Error(msg, args...) }

// Fatal logs at error level and exits.
func Fatal(msg string, args ...any) {
	L().Error(msg, args...)
	os.Exit(1)
}

// Fatalf logs a formatted message at error level and exits.
func Fatalf(format string, args ...any) {
	L().Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}
