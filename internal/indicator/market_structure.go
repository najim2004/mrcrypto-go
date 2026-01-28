package indicator

import (
	"mrcrypto-go/internal/model"
)

// MarketStructure represents the current market structure state
type MarketStructure string

const (
	StructureBullishBOS   MarketStructure = "BULLISH_BOS"   // Break of Structure (bullish)
	StructureBearishBOS   MarketStructure = "BEARISH_BOS"   // Break of Structure (bearish)
	StructureBullishChoCH MarketStructure = "BULLISH_CHOCH" // Change of Character (bullish)
	StructureBearishChoCH MarketStructure = "BEARISH_CHOCH" // Change of Character (bearish)
	StructureNeutral      MarketStructure = "NEUTRAL"       // No clear structure
)

// StructureInfo contains market structure analysis results
type StructureInfo struct {
	Structure      MarketStructure
	LastSwingHigh  float64
	LastSwingLow   float64
	PreviousHigh   float64
	PreviousLow    float64
	HigherHighs    bool
	HigherLows     bool
	LowerHighs     bool
	LowerLows      bool
	StructureScore int // Confluence score contribution
}

// AnalyzeMarketStructure analyzes price action for BOS and ChoCH
// Uses swing high/low detection to identify structure breaks
func AnalyzeMarketStructure(klines []model.Kline, lookback int) *StructureInfo {
	if len(klines) < lookback+10 {
		return &StructureInfo{Structure: StructureNeutral, StructureScore: 0}
	}

	// Find recent swing highs and lows
	swingHighs := findSwingHighs(klines, 5)
	swingLows := findSwingLows(klines, 5)

	if len(swingHighs) < 2 || len(swingLows) < 2 {
		return &StructureInfo{Structure: StructureNeutral, StructureScore: 0}
	}

	info := &StructureInfo{
		LastSwingHigh: swingHighs[len(swingHighs)-1],
		LastSwingLow:  swingLows[len(swingLows)-1],
		PreviousHigh:  swingHighs[len(swingHighs)-2],
		PreviousLow:   swingLows[len(swingLows)-2],
	}

	// Check for higher highs and higher lows (uptrend)
	info.HigherHighs = info.LastSwingHigh > info.PreviousHigh
	info.HigherLows = info.LastSwingLow > info.PreviousLow

	// Check for lower highs and lower lows (downtrend)
	info.LowerHighs = info.LastSwingHigh < info.PreviousHigh
	info.LowerLows = info.LastSwingLow < info.PreviousLow

	currentPrice := klines[len(klines)-1].Close

	// Determine structure
	switch {
	case info.HigherHighs && info.HigherLows:
		// Bullish structure
		if currentPrice > info.PreviousHigh {
			info.Structure = StructureBullishBOS // Break of structure
			info.StructureScore = 10
		} else {
			info.Structure = StructureNeutral
			info.StructureScore = 5
		}
	case info.LowerHighs && info.LowerLows:
		// Bearish structure
		if currentPrice < info.PreviousLow {
			info.Structure = StructureBearishBOS // Break of structure
			info.StructureScore = 10
		} else {
			info.Structure = StructureNeutral
			info.StructureScore = 5
		}
	case info.HigherLows && info.LowerHighs:
		// Consolidation - wait for breakout
		info.Structure = StructureNeutral
		info.StructureScore = 0
	case !info.HigherHighs && info.HigherLows:
		// Potential bullish ChoCH (first higher low after downtrend)
		info.Structure = StructureBullishChoCH
		info.StructureScore = 8
	case !info.LowerLows && info.LowerHighs:
		// Potential bearish ChoCH (first lower high after uptrend)
		info.Structure = StructureBearishChoCH
		info.StructureScore = 8
	default:
		info.Structure = StructureNeutral
		info.StructureScore = 0
	}

	return info
}

// findSwingHighs finds swing high points in price data
func findSwingHighs(klines []model.Kline, lookback int) []float64 {
	var swingHighs []float64

	for i := lookback; i < len(klines)-lookback; i++ {
		isSwingHigh := true
		currentHigh := klines[i].High

		for j := i - lookback; j <= i+lookback; j++ {
			if j != i && klines[j].High >= currentHigh {
				isSwingHigh = false
				break
			}
		}

		if isSwingHigh {
			swingHighs = append(swingHighs, currentHigh)
		}
	}

	return swingHighs
}

// findSwingLows finds swing low points in price data
func findSwingLows(klines []model.Kline, lookback int) []float64 {
	var swingLows []float64

	for i := lookback; i < len(klines)-lookback; i++ {
		isSwingLow := true
		currentLow := klines[i].Low

		for j := i - lookback; j <= i+lookback; j++ {
			if j != i && klines[j].Low <= currentLow {
				isSwingLow = false
				break
			}
		}

		if isSwingLow {
			swingLows = append(swingLows, currentLow)
		}
	}

	return swingLows
}

// GetStructureScore returns score adjustment based on structure alignment
// direction: "LONG" or "SHORT"
func GetStructureScore(structure *StructureInfo, direction string) int {
	if structure == nil {
		return 0
	}

	switch {
	case direction == "LONG" && (structure.Structure == StructureBullishBOS || structure.Structure == StructureBullishChoCH):
		return structure.StructureScore
	case direction == "SHORT" && (structure.Structure == StructureBearishBOS || structure.Structure == StructureBearishChoCH):
		return structure.StructureScore
	case direction == "LONG" && (structure.Structure == StructureBearishBOS || structure.Structure == StructureBearishChoCH):
		return -10 // Penalty for going against structure
	case direction == "SHORT" && (structure.Structure == StructureBullishBOS || structure.Structure == StructureBullishChoCH):
		return -10 // Penalty for going against structure
	default:
		return 0
	}
}
