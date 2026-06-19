package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
)

// SetupGlobalMiddlewares registers all global middlewares in the correct order.
// Order matters:
// 1. Recovery (catch panics)
// 2. CustomLogger (log all requests)
// 3. CORS (handle preflight requests)
// 4. RateLimit (prevent abuse)
func SetupGlobalMiddlewares(router *gin.Engine) {
	// 1. Panic recovery (always first)
	router.Use(gin.Recovery())

	// 2. Request logging (after recovery, before anything else)
	router.Use(CustomLogger())

	// 3. CORS (handle preflight early)
	router.Use(CORSMiddleware())

	// 4. Timeout after 30 seconds (adjust based on your API)
	router.Use(TimeoutMiddleware(30 * time.Second))

}
