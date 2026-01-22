package indicator

import (
	"math"
	internalmath "my-tool-go/internal/math"
)

// CalculateEMA calculates the Exponential Moving Average
func CalculateEMA(closes []float64, period int) []float64 {
	if len(closes) < period {
		return []float64{}
	}

	ema := make([]float64, len(closes))
	multiplier := 2.0 / float64(period+1)

	// First EMA is SMA
	var sum float64
	for i := 0; i < period; i++ {
		sum += closes[i]
	}
	ema[period-1] = sum / float64(period)

	// Calculate subsequent EMAs
	for i := period; i < len(closes); i++ {
		ema[i] = (closes[i]-ema[i-1])*multiplier + ema[i-1]
	}

	return ema
}

func CalculateTrueRange(high, low, close []float64) []float64 {
	return internalmath.CalculateTrueRange(high, low, close)
}

// CalculateADX calculates the Average Directional Index
func CalculateADX(high, low, close []float64, period int) []float64 {
	if len(high) < period+1 || len(low) < period+1 || len(close) < period+1 {
		return []float64{}
	}

	// Calculate +DM and -DM
	plusDM := make([]float64, len(high)-1)
	minusDM := make([]float64, len(high)-1)

	for i := 1; i < len(high); i++ {
		highDiff := high[i] - high[i-1]
		lowDiff := low[i-1] - low[i]

		if highDiff > lowDiff && highDiff > 0 {
			plusDM[i-1] = highDiff
		} else {
			plusDM[i-1] = 0
		}

		if lowDiff > highDiff && lowDiff > 0 {
			minusDM[i-1] = lowDiff
		} else {
			minusDM[i-1] = 0
		}
	}

	// Calculate True Range
	tr := CalculateTrueRange(high, low, close)

	// Calculate smoothed +DM, -DM, and TR
	smoothedPlusDM := smoothArray(plusDM, period)
	smoothedMinusDM := smoothArray(minusDM, period)
	smoothedTR := smoothArray(tr[1:], period)

	// Calculate +DI and -DI
	plusDI := make([]float64, len(smoothedTR))
	minusDI := make([]float64, len(smoothedTR))

	for i := 0; i < len(smoothedTR); i++ {
		if smoothedTR[i] != 0 {
			plusDI[i] = (smoothedPlusDM[i] / smoothedTR[i]) * 100
			minusDI[i] = (smoothedMinusDM[i] / smoothedTR[i]) * 100
		}
	}

	// Calculate DX
	dx := make([]float64, len(plusDI))
	for i := 0; i < len(plusDI); i++ {
		diSum := plusDI[i] + minusDI[i]
		if diSum != 0 {
			dx[i] = (math.Abs(plusDI[i]-minusDI[i]) / diSum) * 100
		}
	}

	// Calculate ADX (smoothed DX)
	adx := smoothArray(dx, period)

	return adx
}

// smoothArray applies Wilder's smoothing
func smoothArray(values []float64, period int) []float64 {
	if len(values) < period {
		return []float64{}
	}

	smoothed := make([]float64, len(values)-period+1)

	// First value is average
	var sum float64
	for i := 0; i < period; i++ {
		sum += values[i]
	}
	smoothed[0] = sum / float64(period)

	// Apply Wilder's smoothing
	for i := 1; i < len(smoothed); i++ {
		smoothed[i] = (smoothed[i-1]*float64(period-1) + values[i+period-1]) / float64(period)
	}

	return smoothed
}

// GetLastADX returns the most recent ADX value
func GetLastADX(high, low, close []float64, period int) float64 {
	adx := CalculateADX(high, low, close, period)
	if len(adx) == 0 {
		return 0
	}
	return adx[len(adx)-1]
}
