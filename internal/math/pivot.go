package math

// PivotPoints represents pivot point levels
type PivotPoints struct {
	Pivot float64 // Central pivot point
	R1    float64 // Resistance 1
	R2    float64 // Resistance 2
	R3    float64 // Resistance 3
	S1    float64 // Support 1
	S2    float64 // Support 2
	S3    float64 // Support 3
}

// CalculateStandardPivots calculates standard pivot points
func CalculateStandardPivots(high, low, close float64) PivotPoints {
	pivot := (high + low + close) / 3.0

	r1 := (2 * pivot) - low
	r2 := pivot + (high - low)
	r3 := high + 2*(pivot-low)

	s1 := (2 * pivot) - high
	s2 := pivot - (high - low)
	s3 := low - 2*(high-pivot)

	return PivotPoints{
		Pivot: pivot,
		R1:    r1,
		R2:    r2,
		R3:    r3,
		S1:    s1,
		S2:    s2,
		S3:    s3,
	}
}

// CalculateWoodiesPivots calculates Woodie's pivot points
func CalculateWoodiesPivots(high, low, close float64) PivotPoints {
	pivot := (high + low + 2*close) / 4.0

	r1 := (2 * pivot) - low
	r2 := pivot + (high - low)
	r3 := high + 2*(pivot-low)

	s1 := (2 * pivot) - high
	s2 := pivot - (high - low)
	s3 := low - 2*(high-pivot)

	return PivotPoints{
		Pivot: pivot,
		R1:    r1,
		R2:    r2,
		R3:    r3,
		S1:    s1,
		S2:    s2,
		S3:    s3,
	}
}

// CalculateCamarillaPivots calculates Camarilla pivot points
func CalculateCamarillaPivots(high, low, close float64) PivotPoints {
	range_ := high - low

	r1 := close + (range_ * 1.1 / 12)
	r2 := close + (range_ * 1.1 / 6)
	r3 := close + (range_ * 1.1 / 4)

	s1 := close - (range_ * 1.1 / 12)
	s2 := close - (range_ * 1.1 / 6)
	s3 := close - (range_ * 1.1 / 4)

	return PivotPoints{
		Pivot: close,
		R1:    r1,
		R2:    r2,
		R3:    r3,
		S1:    s1,
		S2:    s2,
		S3:    s3,
	}
}

// FindNearestPivotLevel finds the closest pivot level to current price
func FindNearestPivotLevel(currentPrice float64, pivots PivotPoints) (float64, string) {
	levelMap := map[string]float64{
		"R3":    pivots.R3,
		"R2":    pivots.R2,
		"R1":    pivots.R1,
		"Pivot": pivots.Pivot,
		"S1":    pivots.S1,
		"S2":    pivots.S2,
		"S3":    pivots.S3,
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
