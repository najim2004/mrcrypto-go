package service

import (
	"fmt"
	"log"
	"math"
	"time"

	"my-tool-go/internal/indicator"
	internalmath "my-tool-go/internal/math"
	"my-tool-go/internal/model"
)

type StrategyService struct {
	binance *BinanceService
}

func NewStrategyService(binance *BinanceService) *StrategyService {
	return &StrategyService{
		binance: binance,
	}
}

// ========================================
// PROFESSIONAL STRATEGY v2.0
// Proper Order: Context → Key Levels → Regime → Confluence → Entry → Risk
// ========================================

// EvaluateSymbol analyzes a symbol using professional multi-factor confluence approach
func (s *StrategyService) EvaluateSymbol(symbol string) (*model.Signal, error) {
	log.Printf("🔄 [Strategy] Evaluating %s...", symbol)

	// ========================================
	// STEP 1: DATA COLLECTION (Higher TF First)
	// ========================================

	// Daily for pivot points
	klines1d, err := s.binance.GetKlines(symbol, "1d", 10)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch 1d klines: %w", err)
	}

	// 4H for trend direction and key levels
	klines4h, err := s.binance.GetKlines(symbol, "4h", 200)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch 4h klines: %w", err)
	}

	// 1H for confirmation
	klines1h, err := s.binance.GetKlines(symbol, "1h", 200)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch 1h klines: %w", err)
	}

	// 15m for alignment
	klines15m, err := s.binance.GetKlines(symbol, "15m", 200)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch 15m klines: %w", err)
	}

	// 5m for entry timing
	klines5m, err := s.binance.GetKlines(symbol, "5m", 200)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch 5m klines: %w", err)
	}

	// Extract price arrays
	closes4h := extractCloses(klines4h)
	highs4h := extractHighs(klines4h)
	lows4h := extractLows(klines4h)

	closes1h := extractCloses(klines1h)
	highs1h := extractHighs(klines1h)
	lows1h := extractLows(klines1h)

	closes15m := extractCloses(klines15m)
	highs15m := extractHighs(klines15m)
	lows15m := extractLows(klines15m)

	closes5m := extractCloses(klines5m)
	highs5m := extractHighs(klines5m)
	lows5m := extractLows(klines5m)
	volumes5m := extractVolumes(klines5m)

	currentPrice := closes5m[len(closes5m)-1]

	// ========================================
	// STEP 2: KEY LEVELS (Before anything else)
	// ========================================
	log.Printf("⏳ [Strategy] %s - Calculating key levels...", symbol)

	// Daily Pivot Points
	var pivotPoints internalmath.PivotPoints
	if len(klines1d) >= 2 {
		prevDay := klines1d[len(klines1d)-2]
		pivotPoints = internalmath.CalculateStandardPivots(prevDay.High, prevDay.Low, prevDay.Close)
	}
	nearestPivotPrice, nearestPivotName := internalmath.FindNearestPivotLevel(currentPrice, pivotPoints)

	// 4H Swing High/Low for Fibonacci
	high4h, low4h := findSwingHighLow(highs4h, lows4h, 50)

	// Determine trend for Fib calculation
	ema50_4h := indicator.CalculateEMA(closes4h, 50)
	if len(ema50_4h) == 0 {
		return nil, nil
	}
	ema50Value := ema50_4h[len(ema50_4h)-1]

	fibTrend := "UP"
	if currentPrice < ema50Value {
		fibTrend = "DOWN"
	}
	fibLevels := internalmath.CalculateRetracements(high4h, low4h, fibTrend)
	nearestFibPrice, nearestFibName := internalmath.FindNearestFibLevel(currentPrice, fibLevels)

	// ATR for volatility-based stops
	atr1h := internalmath.CalculateATR(highs1h, lows1h, closes1h, 14)

	// ========================================
	// STEP 3: INDICATOR CALCULATION
	// ========================================
	log.Printf("⏳ [Strategy] %s - Calculating indicators...", symbol)

	// RSI - Multi-timeframe
	rsi4h := indicator.GetLastRSI(closes4h, 14)
	rsi1h := indicator.GetLastRSI(closes1h, 14)
	rsi15m := indicator.GetLastRSI(closes15m, 14)
	rsi5m := indicator.GetLastRSI(closes5m, 14)

	// ADX - Trend strength
	adx4h := indicator.GetLastADX(highs4h, lows4h, closes4h, 14)
	adx1h := indicator.GetLastADX(highs1h, lows1h, closes1h, 14)
	adx15m := indicator.GetLastADX(highs15m, lows15m, closes15m, 14)

	// Validate data
	if rsi4h == 0 || rsi1h == 0 || adx4h == 0 {
		log.Printf("⚠️  [Strategy] %s - Insufficient data", symbol)
		return nil, nil
	}

	// VWAP & MACD
	vwap := indicator.GetLastVWAP(highs5m, lows5m, closes5m, volumes5m)
	macd, macdSignal, histogram := indicator.GetLastMACD(closes5m, 12, 26, 9)

	// Volume
	avgVol := calculateAverage(volumes5m)
	currentVol := volumes5m[len(volumes5m)-1]
	volRatio := currentVol / avgVol

	// Order Flow
	orderFlowDelta := calculateOrderFlowDelta(klines5m)

	// ========================================
	// STEP 4: REGIME DETECTION
	// ========================================
	regime := detectRegimePro(adx4h, adx1h, currentPrice, ema50Value)

	log.Printf("ℹ️  [Strategy] %s - Regime: %s (ADX4h: %.1f, ADX1h: %.1f)",
		symbol, regime, adx4h, adx1h)

	// Skip choppy markets early
	if regime == model.RegimeChoppy {
		log.Printf("⏭️  [Strategy] %s - Skipped (choppy ADX < 15)", symbol)
		return nil, nil
	}

	// ========================================
	// STEP 5: CONFLUENCE SCORING (0-100)
	// ========================================
	signalDir := determineSignalDirection(regime, currentPrice, ema50Value, rsi4h)
	if signalDir == "" {
		log.Printf("⏭️  [Strategy] %s - No clear direction", symbol)
		return nil, nil
	}

	score := calculateConfluenceScore(
		signalDir, regime,
		rsi4h, rsi1h, rsi15m, rsi5m,
		adx4h, adx1h, adx15m,
		histogram, volRatio, orderFlowDelta,
		currentPrice, pivotPoints, fibLevels,
	)

	log.Printf("📊 [Strategy] %s - Confluence Score: %d/100 (Dir: %s)", symbol, score, signalDir)

	// Minimum score threshold
	if score < 60 {
		log.Printf("⏭️  [Strategy] %s - Score too low (%d < 60)", symbol, score)
		return nil, nil
	}

	// ========================================
	// STEP 6: KEY LEVEL PROXIMITY CHECK
	// ========================================
	pivotProximity := math.Abs(currentPrice-nearestPivotPrice) / currentPrice * 100
	fibProximity := math.Abs(currentPrice-nearestFibPrice) / currentPrice * 100

	// Must be within 2% of a key level for entry
	nearKeyLevel := pivotProximity <= 2.0 || fibProximity <= 2.0
	if !nearKeyLevel && score < 80 {
		log.Printf("⏭️  [Strategy] %s - Not near key level (Pivot: %.2f%%, Fib: %.2f%%)",
			symbol, pivotProximity, fibProximity)
		return nil, nil
	}

	// ========================================
	// STEP 7: DETERMINE TIER
	// ========================================
	tier := model.TierStandard
	if score >= 80 && adx4h >= 30 && volRatio >= 2.0 {
		tier = model.TierPremium
	}

	// ========================================
	// STEP 8: CALCULATE ATR-BASED SL/TP
	// ========================================
	var stopLoss, takeProfit float64
	atrMultiplierSL := 1.5
	atrMultiplierTP := 3.0 // 2:1 R:R minimum

	if signalDir == "LONG" {
		stopLoss = currentPrice - (atr1h * atrMultiplierSL)
		// TP at next resistance or ATR target
		tpATR := currentPrice + (atr1h * atrMultiplierTP)
		tpPivot := getNextResistance(currentPrice, pivotPoints)
		takeProfit = math.Max(tpATR, tpPivot)
	} else {
		stopLoss = currentPrice + (atr1h * atrMultiplierSL)
		// TP at next support or ATR target
		tpATR := currentPrice - (atr1h * atrMultiplierTP)
		tpPivot := getNextSupport(currentPrice, pivotPoints)
		takeProfit = math.Min(tpATR, tpPivot)
	}

	// ========================================
	// STEP 9: RISK MANAGEMENT
	// ========================================
	rrResult := internalmath.CalculateRiskReward(currentPrice, stopLoss, takeProfit)

	// Minimum 2:1 R:R required
	if rrResult.Ratio < 2.0 {
		log.Printf("⏭️  [Strategy] %s - R:R too low (%.2f < 2.0)", symbol, rrResult.Ratio)
		return nil, nil
	}

	// Position sizing with Kelly Criterion
	recommendedSize := internalmath.CalculateKellyCriterion(0.55, rrResult.Ratio, 1.0)

	// ========================================
	// STEP 10: BUILD SIGNAL
	// ========================================
	signalType := model.SignalTypeLong
	if signalDir == "SHORT" {
		signalType = model.SignalTypeShort
	}

	techContext := model.TechnicalContext{
		RSI4h:          rsi4h,
		RSI1h:          rsi1h,
		RSI15m:         rsi15m,
		RSI5m:          rsi5m,
		ADX4h:          adx4h,
		ADX1h:          adx1h,
		ADX15m:         adx15m,
		VWAP:           vwap,
		CurrentVol:     currentVol,
		AvgVol:         avgVol,
		MACD:           macd,
		Signal:         macdSignal,
		Histogram:      histogram,
		OrderFlowDelta: orderFlowDelta,
		Regime:         string(regime),
		PivotPoint:     pivotPoints.Pivot,
		PivotR1:        pivotPoints.R1,
		PivotR2:        pivotPoints.R2,
		PivotR3:        pivotPoints.R3,
		PivotS1:        pivotPoints.S1,
		PivotS2:        pivotPoints.S2,
		PivotS3:        pivotPoints.S3,
		NearestPivot:   nearestPivotName,
		Fib236:         fibLevels.Level236,
		Fib382:         fibLevels.Level382,
		Fib500:         fibLevels.Level500,
		Fib618:         fibLevels.Level618,
		Fib786:         fibLevels.Level786,
		NearestFib:     nearestFibName,
	}

	signal := &model.Signal{
		Symbol:           symbol,
		Type:             signalType,
		Tier:             tier,
		EntryPrice:       currentPrice,
		StopLoss:         stopLoss,
		TakeProfit:       takeProfit,
		RiskRewardRatio:  rrResult.Ratio,
		RecommendedSize:  recommendedSize,
		Regime:           string(regime),
		TechnicalContext: techContext,
		Status:           "ACTIVE",
		Timestamp:        time.Now(),
	}

	log.Printf("✨ [Strategy] %s - %s signal! Score: %d, Tier: %s, R:R: %.2f, Entry: %s, SL: %s, TP: %s",
		symbol, signalDir, score, tier, rrResult.Ratio,
		FormatPrice(currentPrice), FormatPrice(stopLoss), FormatPrice(takeProfit))

	return signal, nil
}

