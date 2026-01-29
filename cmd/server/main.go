package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mrcrypto-go/internal/config"
	"mrcrypto-go/internal/loader"
	"mrcrypto-go/internal/monitor"
	"mrcrypto-go/internal/service"
)

func main() {
	// Global panic recovery
	defer service.RecoverAndLog("main")

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
	shutdownChan := make(chan bool, 1)
	go func() {
		defer service.RecoverAndLog("shutdown handler")
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("\n🛑 Received shutdown signal...")

		// Graceful shutdown with timeout
		shutdownTimer := time.NewTimer(30 * time.Second)
		go func() {
			<-shutdownTimer.C
			log.Println("⚠️  Shutdown timeout - forcing exit")
			os.Exit(1)
		}()

		databaseService.Close()
		shutdownTimer.Stop()
		shutdownChan <- true
		os.Exit(0)
	}()

	// Start the loader (blocks indefinitely)
	log.Println("🚀 Bot is now running...")
	loaderService.Start()
}
