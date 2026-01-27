package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"mrcrypto-go/internal/config"
	"mrcrypto-go/internal/loader"
	"mrcrypto-go/internal/monitor"
	"mrcrypto-go/internal/service"
)

func main() {
	// Load configuration
	config.Load()

	log.Println("🔧 Initializing services...")

	// Initialize services
	// Initialize services
	binanceService := service.NewBinanceService()
	strategyService := service.NewStrategyService(binanceService)
	aiService := service.NewAIService()

	// Create Database service first as SymbolManager needs it
	databaseService, err := service.NewDatabaseService()
	if err != nil {
		log.Fatalf("❌ Failed to initialize Database service: %v", err)
	}
	defer databaseService.Close()

	// Initialize Symbol Manager
	symbolManager := service.NewSymbolManager(databaseService.GetDB())

	telegramService, err := service.NewTelegramService(binanceService, symbolManager)
	if err != nil {
		log.Fatalf("❌ Failed to initialize Telegram service: %v", err)
	}

	// Initialize Signal Monitor for active trade monitoring
	signalMonitor := monitor.NewSignalMonitor(
		databaseService.GetDB(),
		binanceService,
		telegramService,
	)

	log.Println("✅ All services initialized successfully")

	// Create and start loader
	loaderService := loader.NewLoader(
		binanceService,
		strategyService,
		aiService,
		telegramService,
		databaseService,
		signalMonitor,
		symbolManager,
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
