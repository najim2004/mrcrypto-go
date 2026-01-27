package indicator

// CalculateStochRSI calculates Stochastic RSI (K and D)
// rsiValues: Input RSI array
// period: Length of StochRSI (usually 14)
// smoothK: Smoothing for %K (usually 3)
// smoothD: Smoothing for %D (usually 3)
func CalculateStochRSI(rsiValues []float64, period, smoothK, smoothD int) ([]float64, []float64) {
	if len(rsiValues) < period {
		return []float64{}, []float64{}
	}

	stochRSI := make([]float64, len(rsiValues))

	// Calculate raw StochRSI
	for i := period - 1; i < len(rsiValues); i++ {
		// Find Min and Max RSI in period
		minRSI := 100.0
		maxRSI := 0.0
		for j := i - period + 1; j <= i; j++ {
			val := rsiValues[j]
			if val < minRSI {
				minRSI = val
			}
			if val > maxRSI {
				maxRSI = val
			}
		}

		if maxRSI-minRSI == 0 {
			stochRSI[i] = 100 // Edge case: flat RSI
		} else {
			stochRSI[i] = ((rsiValues[i] - minRSI) / (maxRSI - minRSI)) * 100
		}
	}

	// Calculate %K (SMA of StochRSI)
	kLine := CalculateSMA(stochRSI, smoothK)

	// Calculate %D (SMA of %K)
	dLine := CalculateSMA(kLine, smoothD)

	return kLine, dLine
}

// GetLastStochRSI returns the most recent K and D values
func GetLastStochRSI(rsiValues []float64, period, smoothK, smoothD int) (float64, float64) {
	k, d := CalculateStochRSI(rsiValues, period, smoothK, smoothD)
	if len(k) == 0 || len(d) == 0 {
		return 0, 0
	}
	return k[len(k)-1], d[len(d)-1]
}
