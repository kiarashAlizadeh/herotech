package logger

import (
	"log/slog"
	"os"
)

// InitLogger configures the global slog logger based on the application environment.
func InitLogger(env string) {
	var handler slog.Handler

	if env == "development" {
		// Use a human-readable text format for development purposes.
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		// Use structured JSON format for production environments.
		// This is ideal for log aggregation tools like Kibana(ELK), Datadog, or Grafana.
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}

	logger := slog.New(handler)

	// Set the newly configured logger as the default logger for the entire application.
	// This allows using slog.Info() or slog.Error() directly anywhere in the code.
	slog.SetDefault(logger)
}
