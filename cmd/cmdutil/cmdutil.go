package cmdutil

import (
	"log"
	"log/slog"
	"os"
	"strings"
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
