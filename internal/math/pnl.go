package math

import "time"

// TradeResult represents a completed trade
type TradeResult struct {
	Symbol     string
	EntryPrice float64
	ExitPrice  float64
	EntryTime  time.Time
	ExitTime   time.Time
	Direction  string // LONG or SHORT
	PnL        float64
	PnLPercent float64
	IsWin      bool
}

// PnLStats represents PnL statistics
type PnLStats struct {
	TotalTrades   int
	WinningTrades int
	LosingTrades  int
	WinRate       float64
	TotalPnL      float64
	AvgWin        float64
	AvgLoss       float64
	LargestWin    float64
	LargestLoss   float64
	ProfitFactor  float64
	ExpectedValue float64
}

// CalculatePnL calculates profit/loss for a trade
func CalculatePnL(entryPrice, exitPrice float64, direction string, positionSize float64) TradeResult {
	var pnl, pnlPercent float64
	var isWin bool

	if direction == "LONG" {
		pnl = (exitPrice - entryPrice) * positionSize
		pnlPercent = ((exitPrice - entryPrice) / entryPrice) * 100
		isWin = exitPrice > entryPrice
	} else { // SHORT
		pnl = (entryPrice - exitPrice) * positionSize
		pnlPercent = ((entryPrice - exitPrice) / entryPrice) * 100
		isWin = exitPrice < entryPrice
	}

	return TradeResult{
		EntryPrice: entryPrice,
		ExitPrice:  exitPrice,
		Direction:  direction,
		PnL:        pnl,
		PnLPercent: pnlPercent,
		IsWin:      isWin,
	}
}

// CalculatePnLStats calculates trading statistics from trade results
func CalculatePnLStats(trades []TradeResult) PnLStats {
	if len(trades) == 0 {
		return PnLStats{}
	}

	stats := PnLStats{
		TotalTrades: len(trades),
	}

	var totalWins, totalLosses float64
	var wins, losses []float64

	for _, trade := range trades {
		stats.TotalPnL += trade.PnL

		if trade.IsWin {
			stats.WinningTrades++
			totalWins += trade.PnL
			wins = append(wins, trade.PnL)

			if trade.PnL > stats.LargestWin {
				stats.LargestWin = trade.PnL
			}
		} else {
			stats.LosingTrades++
			totalLosses += -trade.PnL // Convert to positive for calculation
			losses = append(losses, -trade.PnL)

			if -trade.PnL > stats.LargestLoss {
				stats.LargestLoss = -trade.PnL
			}
		}
	}

	// Calculate metrics
	if stats.TotalTrades > 0 {
		stats.WinRate = (float64(stats.WinningTrades) / float64(stats.TotalTrades)) * 100
	}

	if len(wins) > 0 {
		stats.AvgWin = totalWins / float64(len(wins))
	}

	if len(losses) > 0 {
		stats.AvgLoss = totalLosses / float64(len(losses))
	}

	if totalLosses > 0 {
		stats.ProfitFactor = totalWins / totalLosses
	}

	if stats.TotalTrades > 0 {
		winProb := float64(stats.WinningTrades) / float64(stats.TotalTrades)
		lossProb := float64(stats.LosingTrades) / float64(stats.TotalTrades)
		stats.ExpectedValue = (winProb * stats.AvgWin) - (lossProb * stats.AvgLoss)
	}

	return stats
}

// CalculateDrawdown calculates drawdown from peak
func CalculateDrawdown(peak, current float64) float64 {
	if peak == 0 {
		return 0
	}
	return ((peak - current) / peak) * 100
}

// CalculateMaxDrawdown finds maximum drawdown from equity curve
func CalculateMaxDrawdown(equity []float64) float64 {
	if len(equity) == 0 {
		return 0
	}

	maxDrawdown := 0.0
	peak := equity[0]

	for _, value := range equity {
		if value > peak {
			peak = value
		}

		drawdown := CalculateDrawdown(peak, value)
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}
