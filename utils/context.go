package utils

import (
	"context"
	"log/slog"
)

type key uint

const (
	loggerKey key = iota
	verboseKey
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

func Verbose(ctx context.Context) bool {
	v, ok := ctx.Value(verboseKey).(bool)
	return ok && v
}

func WithVerbose(ctx context.Context, v bool) context.Context {
	return context.WithValue(ctx, verboseKey, v)
}
