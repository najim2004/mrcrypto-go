package math

import "math"

// RiskRewardRatio represents risk:reward ratio details
type RiskRewardRatio struct {
	Ratio            float64
	RiskAmount       float64
	RewardAmount     float64
	BreakEvenWinRate float64
}

// CalculateRiskReward calculates risk:reward ratio
func CalculateRiskReward(entryPrice, stopLoss, takeProfit float64) RiskRewardRatio {
	risk := math.Abs(entryPrice - stopLoss)
	reward := math.Abs(takeProfit - entryPrice)

	var ratio float64
	if risk == 0 {
		ratio = 0
	} else {
		ratio = reward / risk
	}

	// Break-even win rate = Risk / (Risk + Reward)
	breakEvenWinRate := 0.0
	if risk+reward > 0 {
		breakEvenWinRate = (risk / (risk + reward)) * 100
	}

	return RiskRewardRatio{
		Ratio:            ratio,
		RiskAmount:       risk,
		RewardAmount:     reward,
		BreakEvenWinRate: breakEvenWinRate,
	}
}

// CalculateExpectedValue calculates expected value of a trade
func CalculateExpectedValue(winRate, avgWin, avgLoss float64) float64 {
	lossRate := 1 - winRate
	return (winRate * avgWin) - (lossRate * avgLoss)
}

// CalculateProfitFactor calculates profit factor
func CalculateProfitFactor(totalWins, totalLosses float64) float64 {
	if totalLosses == 0 {
		if totalWins > 0 {
			return 999.0 // Infinite profit factor
		}
		return 0
	}
	return totalWins / totalLosses
}

// CalculateWinRate calculates win rate percentage
func CalculateWinRate(wins, losses int) float64 {
	total := wins + losses
	if total == 0 {
		return 0
	}
	return (float64(wins) / float64(total)) * 100
}

// CalculateAverageWin calculates average winning trade
func CalculateAverageWin(winningTrades []float64) float64 {
	if len(winningTrades) == 0 {
		return 0
	}

	sum := 0.0
	for _, win := range winningTrades {
		sum += win
	}
	return sum / float64(len(winningTrades))
}

// CalculateAverageLoss calculates average losing trade
func CalculateAverageLoss(losingTrades []float64) float64 {
	if len(losingTrades) == 0 {
		return 0
	}

	sum := 0.0
	for _, loss := range losingTrades {
		sum += math.Abs(loss)
	}
	return sum / float64(len(losingTrades))
}

// IsGoodRiskReward checks if risk:reward ratio is acceptable
func IsGoodRiskReward(ratio float64, minimumRatio float64) bool {
	return ratio >= minimumRatio
}
