package routes

import (
	"github.com/kiarashAlizadeh/herotech/internal/registry"

	"github.com/gin-gonic/gin"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRoutes(router *gin.Engine, reg *registry.Registry) {

	// Swagger UI Route
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.DefaultModelsExpandDepth(-1), // hides all models in ui
	))

	// Health Check Route (Public)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// ==========================================
	// API Version 1
	// ==========================================
	pathV1 := router.Group("/api/v1")

	// ==========================================
	// Register Domain Routes
	// ==========================================
	RegisterGuildRoutes(pathV1, reg.GetGuildHandler())
	RegisterItemRoutes(pathV1, reg.GetItemHandler())
	RegisterAuctionRoutes(pathV1, reg.GetAuctionHandler())

}
