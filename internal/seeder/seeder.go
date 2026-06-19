package seeder

import (
	"context"
	"log"
	"time"

	"github.com/kiarashAlizadeh/herotech/internal/config"
	"github.com/kiarashAlizadeh/herotech/internal/registry"
)

// Seeder orchestrates all database seeding operations for the Dragon Marketplace.
type Seeder struct {
	cfg *config.Config
	reg *registry.Registry
}

// NewSeeder creates and returns a new dynamic Seeder instance.
func NewSeeder(cfg *config.Config, reg *registry.Registry) *Seeder {
	return &Seeder{
		cfg: cfg,
		reg: reg,
	}
}

// SeedAll executes all marketplace seed routines sequentially inside a safe timeout context.
func (s *Seeder) SeedAll() error {
	// Global context limit to prevent system boot hangs during seeding
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	log.Println("🌱 Starting dragon marketplace database seeding pipeline...")

	// Execute the core marketplace mock sequence
	if err := s.SeedMarketplaceData(ctx); err != nil {
		log.Printf("❌ Seeding pipeline aborted due to failure: %v", err)
		return err
	}

	log.Println("✨ Database seeding pipeline completed successfully!")
	return nil
}
