package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func New(level, logDir string) *slog.Logger {
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		slog.Error("не удалось создать папку логов", "err", err)
	}

	logFile, err := os.OpenFile(
		filepath.Join(logDir, "app.log"),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0o644,
	)
	if err != nil {
		slog.Error("не удалось открыть файл логов", "err", err)
		logFile = os.Stdout
	}

	opts := &slog.HandlerOptions{Level: parseLevel(level)}
	writer := io.MultiWriter(os.Stdout, logFile)
	return slog.New(slog.NewJSONHandler(writer, opts))
}

func parseLevel(s string) slog.Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
