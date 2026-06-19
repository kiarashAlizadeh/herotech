package worker

import (
	"context"
	"log"
	"time"

	"github.com/kiarashAlizadeh/herotech/internal/repository"
)

type AuctionWorker struct {
	auctionRepo repository.AuctionRepository
}

func NewAuctionWorker(ar repository.AuctionRepository) *AuctionWorker {
	return &AuctionWorker{auctionRepo: ar}
}

func (w *AuctionWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping background auction worker cleanly...")
			return
		case <-ticker.C:
			w.processExpiredAuctions(ctx)
		}
	}
}

func (w *AuctionWorker) processExpiredAuctions(ctx context.Context) {
	auctions, _, err := w.auctionRepo.ListActive(ctx, 10000, 0)
	if err != nil {
		log.Printf("[Worker Error] failed to pull active auctions list: %v\n", err)
		return
	}

	now := time.Now()
	for _, a := range auctions {
		if now.After(a.EndsAt) {
			log.Printf("[Worker Action] Auction %s has expired. Initiating atomic closure sequence...\n", a.ID)
			if err := w.auctionRepo.FinalizeExpiredAuction(ctx, a.ID); err != nil {
				log.Printf("[Worker Failure] failed to close auction %s: %v\n", a.ID, err)
			}
		}
	}
}
