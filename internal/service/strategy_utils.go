package service

import (
	"fmt"
	"mrcrypto-go/internal/model"
)

// UTILITY FUNCTIONS

func extractSeries(klines []model.Kline) ([]float64, []float64, []float64, []float64) {
	length := len(klines)
	closes := make([]float64, length)
	highs := make([]float64, length)
	lows := make([]float64, length)
	volumes := make([]float64, length)

	for i, k := range klines {
		closes[i] = k.Close
		highs[i] = k.High
		lows[i] = k.Low
		volumes[i] = k.Volume
	}
	return closes, highs, lows, volumes
}

func calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateOrderFlowDelta(klines []model.Kline) float64 {
	delta := 0.0
	start := len(klines) - 20
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		k := klines[i]
		if k.Close > k.Open {
			delta += k.Volume
		} else {
			delta -= k.Volume
		}
	}
	return delta
}

func CalculateDynamicDecimals(price float64) int {
	if price < 0.00001 {
		return 8
	} else if price < 0.0001 {
		return 7
	} else if price < 0.001 {
		return 6
	} else if price < 0.01 {
		return 5
	} else if price < 0.1 {
		return 4
	} else if price < 1 {
		return 3
	} else if price < 10 {
		return 2
	}
	return 2
}

func FormatPrice(price float64) string {
	decimals := CalculateDynamicDecimals(price)
	format := fmt.Sprintf("%%.%df", decimals)
	return fmt.Sprintf(format, price)
}