// ========================================
// PROFESSIONAL HELPER FUNCTIONS
// ========================================

// detectRegimePro uses multi-timeframe ADX for better regime detection
func detectRegimePro(adx4h, adx1h, price, ema50 float64) model.MarketRegime {
	avgADX := (adx4h + adx1h) / 2

	if avgADX < 15 {
		return model.RegimeChoppy
	}

	if avgADX < 20 {
		return model.RegimeRanging
	}

	// Strong trend
	if price > ema50 {
		return model.RegimeTrendingUp
	}
	return model.RegimeTrendingDown
}

// determineSignalDirection determines if we should look for LONG or SHORT
func determineSignalDirection(regime model.MarketRegime, price, ema50, rsi4h float64) string {
	if regime == model.RegimeTrendingUp && price > ema50 && rsi4h < 70 {
		return "LONG"
	}
	if regime == model.RegimeTrendingDown && price < ema50 && rsi4h > 30 {
		return "SHORT"
	}
	return ""
}

// calculateConfluenceScore calculates weighted confluence score (0-100)
func calculateConfluenceScore(
	direction string, regime model.MarketRegime,
	rsi4h, rsi1h, rsi15m, rsi5m float64,
	adx4h, adx1h, adx15m float64,
	histogram, volRatio, orderFlow float64,
	price float64, pivots internalmath.PivotPoints, fibs internalmath.FibonacciLevels,
) int {
	score := 0

	// 1. Trend Alignment (4H + 1H same direction) - 25 points
	if regime == model.RegimeTrendingUp || regime == model.RegimeTrendingDown {
		if adx4h > 20 && adx1h > 20 {
			score += 25
		} else if adx4h > 20 || adx1h > 20 {
			score += 15
		}
	}

	// 2. RSI Momentum (not overbought/oversold) - 20 points
	if direction == "LONG" {
		if rsi4h > 40 && rsi4h < 65 && rsi1h > 45 && rsi1h < 70 {
			score += 20
		} else if rsi4h > 35 && rsi4h < 70 {
			score += 10
		}
	} else {
		if rsi4h > 35 && rsi4h < 60 && rsi1h > 30 && rsi1h < 55 {
			score += 20
		} else if rsi4h > 30 && rsi4h < 65 {
			score += 10
		}
	}

	// 3. Key Level Proximity - 20 points
	pivotDist := getPivotDistance(price, pivots)
	fibDist := getFibDistance(price, fibs)

	if pivotDist <= 1.0 || fibDist <= 1.0 {
		score += 20
	} else if pivotDist <= 2.0 || fibDist <= 2.0 {
		score += 12
	} else if pivotDist <= 3.0 || fibDist <= 3.0 {
		score += 5
	}

	// 4. Volume Confirmation - 15 points
	if volRatio >= 2.0 {
		score += 15
	} else if volRatio >= 1.5 {
		score += 10
	} else if volRatio >= 1.2 {
		score += 5
	}

	// 5. MACD Alignment - 10 points
	if direction == "LONG" && histogram > 0 {
		score += 10
	} else if direction == "SHORT" && histogram < 0 {
		score += 10
	}

	// 6. Order Flow - 10 points
	if direction == "LONG" && orderFlow > 0 {
		score += 10
	} else if direction == "SHORT" && orderFlow < 0 {
		score += 10
	}

	return score
}

