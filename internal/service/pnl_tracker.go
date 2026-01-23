package service

import (
	"log"
	"time"

	internalmath "my-tool-go/internal/math"
	"my-tool-go/internal/model"
)

// PnLTracker tracks and calculates PnL for completed trades
type PnLTracker struct {
	trades []internalmath.TradeResult
}

// NewPnLTracker creates a new PnL tracker instance
func NewPnLTracker() *PnLTracker {
	return &PnLTracker{
		trades: []internalmath.TradeResult{},
	}
}

// RecordClosedTrade records a completed trade and calculates its PnL
func (p *PnLTracker) RecordClosedTrade(signal *model.Signal, exitPrice float64, positionSize float64) internalmath.TradeResult {
	direction := "LONG"
	if signal.Type == model.SignalTypeShort {
		direction = "SHORT"
	}

	result := internalmath.CalculatePnL(signal.EntryPrice, exitPrice, direction, positionSize)
	result.Symbol = signal.Symbol
	result.EntryTime = signal.Timestamp
	result.ExitTime = time.Now()

	// Store for stats calculation
	p.trades = append(p.trades, result)

	log.Printf("📊 [PnL] %s closed - Direction: %s, Entry: %.4f, Exit: %.4f, PnL: %.2f (%.2f%%)",
		signal.Symbol, direction, signal.EntryPrice, exitPrice, result.PnL, result.PnLPercent)

	return result
}

// GetStats returns overall trading statistics
func (p *PnLTracker) GetStats() internalmath.PnLStats {
	return internalmath.CalculatePnLStats(p.trades)
}

// GetMaxDrawdown calculates maximum drawdown from equity curve
func (p *PnLTracker) GetMaxDrawdown(initialEquity float64) float64 {
	if len(p.trades) == 0 {
		return 0
	}

	// Build equity curve
	equity := make([]float64, len(p.trades)+1)
	equity[0] = initialEquity

	for i, trade := range p.trades {
		equity[i+1] = equity[i] + trade.PnL
	}

	return internalmath.CalculateMaxDrawdown(equity)
}

// GetPerformanceSummary logs a summary of trading performance
func (p *PnLTracker) GetPerformanceSummary() {
	stats := p.GetStats()

	log.Println("==========================================")
	log.Println("📈 TRADING PERFORMANCE SUMMARY")
	log.Println("==========================================")
	log.Printf("Total Trades: %d", stats.TotalTrades)
	log.Printf("Winning: %d | Losing: %d", stats.WinningTrades, stats.LosingTrades)
	log.Printf("Win Rate: %.2f%%", stats.WinRate)
	log.Printf("Total PnL: $%.2f", stats.TotalPnL)
	log.Printf("Avg Win: $%.2f | Avg Loss: $%.2f", stats.AvgWin, stats.AvgLoss)
	log.Printf("Largest Win: $%.2f | Largest Loss: $%.2f", stats.LargestWin, stats.LargestLoss)
	log.Printf("Profit Factor: %.2f", stats.ProfitFactor)
	log.Printf("Expected Value: $%.2f", stats.ExpectedValue)
	log.Println("------------------------------------------")
	log.Println("📐 PROBABILITY METRICS")
	log.Println("------------------------------------------")
	log.Printf("Sharpe Ratio: %.2f", stats.SharpeRatio)
	log.Printf("Risk of Ruin: %.2f%%", stats.RiskOfRuin*100)
	log.Printf("Optimal Position Size (Kelly): %.2f%%", stats.OptimalF)
	log.Println("==========================================")
}
