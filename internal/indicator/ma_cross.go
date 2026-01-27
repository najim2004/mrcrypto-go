package indicator

// TrendState represents the current trend state based on MA cross
type TrendState string

const (
	GoldenCross TrendState = "Golden Cross" // Bullish (Fast > Slow)
	DeathCross  TrendState = "Death Cross"  // Bearish (Fast < Slow)
	Neutral     TrendState = "Neutral"
)

// CheckTrendState determines the market trend using EMA crossovers
// typically fastPeriod=50, slowPeriod=200
func CheckTrendState(closes []float64, fastPeriod, slowPeriod int) (TrendState, float64, float64) {
	if len(closes) < slowPeriod {
		return Neutral, 0, 0
	}

	fastEMA := CalculateEMA(closes, fastPeriod)
	slowEMA := CalculateEMA(closes, slowPeriod)

	if len(fastEMA) == 0 || len(slowEMA) == 0 {
		return Neutral, 0, 0
	}

	lastFast := fastEMA[len(fastEMA)-1]
	lastSlow := slowEMA[len(slowEMA)-1]

	if lastFast > lastSlow {
		return GoldenCross, lastFast, lastSlow
	} else if lastFast < lastSlow {
		return DeathCross, lastFast, lastSlow
	}

	return Neutral, lastFast, lastSlow
}
