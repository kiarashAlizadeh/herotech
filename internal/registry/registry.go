package registry

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kiarashAlizadeh/herotech/internal/config"
	"github.com/kiarashAlizadeh/herotech/internal/db"
	"github.com/kiarashAlizadeh/herotech/internal/db/sqlc"
)

type Registry struct {
	cfg          *config.Config
	db           *db.DB
	repositories *Repositories
	services     *Services
	handlers     *Handlers
}

// NewRegistry creates a new instance of the container.
// Only stores dependencies here, not the connection.
func NewRegistry(cfg *config.Config, database *db.DB) *Registry {
	return &Registry{
		cfg: cfg,
		db:  database,
	}
}

// GetConfig returns the configuration
func (r *Registry) GetConfig() *config.Config {
	return r.cfg
}

// GetDB provides access to *sqlc.Queries (for lower layers)
func (r *Registry) GetDB() *sqlc.Queries {
	return r.db.Queries
}

// GetPool provides access to the raw pgxpool.Pool instance needed for multi-statement transactions
func (r *Registry) GetPool() *pgxpool.Pool {
	return r.db.Pool
}

// Shutdown releases resources
func (r *Registry) Shutdown() {
	if r.db != nil {
		r.db.Close()
	}
}
