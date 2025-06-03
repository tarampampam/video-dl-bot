package logger

import (
	"errors"
	"log/slog"
	"os"
)

// New creates a new slog.Logger based on the level and format.
func New(l Level, f Format) (*slog.Logger, error) {
	var (
		handler slog.Handler
		opts    slog.HandlerOptions
	)

	// Map custom level to slog.Level
	var slogLevel slog.Level

	switch l {
	case DebugLevel:
		slogLevel = slog.LevelDebug
	case InfoLevel:
		slogLevel = slog.LevelInfo
	case WarnLevel:
		slogLevel = slog.LevelWarn
	case ErrorLevel:
		slogLevel = slog.LevelError
	default:
		return nil, errors.New("unsupported logging level")
	}

	opts = slog.HandlerOptions{
		Level: slogLevel,
	}

	switch f {
	case ConsoleFormat:
		handler = slog.NewTextHandler(os.Stderr, &opts)
	case JSONFormat:
		handler = slog.NewJSONHandler(os.Stderr, &opts)
	default:
		return nil, errors.New("unsupported logging format")
	}

	return slog.New(handler), nil
}