// findSwingHighLow finds swing high and low from recent candles
func findSwingHighLow(highs, lows []float64, lookback int) (float64, float64) {
	if len(highs) < lookback {
		lookback = len(highs)
	}

	high := highs[len(highs)-lookback]
	low := lows[len(lows)-lookback]

	for i := len(highs) - lookback; i < len(highs); i++ {
		if highs[i] > high {
			high = highs[i]
		}
		if lows[i] < low {
			low = lows[i]
		}
	}

	return high, low
}

// getPivotDistance returns minimum % distance to any pivot level
func getPivotDistance(price float64, pivots internalmath.PivotPoints) float64 {
	levels := []float64{pivots.Pivot, pivots.R1, pivots.R2, pivots.R3, pivots.S1, pivots.S2, pivots.S3}
	minDist := 100.0

	for _, level := range levels {
		if level == 0 {
			continue
		}
		dist := math.Abs(price-level) / price * 100
		if dist < minDist {
			minDist = dist
		}
	}

	return minDist
}

// getFibDistance returns minimum % distance to any fib level
func getFibDistance(price float64, fibs internalmath.FibonacciLevels) float64 {
	levels := []float64{fibs.Level236, fibs.Level382, fibs.Level500, fibs.Level618, fibs.Level786}
	minDist := 100.0

	for _, level := range levels {
		if level == 0 {
			continue
		}
		dist := math.Abs(price-level) / price * 100
		if dist < minDist {
			minDist = dist
		}
	}

	return minDist
}

