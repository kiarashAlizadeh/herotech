package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/kiarashAlizadeh/herotech/internal/handler"
)

func RegisterItemRoutes(pathV1 *gin.RouterGroup, itemHandler *handler.ItemHandler) {
	items := pathV1.Group("/items")
	{
		items.POST("", itemHandler.ListItem)
		items.GET("", itemHandler.ListAvailable)
		items.GET("/:id", itemHandler.GetItem)
		items.POST("/:id/buy", itemHandler.BuyItemDirectly) // Direct purchase endpoint
	}
}
