package commons

import (
	"context"
	"log/slog"
)

const CONTEXT_KEY_LOGGER = "logger"
const CONTEXT_KEY_CHAINID = "chain"

func LoggerFromContext(ctx context.Context, args ...any) *slog.Logger {
	var log *slog.Logger
	log, ok := ctx.Value(CONTEXT_KEY_LOGGER).(*slog.Logger)
	if log == nil || !ok {
		log = slog.Default()
	}
	return log.With(args...)
}

func ContextWithLogger(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, CONTEXT_KEY_LOGGER, log)
}

func LabelsToMap(labels []any) map[string]string {
	if len(labels)%2 != 0 {
		panic("labels must be even")
	}
	m := make(map[string]string, len(labels)/2)
	for i := 0; i < len(labels); i += 2 {
		key, kok := labels[i].(string)
		value, vok := labels[i+1].(string)
		if kok && vok {
			m[key] = value
		}
	}
	return m
}
