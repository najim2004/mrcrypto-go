package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"mrcrypto-go/internal/config"
	"mrcrypto-go/internal/loader"
	"mrcrypto-go/internal/service"
)

func main() {
	// Load configuration
	config.Load()

	log.Println("🔧 Initializing services...")

	// Initialize services
	binanceService := service.NewBinanceService()
	strategyService := service.NewStrategyService(binanceService)
	aiService := service.NewAIService()

	telegramService, err := service.NewTelegramService(binanceService)
	if err != nil {
		log.Fatalf("❌ Failed to initialize Telegram service: %v", err)
	}

	databaseService, err := service.NewDatabaseService()
	if err != nil {
		log.Fatalf("❌ Failed to initialize Database service: %v", err)
	}
	defer databaseService.Close()

	log.Println("✅ All services initialized successfully")

	// Create and start loader
	loaderService := loader.NewLoader(
		binanceService,
		strategyService,
		aiService,
		telegramService,
		databaseService,
	)

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("\n🛑 Received shutdown signal...")
		databaseService.Close()
		os.Exit(0)
	}()

	// Start the loader (blocks indefinitely)
	loaderService.Start()
}
