package indicator

import "math"

// CalculateBollingerBands calculates Bollinger Bands
func CalculateBollingerBands(closes []float64, period int, stdDev float64) (upper, middle, lower []float64) {
	if len(closes) < period {
		return []float64{}, []float64{}, []float64{}
	}

	upper = make([]float64, len(closes))
	middle = make([]float64, len(closes))
	lower = make([]float64, len(closes))

	for i := period - 1; i < len(closes); i++ {
		// Calculate SMA (middle band)
		var sum float64
		for j := i - period + 1; j <= i; j++ {
			sum += closes[j]
		}
		sma := sum / float64(period)
		middle[i] = sma

		// Calculate standard deviation
		var variance float64
		for j := i - period + 1; j <= i; j++ {
			variance += math.Pow(closes[j]-sma, 2)
		}
		std := math.Sqrt(variance / float64(period))

		// Calculate upper and lower bands
		upper[i] = sma + (stdDev * std)
		lower[i] = sma - (stdDev * std)
	}

	return upper, middle, lower
}

// GetLastBollingerBands returns the most recent Bollinger Bands values
func GetLastBollingerBands(closes []float64, period int, stdDev float64) (upper, middle, lower float64) {
	upperBand, middleBand, lowerBand := CalculateBollingerBands(closes, period, stdDev)

	if len(upperBand) == 0 {
		return 0, 0, 0
	}

	upper = upperBand[len(upperBand)-1]
	middle = middleBand[len(middleBand)-1]
	lower = lowerBand[len(lowerBand)-1]

	return upper, middle, lower
}