// getNextResistance finds next resistance level above price
func getNextResistance(price float64, pivots internalmath.PivotPoints) float64 {
	levels := []float64{pivots.Pivot, pivots.R1, pivots.R2, pivots.R3}
	nextRes := price * 1.10 // Default 10% above

	for _, level := range levels {
		if level > price && level < nextRes {
			nextRes = level
		}
	}

	return nextRes
}

// getNextSupport finds next support level below price
func getNextSupport(price float64, pivots internalmath.PivotPoints) float64 {
	levels := []float64{pivots.Pivot, pivots.S1, pivots.S2, pivots.S3}
	nextSup := price * 0.90 // Default 10% below

	for _, level := range levels {
		if level < price && level > nextSup {
			nextSup = level
		}
	}

	return nextSup
}

// ========================================
// UTILITY FUNCTIONS
// ========================================

func extractCloses(klines []model.Kline) []float64 {
	closes := make([]float64, len(klines))
	for i, k := range klines {
		closes[i] = k.Close
	}
	return closes
}

func extractHighs(klines []model.Kline) []float64 {
	highs := make([]float64, len(klines))
	for i, k := range klines {
		highs[i] = k.High
	}
	return highs
}

func extractLows(klines []model.Kline) []float64 {
	lows := make([]float64, len(klines))
	for i, k := range klines {
		lows[i] = k.Low
	}
	return lows
}

func extractVolumes(klines []model.Kline) []float64 {
	volumes := make([]float64, len(klines))
	for i, k := range klines {
		volumes[i] = k.Volume
	}
	return volumes
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
