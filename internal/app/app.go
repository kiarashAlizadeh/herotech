package app

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kiarashAlizadeh/herotech/internal/config"
	"github.com/kiarashAlizadeh/herotech/internal/db"
	"github.com/kiarashAlizadeh/herotech/internal/registry"
	"github.com/kiarashAlizadeh/herotech/internal/seeder"
	"github.com/kiarashAlizadeh/herotech/internal/server"
	"github.com/kiarashAlizadeh/herotech/internal/worker"
	"github.com/kiarashAlizadeh/herotech/pkg/logger"
)

type App struct {
	cfg          *config.Config
	registry     *registry.Registry
	srv          *http.Server
	workerCancel context.CancelFunc
}

func NewApp() (*App, error) {
	// 1. Load Config
	log.Println("📦 Loading config...")
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("config load error: %w", err)
	}

	// 2. Initialize Logger
	logger.InitLogger(cfg.Environment)
	slog.Info("Logger initialized successfully", slog.String("env", cfg.Environment))

	// 3. Initialize Database
	log.Println("🔌 Connecting to PostgreSQL database...")
	database, err := db.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("db connection failed: %w", err)
	}

	// 4. Initialize Registry (DI Container)
	log.Println("📁 Initializing registry...")
	reg := registry.NewRegistry(cfg, database)

	// 5. Setup Router
	log.Println("🌐 Setting up router...")
	router := server.SetupRouter(cfg, reg)

	// 6. Create HTTP Server
	httpServer := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	appInstance := &App{
		cfg:      cfg,
		registry: reg,
		srv:      httpServer,
	}

	// 7. Run Seeder
	dbSeeder := seeder.NewSeeder(cfg, reg)
	if err := dbSeeder.SeedAll(); err != nil {
		log.Printf("⚠️ Warning: System data seeding failed: %v", err)
	}

	return appInstance, nil
}

func (a *App) Run() error {
	// Create a dedicated cancellable context for background workers
	workerCtx, cancel := context.WithCancel(context.Background())
	a.workerCancel = cancel

	// Instantiate and Fire up the background Auction Ticker Worker
	log.Println("⏳ Starting background auction lifecycle worker thread...")
	auctionRepo := a.registry.GetRepositories().AuctionRepository
	auctionWorker := worker.NewAuctionWorker(auctionRepo)

	go auctionWorker.Start(workerCtx)

	// Start Server
	go func() {
		log.Printf("🚀 Server listening on port %s", a.cfg.ServerPort)
		if err := a.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen error: %v", err)
		}
	}()

	// Graceful Shutdown Setup
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	// Cancel the background workers immediately to release DB connections held by tickers
	if a.workerCancel != nil {
		a.workerCancel()
	}

	// Context for graceful HTTP shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown order is critical
	if err := a.srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	// Cleanup infra resources
	a.Cleanup()

	log.Println("👋 Server exited gracefully")
	return nil
}

func (a *App) Cleanup() {
	log.Println("🧹 Cleaning up resources...")

	// Shutdown Registry (Repository/Service layer)
	a.registry.Shutdown()
}
