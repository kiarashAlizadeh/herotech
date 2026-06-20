package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/kiarashAlizadeh/herotech/internal/handler"
)

func RegisterAuctionRoutes(pathV1 *gin.RouterGroup, auctionHandler *handler.AuctionHandler) {
	auctions := pathV1.Group("/auctions")
	{
		auctions.POST("", auctionHandler.StartAuction)
		auctions.GET("", auctionHandler.ListActiveAuctions)
		auctions.GET("/:id", auctionHandler.GetAuction)
		auctions.POST("/:id/bid", auctionHandler.PlaceBid)
	}
}
