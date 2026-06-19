package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// CustomLogger returns a structured logging middleware for Gin.
// It logs details of each HTTP request and captures any internal errors attached to the context.
func CustomLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Append query parameters to the request path if they exist.
		if raw != "" {
			path = path + "?" + raw
		}

		// Process the request by passing control to the next middleware or handler.
		c.Next()

		// Collect request details after the handler has finished executing.
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		// Check if the handler attached any internal errors using c.Error(err).
		if len(c.Errors) > 0 {
			// Iterate over all attached errors and log them with full details.
			for _, e := range c.Errors {
				slog.Error("Request failed with internal error",
					slog.Int("status", statusCode),
					slog.String("method", method),
					slog.String("path", path),
					slog.Duration("latency", latency),
					slog.String("client_ip", clientIP),
					// Log the exact underlying error (e.g., DB connection failure) for debugging.
					slog.String("detailed_error", e.Err.Error()),
				)
			}
			return
		}

		// If no internal errors were attached, log a structured success message.
		slog.Info("Request processed successfully",
			slog.Int("status", statusCode),
			slog.String("method", method),
			slog.String("path", path),
			slog.Duration("latency", latency),
			slog.String("client_ip", clientIP),
		)
	}
}
