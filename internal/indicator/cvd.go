package indicator

import (
	"mrcrypto-go/internal/model"
)

// CalculateCVD calculates Cumulative Volume Delta
// CVD tracks net buying vs selling pressure over time
// Positive CVD = Accumulation (buyers stronger)
// Negative CVD = Distribution (sellers stronger)
func CalculateCVD(klines []model.Kline) []float64 {
	if len(klines) < 2 {
		return []float64{}
	}

	cvd := make([]float64, len(klines))
	cumulativeDelta := 0.0

	for i, k := range klines {
		// Determine if candle is bullish or bearish
		var delta float64
		if k.Close > k.Open {
			// Bullish candle - buying pressure
			delta = k.Volume
		} else if k.Close < k.Open {
			// Bearish candle - selling pressure
			delta = -k.Volume
		} else {
			// Doji - neutral
			delta = 0
		}

		cumulativeDelta += delta
		cvd[i] = cumulativeDelta
	}

	return cvd
}

// GetLastCVDTrend returns the latest CVD value and its trend
// trend: positive = uptrend (accumulation), negative = downtrend (distribution)
func GetLastCVDTrend(klines []model.Kline, lookback int) (value float64, trend float64) {
	cvd := CalculateCVD(klines)
	if len(cvd) < lookback+1 {
		return 0, 0
	}

	// Latest CVD value
	value = cvd[len(cvd)-1]

	// Calculate trend over lookback period
	// Compare current CVD vs CVD 'lookback' candles ago
	previousCVD := cvd[len(cvd)-lookback-1]
	trend = value - previousCVD

	return value, trend
}

// GetCVDDivergence detects divergence between price and CVD
// Bullish Divergence: Price making lower lows, CVD making higher lows
// Bearish Divergence: Price making higher highs, CVD making lower highs
func GetCVDDivergence(klines []model.Kline, lookback int) string {
	if len(klines) < lookback*2 {
		return ""
	}

	cvd := CalculateCVD(klines)
	closes := make([]float64, len(klines))
	for i, k := range klines {
		closes[i] = k.Close
	}

	// Find recent swing points
	recentPrice := closes[len(closes)-1]
	recentCVD := cvd[len(cvd)-1]

	previousPrice := closes[len(closes)-lookback-1]
	previousCVD := cvd[len(cvd)-lookback-1]

	// Bullish Divergence: Price down, CVD up
	if recentPrice < previousPrice && recentCVD > previousCVD {
		return "Bullish CVD Divergence"
	}

	// Bearish Divergence: Price up, CVD down
	if recentPrice > previousPrice && recentCVD < previousCVD {
		return "Bearish CVD Divergence"
	}

	return ""
}
