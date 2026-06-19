package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// RecoveryMiddleware recovers from any panic, logs the error and stack trace using slog,
// and returns a 500 Internal Server Error response to the client.
// It prevents the server from crashing due to unhandled panics.
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic, stack trace, and request details using structured logging
				slog.Error("PANIC RECOVERED",
					slog.Any("error", err),
					slog.String("stack_trace", string(debug.Stack())),
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
					slog.String("client_ip", c.ClientIP()),
				)

				// Return a clean, safe error response to the client
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
			}
		}()

		// Process the request
		c.Next()
	}
}
