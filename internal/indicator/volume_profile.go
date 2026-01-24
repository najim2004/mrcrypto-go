package indicator

import (
	"math"
	"mrcrypto-go/internal/model"
)

// VolumeProfileLevel represents a price level with its volume
type VolumeProfileLevel struct {
	Price  float64
	Volume float64
}

// VolumeProfile holds the profile data including POC
type VolumeProfile struct {
	Levels []VolumeProfileLevel
	POC    float64 // Point of Control (Price with max volume)
	VAHigh float64 // Value Area High
	VALow  float64 // Value Area Low
}

// CalculateVolumeProfile calculates the volume profile for the given klines
// It bins the price range into `numBins` buckets
func CalculateVolumeProfile(klines []model.Kline, numBins int) VolumeProfile {
	if len(klines) == 0 {
		return VolumeProfile{}
	}

	// Find high and low of the entire range
	minPrice := klines[0].Low
	maxPrice := klines[0].High
	totalVolume := 0.0

	for _, k := range klines {
		if k.Low < minPrice {
			minPrice = k.Low
		}
		if k.High > maxPrice {
			maxPrice = k.High
		}
		totalVolume += k.Volume
	}

	if numBins <= 0 {
		numBins = 50 // Default
	}

	rangeSize := maxPrice - minPrice
	binSize := rangeSize / float64(numBins)

	// Create bins
	bins := make([]float64, numBins)

	// Distribute volume to bins
	// Simplified approach: Add total volume of candle to the bin corresponding to its Close price
	// Make it better: Distribute between Low and High? (More expensive)
	// Let's stick to Close price for now for performance, or better, (High+Low+Close)/3
	for _, k := range klines {
		avgPrice := (k.High + k.Low + k.Close) / 3.0
		binIndex := int((avgPrice - minPrice) / binSize)
		if binIndex >= numBins {
			binIndex = numBins - 1
		}
		if binIndex < 0 {
			binIndex = 0
		}
		bins[binIndex] += k.Volume
	}

	// Find POC
	maxVol := 0.0
	pocIndex := 0
	var levels []VolumeProfileLevel

	for i, vol := range bins {
		price := minPrice + (float64(i) * binSize) + (binSize / 2) // Midpoint of bin
		levels = append(levels, VolumeProfileLevel{
			Price:  price,
			Volume: vol,
		})

		if vol > maxVol {
			maxVol = vol
			pocIndex = i
		}
	}

	pocPrice := minPrice + (float64(pocIndex) * binSize) + (binSize / 2)

	// Calculate Value Area (70% of total volume around POC)
	targetVol := totalVolume * 0.70
	currentVol := maxVol

	// Start from POC and expand out
	upIdx := pocIndex
	downIdx := pocIndex

	for currentVol < targetVol {
		canGoUp := upIdx < numBins-1
		canGoDown := downIdx > 0

		if !canGoUp && !canGoDown {
			break
		}

		nextUpVol := 0.0
		if canGoUp {
			nextUpVol = bins[upIdx+1]
		}

		nextDownVol := 0.0
		if canGoDown {
			nextDownVol = bins[downIdx-1]
		}

		if nextUpVol > nextDownVol {
			currentVol += nextUpVol
			upIdx++
		} else {
			currentVol += nextDownVol
			downIdx--
		}
	}

	vaHigh := minPrice + (float64(upIdx) * binSize) + (binSize / 2)
	vaLow := minPrice + (float64(downIdx) * binSize) + (binSize / 2)

	return VolumeProfile{
		Levels: levels,
		POC:    pocPrice,
		VAHigh: vaHigh,
		VALow:  vaLow,
	}
}

// GetNearestPOC finds the closest POC from multiple timeframes or just returns the calculated one
func GetPOCDistance(currentPrice float64, poc float64) float64 {
	return math.Abs(currentPrice-poc) / currentPrice * 100
}
