package main

import (
	"log"

	_ "github.com/kiarashAlizadeh/herotech/docs"
	"github.com/kiarashAlizadeh/herotech/internal/app"
)

// @title           DRAGON MARKET PLACE API
// @version         1.0
// @description     Core infrastructure backend for managing Guild Wallets, Item Trading, and Legendary Auctions.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@Aethoria.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1
func main() {
	log.Println("🚀 Starting application...")
	application, err := app.NewApp()
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	log.Println("✅ App initialized, running...")
	if err := application.Run(); err != nil {
		log.Fatalf("Error running app: %v", err)
	}
}
