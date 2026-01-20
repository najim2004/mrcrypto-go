package math

import "math"

// CalculatePercentageChange calculates percentage change between two values
func CalculatePercentageChange(oldValue, newValue float64) float64 {
	if oldValue == 0 {
		return 0
	}
	return ((newValue - oldValue) / oldValue) * 100
}

// CalculatePercentageDifference calculates percentage difference
func CalculatePercentageDifference(value1, value2 float64) float64 {
	avg := (value1 + value2) / 2.0
	if avg == 0 {
		return 0
	}
	return (math.Abs(value1-value2) / avg) * 100
}

// CalculatePercentageOf calculates what percentage one value is of another
func CalculatePercentageOf(part, whole float64) float64 {
	if whole == 0 {
		return 0
	}
	return (part / whole) * 100
}

// AddPercentage adds percentage to a value
func AddPercentage(value, percentage float64) float64 {
	return value + (value * percentage / 100.0)
}

// SubtractPercentage subtracts percentage from a value
func SubtractPercentage(value, percentage float64) float64 {
	return value - (value * percentage / 100.0)
}

// CalculateCompoundGrowth calculates compound growth
func CalculateCompoundGrowth(initial, rate float64, periods int) float64 {
	return initial * math.Pow(1+(rate/100.0), float64(periods))
}

// CalculateGrowthRate calculates growth rate over period
func CalculateGrowthRate(initialValue, finalValue float64, periods int) float64 {
	if initialValue == 0 || periods == 0 {
		return 0
	}
	return (math.Pow(finalValue/initialValue, 1.0/float64(periods)) - 1) * 100
}
