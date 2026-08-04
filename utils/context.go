package utils

import (
	"context"
	"log/slog"
)

type key uint

const (
	loggerKey key = iota
)

func Logger(ctx context.Context) *slog.Logger {
	l, ok := ctx.Value(loggerKey).(*slog.Logger)
	if !ok {
		return slog.Default()
	}
	return l
}

func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}
