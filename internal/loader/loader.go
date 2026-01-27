package loader

import (
	"log"
	"time"

	"mrcrypto-go/internal/model"
	"mrcrypto-go/internal/monitor"
	"mrcrypto-go/internal/service"
	"mrcrypto-go/internal/worker"

	"github.com/robfig/cron/v3"
)

type Loader struct {
	binance       *service.BinanceService
	strategy      *service.StrategyService
	ai            *service.AIService
	telegram      *service.TelegramService
	database      *service.DatabaseService
	signalMonitor *monitor.SignalMonitor
	symbolManager *service.SymbolManager
	isPolling     bool
}

// NewLoader creates a new loader instance
// NewLoader creates a new loader instance
func NewLoader(
	binance *service.BinanceService,
	strategy *service.StrategyService,
	ai *service.AIService,
	telegram *service.TelegramService,
	database *service.DatabaseService,
	signalMonitor *monitor.SignalMonitor,
	symbolManager *service.SymbolManager,
) *Loader {
	return &Loader{
		binance:       binance,
		strategy:      strategy,
		ai:            ai,
		telegram:      telegram,
		database:      database,
		signalMonitor: signalMonitor,
		symbolManager: symbolManager,
		isPolling:     false,
	}
}

// Start begins the scheduled polling
func (l *Loader) Start() {
	log.Println("🚀 Starting Trading Signal Loader...")

	c := cron.New()

	// Run signal generation every 1 minute
	// Monitoring is now "piggybacked" on this poll cycle
	c.AddFunc("@every 1m", func() {
		if l.isPolling {
			log.Println("⏭️  Skipping cycle - previous poll still running")
			return
		}

		l.poll()
	})

	c.Start()

	log.Println("⏰ Scheduler started - scanning & monitoring every 1 minute")

	// Keep the program running
	select {}
}

// poll executes one complete polling cycle
func (l *Loader) poll() {
	l.isPolling = true
	defer func() {
		l.isPolling = false
	}()

	log.Println("===========================================")
	log.Printf("🔄 Polling started at %s", time.Now().Format("15:04:05"))
	log.Println("===========================================")

	// Fetch watchlist
	symbols, err := l.symbolManager.GetWatchlist()
	if err != nil {
		log.Printf("❌ Failed to fetch watchlist: %v", err)
		return
	}

	if len(symbols) == 0 {
		log.Println("⚠️  Watchlist is empty. Add symbols using /symbol add")
		return
	}

	log.Printf("📊 Scanning %d symbols", len(symbols))

	// Create worker pool with 10 workers
	log.Printf("🔄 [Loader] Creating worker pool with 10 workers...")
	pool := worker.NewPool(10, l.strategy)
	pool.Start()

	// Add all symbols as jobs
	log.Printf("⏳ [Loader] Distributing %d jobs to workers...", len(symbols))
	for _, symbol := range symbols {
		pool.AddJob(symbol)
	}

	// Wait for all workers to complete and collect signals AND prices
	signals, prices := pool.Wait()

	// PIGGYBACK MONITORING: Check active signals using the fresh prices we just fetched
	if l.signalMonitor != nil && len(prices) > 0 {
		log.Println("👀 [Loader] Triggering Piggyback Monitoring...")
		l.signalMonitor.CheckActiveSignalsAgainstPrices(prices)
	}

	log.Printf("📈 Generated %d potential signals", len(signals))

	if len(signals) == 0 {
		log.Println("===========================================")
		log.Println("✨ Polling complete - 0 signals generated")
		log.Println("===========================================")
		return
	}

	// Filter signals by cooldown first
	log.Printf("⏳ [Loader] Filtering %d signals by cooldown (4h)...", len(signals))
	var validForAI []*model.Signal
	for _, signal := range signals {
		if l.database.CheckCooldown(signal.Symbol, 4*time.Hour) {
			log.Printf("⏱️  %s - Skipped (cooldown)", signal.Symbol)
			continue
		}
		validForAI = append(validForAI, signal)
	}

	if len(validForAI) == 0 {
		log.Println("===========================================")
		log.Printf("✨ Polling complete - All %d signals in cooldown", len(signals))
		log.Println("===========================================")
		return
	}

	log.Printf("✅ [Loader] %d signals passed cooldown filter", len(validForAI))

	// BATCH AI VALIDATION (Optimized - Single API Call)
	log.Printf("🤖 Batch validating %d signals with AI...", len(validForAI))
	aiResults, err := l.ai.BatchValidateSignals(validForAI)
	if err != nil {
		log.Printf("❌ Batch AI validation failed: %v", err)
		return
	}

	// Process validated signals
	log.Printf("⏳ [Loader] Processing %d AI validation results...", len(aiResults))
	validSignals := 0
	for idx, signal := range validForAI {
		if idx >= len(aiResults) {
			continue
		}

		result := aiResults[idx]
		signal.AIScore = result.Score
		signal.AIConfidence = result.Confidence
		signal.AITier = result.Tier
		signal.AIReason = result.Reason

		// Strict Score Filtering (User Request: Both > 70)
		if result.Score <= 70 || signal.ConfluenceScore <= 70 {
			log.Printf("❌ %s - Scores too low (AI: %d, System: %d). Both must be > 70.",
				signal.Symbol, result.Score, signal.ConfluenceScore)
			continue
		}

		log.Printf("✅ %s - Valid signal! AI Score: %d/100", signal.Symbol, result.Score)

		// Save to database
		if err := l.database.SaveSignal(signal); err != nil {
			log.Printf("⚠️  %s - Failed to save signal: %v", signal.Symbol, err)
			continue
		}

		// Send to Telegram
		if err := l.telegram.SendSignal(signal); err != nil {
			log.Printf("⚠️  %s - Failed to send Telegram notification: %v", signal.Symbol, err)
			continue
		}

		validSignals++
	}

	log.Println("===========================================")
	log.Printf("✨ Polling complete - %d valid signals sent", validSignals)
	log.Println("===========================================")
}
