package indicator

// DetectDivergence checks for Regular Bearish/Bullish Divergence
// Returns "Bullish", "Bearish", or ""
// prices: Highs for Bearish, Lows for Bullish
// indicators: RSI or MACD values corresponding to prices
// lookback: Number of candles to check for peaks (e.g., 20)
func DetectDivergence(prices []float64, indicators []float64, lookback int) string {
	if len(prices) < lookback || len(indicators) < lookback {
		return ""
	}

	// We need at least 2 peaks/troughs to compare
	// This is a simplified logic: checks current peak vs previous peak in the window

	lenData := len(prices)
	currentPrice := prices[lenData-1]
	currentInd := indicators[lenData-1]

	// 1. Check for BEARISH Divergence (Price Higher High, Indicator Lower High)
	// We assume we are at a local high if current price is higher than recent neighbors
	// Simplified: Check against the highest point in the first half of the lookback window
	maxPricePrev := -99999.0
	maxIndPrev := -99999.0
	foundPeak := false

	// Scan previous window (excluding last 3 candles to find distinct previous peak)
	for i := lenData - lookback; i < lenData-5; i++ {
		if prices[i] > maxPricePrev {
			maxPricePrev = prices[i]
			maxIndPrev = indicators[i]
			foundPeak = true
		}
	}

	if foundPeak {
		// Bearish: Price made Higher High, Indicator made Lower High
		if currentPrice > maxPricePrev && currentInd < maxIndPrev {
			// Filter: Indicator should be somewhat high (e.g. RSI > 50) to matter
			if currentInd > 50 {
				return "Bearish"
			}
		}
	}

	// 2. Check for BULLISH Divergence (Price Lower Low, Indicator Higher Low)
	minPricePrev := 99999.0
	minIndPrev := 99999.0
	foundTrough := false

	for i := lenData - lookback; i < lenData-5; i++ {
		if prices[i] < minPricePrev {
			minPricePrev = prices[i]
			minIndPrev = indicators[i]
			foundTrough = true
		}
	}

	if foundTrough {
		// Bullish: Price made Lower Low, Indicator made Higher Low
		if currentPrice < minPricePrev && currentInd > minIndPrev {
			// Filter: Indicator should be somewhat low (e.g. RSI < 50)
			if currentInd < 50 {
				return "Bullish"
			}
		}
	}

	return ""
}
