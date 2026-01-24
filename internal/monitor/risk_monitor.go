package monitor

import (
	"fmt"
	"log"
	"my-tool-go/internal/model"
)

type RiskMonitor struct {
	maxRiskPercentage float64
	maxOpenTrades     int
	accountBalance    float64
}

func NewRiskMonitor(accountBalance, maxRiskPercentage float64, maxOpenTrades int) *RiskMonitor {
	return &RiskMonitor{
		maxRiskPercentage: maxRiskPercentage,
		maxOpenTrades:     maxOpenTrades,
		accountBalance:    accountBalance,
	}
}

// CalculatePortfolioRisk calculates total risk exposure
func (rm *RiskMonitor) CalculatePortfolioRisk(activeSignals []*model.Signal) float64 {
	totalRisk := 0.0

	for _, signal := range activeSignals {
		riskPerTrade := calculateSignalRisk(signal)
		totalRisk += riskPerTrade
	}

	return (totalRisk / rm.accountBalance) * 100
}

// CheckRiskLimits checks if adding a new signal exceeds risk limits
func (rm *RiskMonitor) CheckRiskLimits(activeSignals []*model.Signal, newSignal *model.Signal) (bool, string) {
	// Check max open trades
	if len(activeSignals) >= rm.maxOpenTrades {
		return false, "Max open trades limit reached"
	}

	// Calculate current risk
	currentRisk := rm.CalculatePortfolioRisk(activeSignals)

	// Calculate new signal risk
	newRisk := calculateSignalRisk(newSignal)
	newRiskPercentage := (newRisk / rm.accountBalance) * 100

	// Check if total risk would exceed limit
	totalRisk := currentRisk + newRiskPercentage
	if totalRisk > rm.maxRiskPercentage {
		log.Printf("⚠️  Risk limit exceeded: %.2f%% (max: %.2f%%)", totalRisk, rm.maxRiskPercentage)
		return false, "Portfolio risk limit exceeded"
	}

	return true, "Risk within limits"
}

func calculateSignalRisk(signal *model.Signal) float64 {
	// Risk is the distance between entry and stop loss
	if signal.EntryPrice > signal.StopLoss {
		return signal.EntryPrice - signal.StopLoss
	}
	return signal.StopLoss - signal.EntryPrice
}

// GetRiskSummary returns a risk summary
func (rm *RiskMonitor) GetRiskSummary(activeSignals []*model.Signal) string {
	totalRisk := rm.CalculatePortfolioRisk(activeSignals)

	summary := "📊 Risk Monitor Summary\n"
	summary += "------------------------\n"
	summary += fmt.Sprintf("Active Trades: %d/%d\n", len(activeSignals), rm.maxOpenTrades)
	summary += fmt.Sprintf("Total Risk: %.2f%%\n", totalRisk)
	summary += fmt.Sprintf("Risk Limit: %.2f%%\n", rm.maxRiskPercentage)
	summary += fmt.Sprintf("Available Risk: %.2f%%", rm.maxRiskPercentage-totalRisk)

	return summary
}
