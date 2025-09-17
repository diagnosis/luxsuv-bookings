package logger

import (
	"context"
	"log/slog"
	"os"
)

type contextKey string

const (
	RequestIDKey contextKey = "request_id"
	UserIDKey    contextKey = "user_id"
	ServiceKey   contextKey = "service"
)

var defaultLogger *slog.Logger

func init() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	
	if os.Getenv("LOG_LEVEL") == "debug" {
		opts.Level = slog.LevelDebug
	}
	
	handler := slog.NewJSONHandler(os.Stdout, opts)
	defaultLogger = slog.New(handler)
}

func Default() *slog.Logger {
	return defaultLogger
}

func WithContext(ctx context.Context) *slog.Logger {
	logger := defaultLogger
	
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		logger = logger.With("request_id", requestID)
	}
	
	if userID := ctx.Value(UserIDKey); userID != nil {
		logger = logger.With("user_id", userID)
	}
	
	if service := ctx.Value(ServiceKey); service != nil {
		logger = logger.With("service", service)
	}
	
	return logger
}

func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

func InfoContext(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Info(msg, args...)
}

func ErrorContext(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Error(msg, args...)
}

func DebugContext(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Debug(msg, args...)
}

func WarnContext(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Warn(msg, args...)
}