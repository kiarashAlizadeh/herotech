package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kiarashAlizadeh/herotech/internal/config"
	"github.com/kiarashAlizadeh/herotech/internal/db/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps the connection pool and queries.
// It's responsible for connecting, pinging, and closing.
type DB struct {
	Pool    *pgxpool.Pool
	Queries *sqlc.Queries
}

// New creates a new database connection and initializes sqlc.Queries.
func New(cfg *config.Config) (*DB, error) {
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, err
	}

	// Configuration for Connection Pool
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	// DB Connection test
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	log.Println("✅ Database connection established")

	return &DB{
		Pool:    pool,
		Queries: sqlc.New(pool),
	}, nil
}

// Close gracefully closes the database connection pool.
func (d *DB) Close() {
	if d.Pool != nil {
		d.Pool.Close()
		log.Println("🔌 Database connection closed")
	}
}
