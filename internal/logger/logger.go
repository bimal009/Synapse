package logger

import (
	"log"
	"log/slog"
	"os"
)

func New(env string) *slog.Logger {
	var handler slog.Handler
	if env == "" {
		log.Fatalf("the app env is empty")
	}

	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}

	return slog.New(handler)

}
