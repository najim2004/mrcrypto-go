package main

import (
	"encoding/json"
	"fmt"
	"log"

	"mrcrypto-go/internal/config"
	"mrcrypto-go/internal/service"
)

func main() {
	// Load config to get API keys if needed
	config.Load()

	log.Println("🧪 Starting Strategy Verification...")

	// Initialize services
	binanceService := service.NewBinanceService()
	strategyService := service.NewStrategyService(binanceService)

	symbol := "ETHUSDT"
	log.Printf("🔍 Evaluating %s...", symbol)

	signal, err := strategyService.EvaluateSymbol(symbol)
	if err != nil {
		log.Fatalf("❌ Error evaluating symbol: %v", err)
	}

	if signal == nil {
		log.Println("⚠️  No signal generated (conditions not met)")
		return
	}

	// Print JSON for inspection
	jsonData, _ := json.MarshalIndent(signal, "", "  ")
	fmt.Println(string(jsonData))

	fmt.Println("\n✅ Verification Successful!")
	fmt.Printf("BTC Correlation: %s\n", signal.TechnicalContext.BTCCorrelation)
	fmt.Printf("SMC FVG: In Gap? %s (Type: %s)\n", boolToYesNo(signal.TechnicalContext.FVGType != ""), signal.TechnicalContext.FVGType)
	fmt.Printf("SMC OB: In Block? %s (Type: %s)\n", boolToYesNo(signal.TechnicalContext.OBType != ""), signal.TechnicalContext.OBType)
	fmt.Printf("POC: %.2f (Dist: %.2f%%)\n", signal.TechnicalContext.POC, signal.TechnicalContext.POCDistance)
}

func boolToYesNo(b bool) string {
	if b {
		return "YES"
	}
	return "NO"
}
