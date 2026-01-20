package indicator

// CalculateMACD calculates the Moving Average Convergence Divergence
func CalculateMACD(closes []float64, fastPeriod, slowPeriod, signalPeriod int) (macd, signal, histogram []float64) {
	if len(closes) < slowPeriod {
		return []float64{}, []float64{}, []float64{}
	}

	// Calculate fast and slow EMAs
	fastEMA := CalculateEMA(closes, fastPeriod)
	slowEMA := CalculateEMA(closes, slowPeriod)

	// Calculate MACD line
	macdLine := make([]float64, len(closes))
	for i := slowPeriod - 1; i < len(closes); i++ {
		macdLine[i] = fastEMA[i] - slowEMA[i]
	}

	// Calculate signal line (EMA of MACD)
	signalLine := CalculateEMA(macdLine[slowPeriod-1:], signalPeriod)

	// Calculate histogram
	histogramLine := make([]float64, len(signalLine))
	for i := 0; i < len(signalLine); i++ {
		histogramLine[i] = macdLine[slowPeriod-1+i] - signalLine[i]
	}

	return macdLine, signalLine, histogramLine
}

// GetLastMACD returns the most recent MACD, signal, and histogram values
func GetLastMACD(closes []float64, fastPeriod, slowPeriod, signalPeriod int) (macd, signal, histogram float64) {
	macdLine, signalLine, histogramLine := CalculateMACD(closes, fastPeriod, slowPeriod, signalPeriod)

	if len(macdLine) == 0 || len(signalLine) == 0 || len(histogramLine) == 0 {
		return 0, 0, 0
	}

	macd = macdLine[len(macdLine)-1]
	signal = signalLine[len(signalLine)-1]
	histogram = histogramLine[len(histogramLine)-1]

	return macd, signal, histogram
}
