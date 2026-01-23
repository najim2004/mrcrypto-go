package math

import "math"

// =========================================
// PROBABILITY MODULE
// Core probability calculations for trading signals
// =========================================

// CalculateSignalProbability converts confluence score to probability
// Returns a value between 0.0 and 1.0
func CalculateSignalProbability(confluenceScore int) float64 {
	if confluenceScore < 0 {
		return 0.0
	}
	if confluenceScore > 100 {
		return 1.0
	}
	return float64(confluenceScore) / 100.0
}

// CalculateConditionsProbability calculates combined probability of multiple conditions
// Uses product rule: P(A and B) = P(A) * P(B) for independent conditions
func CalculateConditionsProbability(conditions []float64) float64 {
	if len(conditions) == 0 {
		return 0.0
	}

	probability := 1.0
	for _, p := range conditions {
		probability *= p
	}
	return probability
}

// CalculateWeightedProbability calculates weighted average of probabilities
func CalculateWeightedProbability(probabilities []float64, weights []float64) float64 {
	if len(probabilities) == 0 || len(probabilities) != len(weights) {
		return 0.0
	}

	weightedSum := 0.0
	totalWeight := 0.0

	for i, p := range probabilities {
		weightedSum += p * weights[i]
		totalWeight += weights[i]
	}

	if totalWeight == 0 {
		return 0.0
	}

	return weightedSum / totalWeight
}

// CalculateRiskOfRuin calculates the probability of account being wiped out
// Uses fixed-ratio risk of ruin formula
// winRate: probability of winning (0.0-1.0)
// riskRewardRatio: reward/risk ratio
// riskPerTrade: percentage of account risked per trade (e.g., 2.0 for 2%)
func CalculateRiskOfRuin(winRate, riskRewardRatio, riskPerTrade float64) float64 {
	if winRate <= 0 || winRate >= 1 || riskRewardRatio <= 0 || riskPerTrade <= 0 {
		return 1.0 // 100% risk of ruin for invalid inputs
	}

	// Calculate edge
	lossRate := 1 - winRate
	expectedValue := (winRate * riskRewardRatio) - lossRate

	// If no edge, risk of ruin is 100%
	if expectedValue <= 0 {
		return 1.0
	}

	// Risk of Ruin formula: ((1-edge)/(1+edge))^(capital_units)
	// capital_units = 100 / riskPerTrade (how many losses to wipe out)
	edge := expectedValue / (1 + riskRewardRatio)
	capitalUnits := 100 / riskPerTrade

	// RoR = ((1-edge)/(1+edge))^capitalUnits
	base := (1 - edge) / (1 + edge)
	if base <= 0 {
		return 0.0
	}

	ror := math.Pow(base, capitalUnits)

	// Cap at 0-1 range
	if ror < 0 {
		return 0.0
	}
	if ror > 1 {
		return 1.0
	}

	return ror
}

// CalculateOptimalF calculates the optimal fraction to risk using Kelly formula
// This is the fraction that maximizes geometric growth
func CalculateOptimalF(winRate, winLossRatio float64) float64 {
	if winLossRatio <= 0 || winRate <= 0 || winRate >= 1 {
		return 0.0
	}

	// Kelly formula: f* = (bp - q) / b
	// where b = win/loss ratio, p = win probability, q = loss probability
	b := winLossRatio
	p := winRate
	q := 1 - p

	optimalF := (b*p - q) / b

	// Cap between 0 and 0.25 (never risk more than 25%)
	if optimalF < 0 {
		return 0.0
	}
	if optimalF > 0.25 {
		return 0.25
	}

	return optimalF
}

// CalculateSharpeRatio calculates the risk-adjusted return
// Higher Sharpe ratio = better risk-adjusted performance
// returns: array of period returns (as decimals, e.g., 0.05 for 5%)
// riskFreeRate: annual risk-free rate (e.g., 0.04 for 4%)
func CalculateSharpeRatio(returns []float64, riskFreeRate float64) float64 {
	if len(returns) < 2 {
		return 0.0
	}

	// Calculate mean return
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	// Calculate standard deviation
	variance := 0.0
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(returns))
	stdDev := math.Sqrt(variance)

	if stdDev == 0 {
		return 0.0
	}

	// Annualize (assuming daily returns, 365 trading days for crypto)
	annualizedReturn := mean * 365
	annualizedStdDev := stdDev * math.Sqrt(365)

	sharpe := (annualizedReturn - riskFreeRate) / annualizedStdDev

	return sharpe
}

