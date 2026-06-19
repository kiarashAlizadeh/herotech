package registry

import (
	"github.com/kiarashAlizadeh/herotech/internal/handler"
)

type Handlers struct {
	GuildHandler   *handler.GuildHandler
	ItemHandler    *handler.ItemHandler
	AuctionHandler *handler.AuctionHandler
}

// GetHandlers returns singleton instances of all HTTP handlers (Lazy Loading).
func (r *Registry) GetHandlers() *Handlers {
	if r.handlers == nil {
		r.handlers = &Handlers{}

		// Resolve functional service layer abstractions
		services := r.GetServices()

		r.handlers.GuildHandler = handler.NewGuildHandler(services.GuildService)
		r.handlers.ItemHandler = handler.NewItemHandler(services.ItemService)
		r.handlers.AuctionHandler = handler.NewAuctionHandler(services.AuctionService)
	}
	return r.handlers
}

// GetGuildHandler provides access to the Guild HTTP endpoint controller
func (r *Registry) GetGuildHandler() *handler.GuildHandler {
	return r.GetHandlers().GuildHandler
}

// GetItemHandler provides access to the Item trade operations controller
func (r *Registry) GetItemHandler() *handler.ItemHandler {
	return r.GetHandlers().ItemHandler
}

// GetAuctionHandler provides access to the Auction bidding controller
func (r *Registry) GetAuctionHandler() *handler.AuctionHandler {
	return r.GetHandlers().AuctionHandler
}
