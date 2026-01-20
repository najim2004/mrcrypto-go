package math

import "math"

// CalculatePositionSize calculates position size based on risk percentage
func CalculatePositionSize(accountBalance, riskPercentage, entryPrice, stopLoss float64) float64 {
	riskAmount := accountBalance * (riskPercentage / 100.0)
	priceRisk := math.Abs(entryPrice - stopLoss)

	if priceRisk == 0 {
		return 0
	}

	return riskAmount / priceRisk
}

// CalculateKellyCriterion calculates optimal position size using Kelly Criterion
func CalculateKellyCriterion(winRate, avgWin, avgLoss float64) float64 {
	if avgLoss == 0 {
		return 0
	}

	winLossRatio := avgWin / avgLoss
	kelly := (winRate - ((1 - winRate) / winLossRatio)) * 100

	// Conservative Kelly (use half Kelly to be safe)
	conservativeKelly := kelly / 2.0

	// Cap at 25% max
	if conservativeKelly > 25 {
		return 25
	}
	if conservativeKelly < 0 {
		return 0
	}

	return conservativeKelly
}

// CalculateRiskAmount calculates dollar risk amount
func CalculateRiskAmount(positionSize, entryPrice, stopLoss float64) float64 {
	return positionSize * math.Abs(entryPrice-stopLoss)
}

// CalculateLeverage calculates leverage used
func CalculateLeverage(positionValue, accountBalance float64) float64 {
	if accountBalance == 0 {
		return 0
	}
	return positionValue / accountBalance
}

// CalculateMarginRequired calculates margin needed for leveraged position
func CalculateMarginRequired(positionValue, leverage float64) float64 {
	if leverage == 0 {
		return positionValue
	}
	return positionValue / leverage
}
