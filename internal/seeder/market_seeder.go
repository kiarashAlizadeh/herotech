package seeder

import (
	"context"
	"log"
	"time"

	"github.com/kiarashAlizadeh/herotech/internal/domain"
)

// SeedMarketplaceData populates initial guilds, trading items, and active auctions if empty.
func (s *Seeder) SeedMarketplaceData(ctx context.Context) error {
	repos := s.reg.GetRepositories()

	// 1. Idempotency Check: If we already have items listed, skip seeding to avoid duplicates
	existingItems, _, err := repos.ItemRepository.ListAvailable(ctx, nil, 1, 0)
	if err == nil && len(existingItems) > 0 {
		log.Println("ℹ️ Marketplace already contains data listings. Skipping seeder sequence.")
		return nil
	}

	log.Println("⚡ Populating fresh genesis marketplace data profiles...")

	// 2. Seed Initial Guild Treasuries
	slayerGuild, err := repos.GuildRepository.Create(ctx, "Dragon Slayers Order", 50000)
	if err != nil {
		return err
	}
	// Give them initial starting liquidity
	_, _ = repos.GuildRepository.DepositGold(ctx, slayerGuild.ID, 100000)

	shadowGuild, err := repos.GuildRepository.Create(ctx, "Shadow Whispers Syndicate", 30000)
	if err != nil {
		return err
	}
	_, _ = repos.GuildRepository.DepositGold(ctx, shadowGuild.ID, 75000)

	log.Printf("✅ Seeded Guilds: %s and %s\n", slayerGuild.Name, shadowGuild.Name)

	// 3. Seed Fixed Price Items (Common & Rare)
	commonItemPrice := int64(150)
	_, err = repos.ItemRepository.Create(ctx, "Iron Broadsword", domain.ItemTypeCommon, slayerGuild.ID, 100, commonItemPrice)
	if err != nil {
		return err
	}

	rareItemPrice := int64(1200)
	_, err = repos.ItemRepository.Create(ctx, "Mana Infused Chestplate", domain.ItemTypeRare, shadowGuild.ID, 800, rareItemPrice)
	if err != nil {
		return err
	}

	log.Println("✅ Seeded initial Common and Rare inventory assets.")

	// 4. Seed Legendary Item & Instantly Open an Auction Board for it
	// Legendary assets holding NO direct list price (0)
	legendaryItem, err := repos.ItemRepository.Create(ctx, "Vorynthax Frostmaw Staff", domain.ItemTypeLegendary, slayerGuild.ID, 5000, 0)
	if err != nil {
		return err
	}

	log.Printf("✅ Minted Legendary Asset: %s\n", legendaryItem.Name)

	// Open a competitive auction for this legendary item owned by Slayer Guild
	auctionDuration := 24 * time.Hour
	startingBidPrice := int64(6500)

	activeAuction, err := repos.AuctionRepository.Create(ctx, legendaryItem.ID, slayerGuild.ID, startingBidPrice, auctionDuration)
	if err != nil {
		log.Printf("⚠️ Warning: Failed to automatically spin up auction for legendary item: %v", err)
		return nil // Don't crash the entire app if just the auction init fails
	}

	log.Printf("🚀 Legacy Auction Board Live! Auction ID: %s (Starting Price: %d Gold)\n", activeAuction.ID, startingBidPrice)

	return nil
}
