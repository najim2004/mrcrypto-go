package math

import "math"

// CalculateATR calculates Average True Range
func CalculateATR(high, low, close []float64, period int) float64 {
	if len(high) < period+1 {
		return 0
	}

	trueRanges := make([]float64, len(high)-1)

	for i := 1; i < len(high); i++ {
		hl := high[i] - low[i]
		hc := math.Abs(high[i] - close[i-1])
		lc := math.Abs(low[i] - close[i-1])

		trueRanges[i-1] = math.Max(hl, math.Max(hc, lc))
	}

	// Calculate average of last 'period' true ranges
	sum := 0.0
	for i := len(trueRanges) - period; i < len(trueRanges); i++ {
		sum += trueRanges[i]
	}

	return sum / float64(period)
}

// CalculateVolatility calculates volatility percentage
func CalculateVolatility(close []float64, period int) float64 {
	if len(close) < period {
		return 0
	}

	// Calculate returns
	returns := make([]float64, len(close)-1)
	for i := 1; i < len(close); i++ {
		returns[i-1] = (close[i] - close[i-1]) / close[i-1]
	}

	// Calculate standard deviation of last 'period' returns
	recentReturns := returns[len(returns)-period:]

	mean := 0.0
	for _, r := range recentReturns {
		mean += r
	}
	mean /= float64(len(recentReturns))

	variance := 0.0
	for _, r := range recentReturns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(recentReturns))

	stdDev := math.Sqrt(variance)

	// Annualize volatility (252 trading days, 24 hours)
	annualizedVolatility := stdDev * math.Sqrt(365*24) * 100

	return annualizedVolatility
}

// CalculateStandardDeviation calculates standard deviation of price
func CalculateStandardDeviation(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))

	return math.Sqrt(variance)
}

// CalculateBeta calculates beta (volatility relative to market)
func CalculateBeta(assetReturns, marketReturns []float64) float64 {
	if len(assetReturns) != len(marketReturns) || len(assetReturns) == 0 {
		return 1.0 // Default beta
	}

	// Calculate covariance
	assetMean := 0.0
	marketMean := 0.0
	for i := 0; i < len(assetReturns); i++ {
		assetMean += assetReturns[i]
		marketMean += marketReturns[i]
	}
	assetMean /= float64(len(assetReturns))
	marketMean /= float64(len(marketReturns))

	covariance := 0.0
	marketVariance := 0.0
	for i := 0; i < len(assetReturns); i++ {
		assetDiff := assetReturns[i] - assetMean
		marketDiff := marketReturns[i] - marketMean
		covariance += assetDiff * marketDiff
		marketVariance += marketDiff * marketDiff
	}

	if marketVariance == 0 {
		return 1.0
	}

	return covariance / marketVariance
}
