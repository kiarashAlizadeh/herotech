package registry

import (
	"github.com/kiarashAlizadeh/herotech/internal/repository"
)

type Repositories struct {
	GuildRepository   repository.GuildRepository
	ItemRepository    repository.ItemRepository
	AuctionRepository repository.AuctionRepository
}

// GetRepositories returns singleton instances of repositories (Lazy Loading).
func (r *Registry) GetRepositories() *Repositories {
	if r.repositories == nil {
		// Extract both components from the internal db package structure cleanly
		pool := r.db.Pool
		queries := r.db.Queries

		r.repositories = &Repositories{
			GuildRepository:   repository.NewGuildRepository(pool, queries),
			ItemRepository:    repository.NewItemRepository(pool, queries),
			AuctionRepository: repository.NewAuctionRepository(pool, queries),
		}
	}
	return r.repositories
}

// GetGuildRepository safely fetches the initialized Guild infrastructure adapter
func (r *Registry) GetGuildRepository() repository.GuildRepository {
	return r.GetRepositories().GuildRepository
}

// GetItemRepository safely fetches the initialized Item infrastructure adapter
func (r *Registry) GetItemRepository() repository.ItemRepository {
	return r.GetRepositories().ItemRepository
}

// GetAuctionRepository safely fetches the initialized Auction infrastructure adapter
func (r *Registry) GetAuctionRepository() repository.AuctionRepository {
	return r.GetRepositories().AuctionRepository
}
