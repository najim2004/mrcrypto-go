package service

import (
	"fmt"
	"log"
	"math"
	"runtime/debug"
)

// RecoverAndLog recovers from panic and logs it with context
func RecoverAndLog(context string) {
	if r := recover(); r != nil {
		log.Printf("❌ [PANIC RECOVERED] %s: %v\n%s", context, r, string(debug.Stack()))
	}
}

// SafeGo launches a goroutine with panic recovery
func SafeGo(name string, fn func()) {
	go func() {
		defer RecoverAndLog(fmt.Sprintf("Goroutine: %s", name))
		fn()
	}()
}

// ValidateFloat64 checks if a float64 is valid (not NaN or Inf)
func ValidateFloat64(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// SafeDivide performs division with zero check
func SafeDivide(numerator, denominator float64) (float64, error) {
	if denominator == 0 {
		return 0, fmt.Errorf("division by zero")
	}

	// Also check for very small denominators that could cause numerical instability
	if math.Abs(denominator) < 1e-10 {
		return 0, fmt.Errorf("denominator too small: %e", denominator)
	}

	result := numerator / denominator

	if !ValidateFloat64(result) {
		return 0, fmt.Errorf("invalid division result: %v / %v = %v", numerator, denominator, result)
	}

	return result, nil
}

// ClampFloat64 clamps a value between min and max
func ClampFloat64(value, min, max float64) float64 {
	if !ValidateFloat64(value) {
		return min
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// SafeSliceAccess checks if an index is valid for a slice
func SafeSliceAccess(length, index int) bool {
	return index >= 0 && index < length
}

// SafeGetLastElement returns the last element of a float64 slice safely
func SafeGetLastElement(slice []float64, defaultValue float64) float64 {
	if len(slice) == 0 {
		return defaultValue
	}
	return slice[len(slice)-1]
}

// SafeGetElement returns an element at index from a float64 slice safely
func SafeGetElement(slice []float64, index int, defaultValue float64) float64 {
	if !SafeSliceAccess(len(slice), index) {
		return defaultValue
	}
	return slice[index]
}

// ValidatePercentage validates that a percentage is reasonable
func ValidatePercentage(percent float64) bool {
	return ValidateFloat64(percent) && percent >= -100 && percent <= 10000
}

// SafeTypeAssertFloat checks type assertion to float64 safely
func SafeTypeAssertFloat(value interface{}, defaultValue float64) float64 {
	if f, ok := value.(float64); ok {
		if ValidateFloat64(f) {
			return f
		}
	}
	return defaultValue
}

// SafeTypeAssertString checks type assertion to string safely
func SafeTypeAssertString(value interface{}, defaultValue string) string {
	if s, ok := value.(string); ok {
		return s
	}
	return defaultValue
}

// ValidatePrice checks if a price value is valid for trading
func ValidatePrice(price float64) bool {
	return ValidateFloat64(price) && price > 0 && price < 1e10
}

// ValidateSliceLength checks if a slice has at least the required length
func ValidateSliceLength(length, required int) error {
	if length < required {
		return fmt.Errorf("insufficient data: got %d, need at least %d", length, required)
	}
	return nil
}
