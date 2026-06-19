package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/kiarashAlizadeh/herotech/internal/handler"
)

func RegisterGuildRoutes(pathV1 *gin.RouterGroup, guildHandler *handler.GuildHandler) {
	guilds := pathV1.Group("/guilds")
	{
		guilds.POST("", guildHandler.CreateGuild)
		guilds.GET("", guildHandler.ListGuilds)
		guilds.GET("/:id/inventory", guildHandler.GetGuildInventory)
		guilds.GET("/:id/wallet", guildHandler.GetWalletSummary)
		guilds.POST("/:id/wallet/deposit", guildHandler.DepositGold)
	}
}
