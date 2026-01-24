package indicator

import (
	"math"
	"mrcrypto-go/internal/model"
)

// FVG represents a Fair Value Gap
type FVG struct {
	Top       float64
	Bottom    float64
	Type      string // "BULLISH" or "BEARISH"
	CreatedAt int64
}

// OrderBlock represents an institutional order block
type OrderBlock struct {
	Top       float64
	Bottom    float64
	Type      string // "BULLISH" or "BEARISH"
	CreatedAt int64
}

// FindFVGs identifies Fair Value Gaps in the provided klines
// FVG occurs when the 1st candle's wick doesn't overlap with 3rd candle's wick
func FindFVGs(klines []model.Kline) []FVG {
	var fvgs []FVG
	if len(klines) < 3 {
		return fvgs
	}

	for i := len(klines) - 2; i >= 1; i-- { // Iterate backwards
		current := klines[i]
		prev := klines[i-1]
		next := klines[i+1]

		// Bullish FVG
		// Caused by a large green candle (current)
		// Gap is between prev.High and next.Low
		if current.Close > current.Open && next.Low > prev.High {
			gapSize := next.Low - prev.High
			// Minimal filter: gap should be somewhat significant
			if gapSize > (current.High-current.Low)*0.1 {
				fvgs = append(fvgs, FVG{
					Top:       next.Low,
					Bottom:    prev.High,
					Type:      "BULLISH",
					CreatedAt: current.CloseTime,
				})
			}
		}

		// Bearish FVG
		// Caused by a large red candle (current)
		// Gap is between prev.Low and next.High
		if current.Close < current.Open && next.High < prev.Low {
			gapSize := prev.Low - next.High
			if gapSize > (current.High-current.Low)*0.1 {
				fvgs = append(fvgs, FVG{
					Top:       prev.Low,
					Bottom:    next.High,
					Type:      "BEARISH",
					CreatedAt: current.CloseTime,
				})
			}
		}

		// Limit to last 5 FVGs to keep it snappy
		if len(fvgs) >= 5 {
			break
		}
	}
	return fvgs
}

// FindOrderBlocks identifies potential Order Blocks
// A basic OB definition: Last opposing candle before a displacement move
func FindOrderBlocks(klines []model.Kline) []OrderBlock {
	var obs []OrderBlock
	if len(klines) < 5 {
		return obs
	}

	// Simple lookback
	for i := len(klines) - 4; i >= 1; i-- {
		// Bullish OB: Last Red candle before a strong Green move that breaks structure/high
		if klines[i].Close < klines[i].Open { // Red candle
			// Check if subsequent candles pushed higher significantly
			moveUp := false
			highestAfter := 0.0
			for j := 1; j <= 3; j++ {
				if i+j < len(klines) {
					if klines[i+j].Close > klines[i+j].Open && klines[i+j].Close > klines[i].High {
						highestAfter = math.Max(highestAfter, klines[i+j].Close)
					}
				}
			}

			// Displacement check: move greater than OB size * 2
			obSize := klines[i].High - klines[i].Low
			if highestAfter > klines[i].High+(obSize*2) {
				moveUp = true
			}

			if moveUp {
				obs = append(obs, OrderBlock{
					Top:       klines[i].High,
					Bottom:    klines[i].Low,
					Type:      "BULLISH",
					CreatedAt: klines[i].CloseTime,
				})
			}
		}

		// Bearish OB: Last Green candle before a strong Red move
		if klines[i].Close > klines[i].Open { // Green candle
			moveDown := false
			lowestAfter := math.MaxFloat64
			for j := 1; j <= 3; j++ {
				if i+j < len(klines) {
					if klines[i+j].Close < klines[i+j].Open && klines[i+j].Close < klines[i].Low {
						lowestAfter = math.Min(lowestAfter, klines[i+j].Close)
					}
				}
			}

			obSize := klines[i].High - klines[i].Low
			if lowestAfter < klines[i].Low-(obSize*2) {
				moveDown = true
			}

			if moveDown {
				obs = append(obs, OrderBlock{
					Top:       klines[i].High,
					Bottom:    klines[i].Low,
					Type:      "BEARISH",
					CreatedAt: klines[i].CloseTime,
				})
			}
		}

		if len(obs) >= 5 {
			break
		}
	}
	return obs
}

// IsPriceInFVG checks if current price is inside any active FVG
func IsPriceInFVG(price float64, fvgs []FVG) (bool, string) {
	for _, fvg := range fvgs {
		if price >= fvg.Bottom && price <= fvg.Top {
			return true, fvg.Type
		}
	}
	return false, ""
}

// IsPriceInOB checks if current price is inside any active Order Block
func IsPriceInOB(price float64, obs []OrderBlock) (bool, string) {
	for _, ob := range obs {
		// Check if we are re-testing the OB
		if price >= ob.Bottom && price <= ob.Top {
			return true, ob.Type
		}
	}
	return false, ""
}
