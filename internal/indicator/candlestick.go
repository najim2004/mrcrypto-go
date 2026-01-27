package indicator

import (
	"math"
	"mrcrypto-go/internal/model"
)

// IdentifyPattern analyzes the last candle(s) to find specific patterns
func IdentifyPattern(klines []model.Kline) string {
	if len(klines) < 3 {
		return ""
	}

	last := klines[len(klines)-1]
	prev := klines[len(klines)-2]
	prev2 := klines[len(klines)-3]

	// Basic properties
	bodySize := math.Abs(last.Close - last.Open)
	upperWick := last.High - math.Max(last.Open, last.Close)
	lowerWick := math.Min(last.Open, last.Close) - last.Low
	totalRange := last.High - last.Low

	isBullish := last.Close > last.Open
	isBearish := last.Close < last.Open
	prevBearish := prev.Close < prev.Open
	prevBullish := prev.Close > prev.Open

	// 1. HAMMER / SHOOTING STAR (Single Candle)
	// Hammer: Small body at top, long lower wick (at least 2x body)
	if lowerWick > bodySize*2 && upperWick < bodySize {
		if isBearish {
			return "Hammer (Weak)"
		}
		return "Hammer"
	}
	// Shooting Star: Small body at bottom, long upper wick
	if upperWick > bodySize*2 && lowerWick < bodySize {
		return "Shooting Star"
	}

	// 2. DOJI (Indecision)
	if bodySize <= totalRange*0.1 { // Body is less than 10% of range
		if upperWick > bodySize && lowerWick > bodySize {
			return "Doji"
		}
	}

	// 3. ENGULFING (Two Candles)
	// Bullish Engulfing: Prev Red, Current Green engulfs prev body
	if prevBearish && isBullish {
		if last.Close > prev.Open && last.Open < prev.Close {
			return "Bullish Engulfing"
		}
	}
	// Bearish Engulfing: Prev Green, Current Red engulfs prev body
	if prevBullish && isBearish {
		if last.Close < prev.Open && last.Open > prev.Close {
			return "Bearish Engulfing"
		}
	}

	// 4. MORNING STAR / EVENING STAR (Three Candles)
	// Morning Star: Big Red -> Small Body (Gap Down) -> Big Green (Gap Up)
	if prev2.Close < prev2.Open { // 1. Big Red
		prevBody := math.Abs(prev.Close - prev.Open)
		prevRange := prev.High - prev.Low
		if prevBody < prevRange*0.3 { // 2. Small Body (Doji/Spinning Top)
			if last.Close > last.Open && last.Close > (prev2.Open+prev2.Close)/2 { // 3. Big Green > 50% of first red
				return "Morning Star"
			}
		}
	}

	// Evening Star: Big Green -> Small Body -> Big Red
	if prev2.Close > prev2.Open { // 1. Big Green
		prevBody := math.Abs(prev.Close - prev.Open)
		prevRange := prev.High - prev.Low
		if prevBody < prevRange*0.3 { // 2. Small Body
			if last.Close < last.Open && last.Close < (prev2.Open+prev2.Close)/2 { // 3. Big Red
				return "Evening Star"
			}
		}
	}

	return ""
}
