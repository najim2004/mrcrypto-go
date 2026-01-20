package math

// FibonacciLevels represents Fibonacci retracement levels
type FibonacciLevels struct {
	Level0   float64 // 0% (Top/Bottom)
	Level236 float64 // 23.6%
	Level382 float64 // 38.2%
	Level500 float64 // 50%
	Level618 float64 // 61.8%
	Level786 float64 // 78.6%
	Level100 float64 // 100% (Bottom/Top)
}

// CalculateRetracements calculates Fibonacci retracement levels
func CalculateRetracements(high, low float64, trend string) FibonacciLevels {
	diff := high - low

	if trend == "UP" {
		// Retracing DOWN from High
		return FibonacciLevels{
			Level0:   high,
			Level236: high - diff*0.236,
			Level382: high - diff*0.382,
			Level500: high - diff*0.5,
			Level618: high - diff*0.618,
			Level786: high - diff*0.786,
			Level100: low,
		}
	}

	// Retracing UP from Low
	return FibonacciLevels{
		Level0:   low,
		Level236: low + diff*0.236,
		Level382: low + diff*0.382,
		Level500: low + diff*0.5,
		Level618: low + diff*0.618,
		Level786: low + diff*0.786,
		Level100: high,
	}
}

// CalculateExtension calculates Fibonacci extension (1.618)
func CalculateExtension(high, low float64, trend string) float64 {
	diff := high - low
	if trend == "UP" {
		return high + diff*0.618
	}
	return low - diff*0.618
}

// FindNearestFibLevel finds the closest Fibonacci level to current price
func FindNearestFibLevel(currentPrice float64, levels FibonacciLevels) (float64, string) {
	levelMap := map[string]float64{
		"0%":    levels.Level0,
		"23.6%": levels.Level236,
		"38.2%": levels.Level382,
		"50%":   levels.Level500,
		"61.8%": levels.Level618,
		"78.6%": levels.Level786,
		"100%":  levels.Level100,
	}

	minDiff := 999999.0
	nearestLevel := ""
	nearestPrice := 0.0

	for name, price := range levelMap {
		diff := abs(currentPrice - price)
		if diff < minDiff {
			minDiff = diff
			nearestLevel = name
			nearestPrice = price
		}
	}

	return nearestPrice, nearestLevel
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
