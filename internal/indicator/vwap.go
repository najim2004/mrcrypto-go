package indicator

// CalculateVWAP calculates the Volume Weighted Average Price
func CalculateVWAP(high, low, close, volume []float64) []float64 {
	if len(high) != len(low) || len(low) != len(close) || len(close) != len(volume) {
		return []float64{}
	}

	vwap := make([]float64, len(close))
	cumulativeTPV := 0.0 // Cumulative Typical Price * Volume
	cumulativeVolume := 0.0

	for i := 0; i < len(close); i++ {
		typicalPrice := (high[i] + low[i] + close[i]) / 3.0
		tpv := typicalPrice * volume[i]

		cumulativeTPV += tpv
		cumulativeVolume += volume[i]

		if cumulativeVolume != 0 {
			vwap[i] = cumulativeTPV / cumulativeVolume
		}
	}

	return vwap
}

// GetLastVWAP returns the most recent VWAP value
func GetLastVWAP(high, low, close, volume []float64) float64 {
	vwap := CalculateVWAP(high, low, close, volume)
	if len(vwap) == 0 {
		return 0
	}
	return vwap[len(vwap)-1]
}
