package registry

import (
	"github.com/kiarashAlizadeh/herotech/internal/oracle"
	"github.com/kiarashAlizadeh/herotech/internal/service"
)

type Services struct {
	GuildService   service.GuildService
	ItemService    service.ItemService
	AuctionService service.AuctionService
}

// GetServices returns singleton instances of all business services (Lazy Loading).
func (r *Registry) GetServices() *Services {
	if r.services == nil {
		r.services = &Services{}

		// Resolve underlying repository dependencies from the container
		repos := r.GetRepositories()

		// Wire up the fault-tolerant price oracle architecture sequence
		mockOracle := oracle.NewMockPriceOracle()
		safeOracle := oracle.NewSafePriceOracle(mockOracle)

		r.services.GuildService = service.NewGuildService(repos.GuildRepository, repos.ItemRepository)

		// Inject the wrapped, resilient price oracle into the Item trading core
		r.services.ItemService = service.NewItemService(repos.ItemRepository, safeOracle)

		// AuctionService requires both auction and item repositories for multi-entity validations
		r.services.AuctionService = service.NewAuctionService(repos.AuctionRepository, repos.ItemRepository)
	}
	return r.services
}

// GetGuildService provides access to the Guild logic unit
func (r *Registry) GetGuildService() service.GuildService {
	return r.GetServices().GuildService
}

// GetItemService provides access to the Item trading logic unit
func (r *Registry) GetItemService() service.ItemService {
	return r.GetServices().ItemService
}

// GetAuctionService provides access to the Legendary bidding process logic unit
func (r *Registry) GetAuctionService() service.AuctionService {
	return r.GetServices().AuctionService
}
