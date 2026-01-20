package monitor

import (
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
	summary += "Active Trades: " + string(rune(len(activeSignals))) + "/" + string(rune(rm.maxOpenTrades)) + "\n"
	summary += "Total Risk: " + string(rune(int(totalRisk))) + "%\n"
	summary += "Risk Limit: " + string(rune(int(rm.maxRiskPercentage))) + "%\n"
	summary += "Available Risk: " + string(rune(int(rm.maxRiskPercentage-totalRisk))) + "%"

	return summary
}
