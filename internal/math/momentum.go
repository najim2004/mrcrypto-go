package math

// CalculateMomentum calculates price momentum
func CalculateMomentum(closes []float64, period int) float64 {
	if len(closes) < period+1 {
		return 0
	}

	current := closes[len(closes)-1]
	past := closes[len(closes)-1-period]

	return current - past
}

// CalculateROC calculates Rate of Change
func CalculateROC(closes []float64, period int) float64 {
	if len(closes) < period+1 {
		return 0
	}

	current := closes[len(closes)-1]
	past := closes[len(closes)-1-period]

	if past == 0 {
		return 0
	}

	return ((current - past) / past) * 100
}

// CalculateRSIMomentum calculates momentum using RSI
func CalculateRSIMomentum(rsi float64) string {
	if rsi > 70 {
		return "OVERBOUGHT"
	} else if rsi < 30 {
		return "OVERSOLD"
	} else if rsi > 50 {
		return "BULLISH"
	} else if rsi < 50 {
		return "BEARISH"
	}
	return "NEUTRAL"
}

// CalculateStochastic calculates Stochastic Oscillator
func CalculateStochastic(high, low, close []float64, period int) (float64, float64) {
	if len(high) < period || len(low) < period || len(close) < period {
		return 0, 0
	}

	// Get recent period
	recentHigh := high[len(high)-period:]
	recentLow := low[len(low)-period:]
	currentClose := close[len(close)-1]

	// Find highest high and lowest low
	highestHigh := recentHigh[0]
	lowestLow := recentLow[0]

	for i := 1; i < len(recentHigh); i++ {
		if recentHigh[i] > highestHigh {
			highestHigh = recentHigh[i]
		}
		if recentLow[i] < lowestLow {
			lowestLow = recentLow[i]
		}
	}

	// Calculate %K
	var percentK float64
	if highestHigh-lowestLow == 0 {
		percentK = 50
	} else {
		percentK = ((currentClose - lowestLow) / (highestHigh - lowestLow)) * 100
	}

	// %D is simple moving average of %K (simplified to just return %K for now)
	percentD := percentK

	return percentK, percentD
}
