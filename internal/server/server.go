package server

import (
	"github.com/kiarashAlizadeh/herotech/internal/config"
	"github.com/kiarashAlizadeh/herotech/internal/middleware"
	"github.com/kiarashAlizadeh/herotech/internal/registry"
	"github.com/kiarashAlizadeh/herotech/internal/routes"

	"github.com/gin-gonic/gin"
)

// SetupRouter performs all Gin configuration (middlewares + routes)
// and returns a ready-to-run gin.Engine.
func SetupRouter(cfg *config.Config, reg *registry.Registry) *gin.Engine {
	// Set Mode
	if cfg.Environment == "development" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Register global middlewares
	middleware.SetupGlobalMiddlewares(router)

	// Register all routes
	registerRoutes(router, reg)

	return router
}

// registerRoutes is an internal function for registering routes (private)
func registerRoutes(router *gin.Engine, reg *registry.Registry) {
	routes.SetupRoutes(router, reg)
}
