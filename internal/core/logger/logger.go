package logger

import (
	"log/slog"
	"os"
)

// Logger is the application logger interface
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

// SlogLogger wraps slog.Logger to implement our Logger interface
type SlogLogger struct {
	logger *slog.Logger
}

// New creates a new logger with the specified level
func New(level string) Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	return &SlogLogger{
		logger: slog.New(handler),
	}
}

// NewText creates a logger with text output (for development)
func NewText(level string) Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	return &SlogLogger{
		logger: slog.New(handler),
	}
}

func (l *SlogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l *SlogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *SlogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *SlogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

func (l *SlogLogger) With(args ...any) Logger {
	return &SlogLogger{
		logger: l.logger.With(args...),
	}
}

// Helper functions for common fields
func AppID(id string) slog.Attr {
	return slog.String("app_id", id)
}

func DeploymentID(id string) slog.Attr {
	return slog.String("deployment_id", id)
}

func ContainerID(id string) slog.Attr {
	return slog.String("container_id", id)
}

func Err(err error) slog.Attr {
	return slog.Any("error", err)
}
