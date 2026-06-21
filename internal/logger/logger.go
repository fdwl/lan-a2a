package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

var defaultLogger *slog.Logger

func Init(level string, output io.Writer) {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	if output == nil {
		output = os.Stderr
	}

	defaultLogger = slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{
		Level: lvl,
	}))
	slog.SetDefault(defaultLogger)
}

func Get() *slog.Logger {
	if defaultLogger == nil {
		Init("info", nil)
	}
	return defaultLogger
}

func With(args ...any) *slog.Logger {
	return Get().With(args...)
}

func Debug(msg string, args ...any) { Get().Debug(msg, args...) }
func Info(msg string, args ...any)  { Get().Info(msg, args...) }
func Warn(msg string, args ...any)  { Get().Warn(msg, args...) }
func Error(msg string, args ...any) { Get().Error(msg, args...) }

func DebugCtx(ctx context.Context, msg string, args ...any) { Get().DebugContext(ctx, msg, args...) }
func InfoCtx(ctx context.Context, msg string, args ...any)  { Get().InfoContext(ctx, msg, args...) }
func WarnCtx(ctx context.Context, msg string, args ...any)  { Get().WarnContext(ctx, msg, args...) }
func ErrorCtx(ctx context.Context, msg string, args ...any) { Get().ErrorContext(ctx, msg, args...) }

func WithComponent(name string) *slog.Logger {
	return Get().With("component", name)
}