// CalculateSortinoRatio calculates downside risk-adjusted return
// Only considers negative returns for volatility calculation
func CalculateSortinoRatio(returns []float64, riskFreeRate float64) float64 {
	if len(returns) < 2 {
		return 0.0
	}

	// Calculate mean return
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	// Calculate downside deviation (only negative returns)
	downsideVariance := 0.0
	downsideCount := 0
	for _, r := range returns {
		if r < 0 {
			downsideVariance += r * r
			downsideCount++
		}
	}

	if downsideCount == 0 {
		return 999.0 // No downside = infinite Sortino
	}

	downsideVariance /= float64(downsideCount)
	downsideDeviation := math.Sqrt(downsideVariance)

	if downsideDeviation == 0 {
		return 999.0
	}

	// Annualize
	annualizedReturn := mean * 365
	annualizedDownside := downsideDeviation * math.Sqrt(365)

	sortino := (annualizedReturn - riskFreeRate) / annualizedDownside

	return sortino
}

// CalculateBayesianUpdate updates probability based on new evidence
// priorProbability: initial probability P(H)
// likelihoodGivenTrue: P(E|H) - probability of evidence given hypothesis is true
// likelihoodGivenFalse: P(E|not H) - probability of evidence given hypothesis is false
// Returns: P(H|E) - updated probability
func CalculateBayesianUpdate(priorProbability, likelihoodGivenTrue, likelihoodGivenFalse float64) float64 {
	if likelihoodGivenTrue < 0 || likelihoodGivenFalse < 0 {
		return priorProbability
	}

	// P(E) = P(E|H)*P(H) + P(E|not H)*P(not H)
	pEvidence := likelihoodGivenTrue*priorProbability + likelihoodGivenFalse*(1-priorProbability)

	if pEvidence == 0 {
		return priorProbability
	}

	// Bayes' theorem: P(H|E) = P(E|H) * P(H) / P(E)
	posteriorProbability := (likelihoodGivenTrue * priorProbability) / pEvidence

	// Cap at 0-1 range
	if posteriorProbability < 0 {
		return 0.0
	}
	if posteriorProbability > 1 {
		return 1.0
	}

	return posteriorProbability
}

// CalculateBreakEvenWinRate calculates required win rate to break even
// given a risk:reward ratio
func CalculateBreakEvenWinRate(riskRewardRatio float64) float64 {
	if riskRewardRatio <= 0 {
		return 1.0 // Need 100% win rate
	}

	// Break-even: WinRate * Reward = LossRate * Risk
	// WinRate * RR = (1 - WinRate) * 1
	// WinRate * RR = 1 - WinRate
	// WinRate * (RR + 1) = 1
	// WinRate = 1 / (RR + 1)
	breakEven := 1.0 / (riskRewardRatio + 1.0)

	return breakEven * 100 // Return as percentage
}

// CalculateExpectedValueFromProbability calculates expected value
// probability: win probability (0.0-1.0)
// avgWin: average winning amount
// avgLoss: average losing amount (as positive number)
func CalculateExpectedValueFromProbability(probability, avgWin, avgLoss float64) float64 {
	return (probability * avgWin) - ((1 - probability) * avgLoss)
}

// CalculateConfidenceInterval calculates confidence interval for win rate
// based on sample size using Wilson score interval
// wins: number of winning trades
// total: total number of trades
// z: z-score for desired confidence level (1.96 for 95%)
func CalculateConfidenceInterval(wins, total int, z float64) (lower, upper float64) {
	if total == 0 {
		return 0, 0
	}

	n := float64(total)
	p := float64(wins) / n

	denominator := 1 + (z*z)/n

	center := p + (z*z)/(2*n)
	spread := z * math.Sqrt((p*(1-p)+(z*z)/(4*n))/n)

	lower = (center - spread) / denominator
	upper = (center + spread) / denominator

	// Clamp to 0-1
	if lower < 0 {
		lower = 0
	}
	if upper > 1 {
		upper = 1
	}

	return lower, upper
}

// IsPositiveExpectancy checks if a strategy has positive expected value
func IsPositiveExpectancy(winRate, avgWin, avgLoss float64) bool {
	ev := CalculateExpectedValueFromProbability(winRate, avgWin, avgLoss)
	return ev > 0
}
