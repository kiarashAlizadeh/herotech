package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// TimeoutMiddleware sets a deadline for the request.
// If the handler does not complete within the given duration,
// it returns a 503 Service Unavailable error.
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		done := make(chan bool)
		go func() {
			c.Next()
			done <- true
		}()

		select {
		case <-done:
			// Handler completed within timeout
			return
		case <-ctx.Done():
			// Timeout exceeded
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "Request timeout",
			})
		}
	}
}
