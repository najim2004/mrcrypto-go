package indicator

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

// CalculateSMA calculates the Simple Moving Average
func CalculateSMA(closes []float64, period int) []float64 {
	if len(closes) < period {
		return []float64{}
	}

	sma := make([]float64, len(closes))
	for i := period - 1; i < len(closes); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += closes[j]
		}
		sma[i] = sum / float64(period)
	}
	return sma
}
