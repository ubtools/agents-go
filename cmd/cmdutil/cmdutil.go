package cmdutil

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
)

func SlogLevelFromString(lvl string) (programLevel slog.Level) {
	switch strings.ToUpper(lvl) {
	case "DEBUG":
		programLevel = slog.LevelDebug
	case "INFO":
		programLevel = slog.LevelInfo
	case "WARN":
		programLevel = slog.LevelWarn
	case "ERROR":
		programLevel = slog.LevelError
	default:
		log.Fatalf("Invalid log level %s", lvl)
	}
	return programLevel
}

func InitLogger(lvlStr string) {
	var lvl = SlogLevelFromString(lvlStr)
	h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(h))
}

// interceptorLogger adapts go-kit logger to interceptor logger.
func InterceptorLogger(logger *slog.Logger) logging.Logger {
	if logger == nil {
		logger = slog.Default()
	}
	return logging.LoggerFunc(func(_ context.Context, lvl logging.Level, msg string, fields ...any) {
		switch lvl {
		case logging.LevelDebug:
			logger.Debug(msg, fields...)
		case logging.LevelInfo:
			logger.Info(msg, fields...)
		case logging.LevelWarn:
			logger.Warn(msg, fields...)
		case logging.LevelError:
			logger.Debug(msg, fields...)
		default:
			panic(fmt.Sprintf("unknown level %v", lvl))
		}
	})
}
