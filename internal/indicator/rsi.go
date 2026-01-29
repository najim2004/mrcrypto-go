package indicator

import "math"

// CalculateRSI calculates the Relative Strength Index
func CalculateRSI(closes []float64, period int) []float64 {
	if len(closes) < period+1 {
		return []float64{}
	}

	const epsilon = 1e-10 // Threshold for near-zero values

	rsi := make([]float64, len(closes))
	gains := make([]float64, len(closes)-1)
	losses := make([]float64, len(closes)-1)

	// Calculate gains and losses
	for i := 1; i < len(closes); i++ {
		change := closes[i] - closes[i-1]
		if change > 0 {
			gains[i-1] = change
			losses[i-1] = 0
		} else {
			gains[i-1] = 0
			losses[i-1] = math.Abs(change)
		}
	}

	// Calculate first average gain and loss
	var avgGain, avgLoss float64
	for i := 0; i < period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	// Calculate RSI for first period
	if avgLoss < epsilon {
		rsi[period] = 100
	} else {
		rs := avgGain / avgLoss
		rsi[period] = 100 - (100 / (1 + rs))
	}

	// Clamp to valid range
	rsi[period] = math.Max(0, math.Min(100, rsi[period]))

	// Calculate subsequent RSI values using smoothed averages
	for i := period; i < len(gains); i++ {
		avgGain = (avgGain*float64(period-1) + gains[i]) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + losses[i]) / float64(period)

		if avgLoss < epsilon {
			rsi[i+1] = 100
		} else {
			rs := avgGain / avgLoss
			rsi[i+1] = 100 - (100 / (1 + rs))
		}

		// Clamp to valid range
		rsi[i+1] = math.Max(0, math.Min(100, rsi[i+1]))
	}

	return rsi
}

// GetLastRSI returns the most recent RSI value
func GetLastRSI(closes []float64, period int) float64 {
	rsi := CalculateRSI(closes, period)
	if len(rsi) == 0 {
		return 0
	}
	return rsi[len(rsi)-1]
}
