package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/kiarashAlizadeh/herotech/internal/handler"
)

func RegisterItemRoutes(pathV1 *gin.RouterGroup, itemHandler *handler.ItemHandler) {
	items := pathV1.Group("/items")
	{
		items.POST("", itemHandler.CreateItem)           // Pure minting/dropping action
		items.POST("/:id/list", itemHandler.ListForSale) // Lists an existing item on the market
		items.GET("", itemHandler.ListAvailable)         // Streams marketplace window
		items.GET("/:id", itemHandler.GetItem)
		items.POST("/:id/buy", itemHandler.BuyItemDirectly)
	}
}
