package indicator

import (
	"math"
	"mrcrypto-go/internal/model"
)

// CalculateATR calculates the Average True Range
func CalculateATR(klines []model.Kline, period int) float64 {
	if len(klines) < period+1 {
		return 0
	}

	trValues := make([]float64, len(klines))

	// Calculate TR (True Range) for each candle
	for i := 1; i < len(klines); i++ {
		high := klines[i].High
		low := klines[i].Low
		prevClose := klines[i-1].Close

		tr1 := high - low
		tr2 := math.Abs(high - prevClose)
		tr3 := math.Abs(low - prevClose)

		trValues[i] = math.Max(tr1, math.Max(tr2, tr3))
	}

	// First ATR is simple average of TR for the period
	sumTR := 0.0
	for i := 1; i <= period; i++ {
		sumTR += trValues[i]
	}
	currentATR := sumTR / float64(period)

	// Subsequent ATRs using smoothing: ATR = ((Prior ATR * (n-1)) + Current TR) / n
	// Start from period + 1
	for i := period + 1; i < len(klines); i++ {
		currentATR = ((currentATR * float64(period-1)) + trValues[i]) / float64(period)
	}

	return currentATR
}
