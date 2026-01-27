package service

import (
	"fmt"
	"log"
	"math"
	"time"

	"mrcrypto-go/internal/indicator"
	internalmath "mrcrypto-go/internal/math"
	"mrcrypto-go/internal/model"
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

	// ========================================
	// STEP 1.1: FETCH BTC CONTEXT (Correlation)
	// ========================================
	var btcTrend string
	if symbol != "BTCUSDT" {
		btcKlines4h, err := s.binance.GetKlines("BTCUSDT", "4h", 200)
		if err == nil {
			btcCloses, _, _, _ := extractSeries(btcKlines4h)
			btcEma50 := indicator.CalculateEMA(btcCloses, 50)
			if len(btcEma50) > 0 {
				lastBtcPrice := btcCloses[len(btcCloses)-1]
				lastBtcEma := btcEma50[len(btcEma50)-1]
				if lastBtcPrice > lastBtcEma {
					btcTrend = "UP"
				} else {
					btcTrend = "DOWN"
				}
			}
		} else {
			log.Printf("⚠️  [Strategy] Failed to fetch BTC klines: %v", err)
		}
	}

	// Extract price arrays
	closes4h, highs4h, lows4h, _ := extractSeries(klines4h)
	closes1h, highs1h, lows1h, _ := extractSeries(klines1h)
	closes15m, highs15m, lows15m, _ := extractSeries(klines15m)
	closes5m, highs5m, lows5m, volumes5m := extractSeries(klines5m)

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
	log.Printf("Debug: Calculating RSI for %s...", symbol)
	rsi4h := indicator.GetLastRSI(closes4h, 14)
	rsi1h := indicator.GetLastRSI(closes1h, 14)
	rsi15m := indicator.GetLastRSI(closes15m, 14)
	rsi5m := indicator.GetLastRSI(closes5m, 14)

	// ADX - Trend strength
	log.Printf("Debug: Calculating ADX for %s...", symbol)
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
	log.Printf("Debug: Calculating Order Flow for %s...", symbol)
	orderFlowDelta := calculateOrderFlowDelta(klines5m)

	// ========================================
	// STEP 3.1: SMC & VOLUME PROFILE
	// ========================================
	// SMC (Smart Money Concepts) - Use 1H for reliability
	log.Printf("Debug: Calculating SMC for %s...", symbol)
	fvgs := indicator.FindFVGs(klines1h)
	obs := indicator.FindOrderBlocks(klines1h)

	inFVG, fvgType := indicator.IsPriceInFVG(currentPrice, fvgs)
	inOB, obType := indicator.IsPriceInOB(currentPrice, obs)

	// Volume Profile - Use 4H for major levels
	log.Printf("Debug: Calculating Volume Profile for %s...", symbol)
	vp := indicator.CalculateVolumeProfile(klines4h, 100)
	pocDist := indicator.GetPOCDistance(currentPrice, vp.POC)

	// ========================================
	// STEP 4: REGIME DETECTION
	// ========================================
	// Use 1H and 15m for faster regime detection
	regime := detectRegimePro(adx1h, adx15m, currentPrice, ema50Value)

	log.Printf("ℹ️  [Strategy] %s - Regime: %s (ADX1h: %.1f, ADX15m: %.1f)",
		symbol, regime, adx1h, adx15m)

	// Skip choppy markets early
	if regime == model.RegimeChoppy {
		log.Printf("⏭️  [Strategy] %s - Skipped (choppy ADX < 20)", symbol)
		return nil, nil
	}

	// ========================================
	// STEP 5: CONFLUENCE SCORING (0-100)
	// ========================================
	// ========================================
	// STEP 5: CONFLUENCE SCORING (Strict 0-100)
	// ========================================
	signalDir := determineSignalDirection(regime, currentPrice, ema50Value, rsi4h)
	if signalDir == "" {
		log.Printf("⏭️  [Strategy] %s - No clear direction", symbol)
		return nil, nil
	}

	// Calculate Score (Max 100)
	score := calculateConfluenceScore(
		signalDir, regime,
		rsi4h, rsi1h, rsi15m, // Added rsi15m
		adx4h, adx1h, adx15m, // Added adx15m
		histogram, volRatio, orderFlowDelta,
		currentPrice, pivotPoints, fibLevels,
		btcTrend, inFVG, fvgType, inOB, obType, pocDist,
	)

	log.Printf("📊 [Strategy] %s - Confluence Score: %d/100 (Dir: %s)", symbol, score, signalDir)

	// Minimum score threshold (Strict 70)
	if score < 70 {
		log.Printf("⏭️  [Strategy] %s - Score too low (%d < 70)", symbol, score)
		return nil, nil
	}

	// ========================================
	// STEP 6: KEY LEVEL PROXIMITY CHECK
	// ========================================
	pivotProximity := math.Abs(currentPrice-nearestPivotPrice) / currentPrice * 100
	fibProximity := math.Abs(currentPrice-nearestFibPrice) / currentPrice * 100

	// Must be within 2% of a key level for entry
	// EXCEPTION: If score is Premium (>= 90), we allow slightly wider entry
	nearKeyLevel := pivotProximity <= 2.0 || fibProximity <= 2.0
	if !nearKeyLevel && score < 90 {
		log.Printf("⏭️  [Strategy] %s - Not near key level (Pivot: %.2f%%, Fib: %.2f%%)",
			symbol, pivotProximity, fibProximity)
		return nil, nil
	}

	// ========================================
	// STEP 7: DETERMINE TIER
	// ========================================
	tier := model.TierStandard
	if score >= 90 {
		tier = model.TierPremium
	}

	// ========================================
	// ========================================
	// STEP 8: CALCULATE SL/TP WITH PROPER R:R
	// ========================================
	var stopLoss, takeProfit1, takeProfit2 float64

	// Use percentage-based SL/TP for consistent R:R
	// SL: 2%
	// TP1: 3% (1:1.5 R:R) -> Book 50%
	// TP2: 6% (1:3 R:R) -> Book 50%
	slPercent := 2.0 / 100.0
	tp1Percent := 3.0 / 100.0
	tp2Percent := 6.0 / 100.0

	// Adjust based on ATR volatility
	atrPercent := (atr1h / currentPrice) * 100
	if atrPercent > 3.0 {
		// High volatility - use wider stops/targets
		slPercent = 3.0 / 100.0
		tp1Percent = 4.5 / 100.0
		tp2Percent = 9.0 / 100.0
	} else if atrPercent < 1.0 {
		// Low volatility - use tighter stops/targets
		slPercent = 1.5 / 100.0
		tp1Percent = 2.25 / 100.0
		tp2Percent = 4.5 / 100.0
	}

	if signalDir == "LONG" {
		// LONG calculation
		stopLoss = currentPrice * (1 - slPercent)
		takeProfit1 = currentPrice * (1 + tp1Percent)
		takeProfit2 = currentPrice * (1 + tp2Percent)

		// Optional: Adjust TP2 to resistance if meaningful
		tpPivot := getNextResistance(currentPrice, pivotPoints)
		if tpPivot > takeProfit2 && tpPivot < currentPrice*1.15 {
			takeProfit2 = tpPivot
			// Recalculate TP2 percent if adjusted
			tp2Percent = (takeProfit2 - currentPrice) / currentPrice
		}
	} else {
		// SHORT calculation
		stopLoss = currentPrice * (1 + slPercent)
		takeProfit1 = currentPrice * (1 - tp1Percent)
		takeProfit2 = currentPrice * (1 - tp2Percent)

		// Optional: Adjust TP2 to support if meaningful
		tpPivot := getNextSupport(currentPrice, pivotPoints)
		if tpPivot < takeProfit2 && tpPivot > currentPrice*0.85 {
			takeProfit2 = tpPivot
			// Recalculate TP2 percent if adjusted
			tp2Percent = (currentPrice - takeProfit2) / currentPrice // absolute % change
		}
	}

	// ========================================
	// STEP 9: RISK MANAGEMENT & PROBABILITY
	// ========================================
	// Calculate R:R based on TP2 (Main Target)
	rrResult := internalmath.CalculateRiskReward(currentPrice, stopLoss, takeProfit2)

	// Minimum 2:1 R:R required (based on final target)
	if rrResult.Ratio < 2.0 {
		log.Printf("⏭️  [Strategy] %s - R:R too low (%.2f < 2.0)", symbol, rrResult.Ratio)
		return nil, nil
	}

	// Calculate probability metrics
	signalProbability := internalmath.CalculateSignalProbability(score)
	breakEvenWinRate := internalmath.CalculateBreakEvenWinRate(rrResult.Ratio)

	// Calculate percentages
	riskPercent := math.Abs(currentPrice-stopLoss) / currentPrice * 100
	rewardPercent := math.Abs(takeProfit2-currentPrice) / currentPrice * 100
	tp1PercentVal := math.Abs(takeProfit1-currentPrice) / currentPrice * 100

	// Calculate nearest level distance
	nearestLevelDist := math.Min(pivotProximity, fibProximity)

	// Dynamic Kelly position sizing using signal probability
	recommendedSize := internalmath.CalculateKellyCriterion(signalProbability, rrResult.Ratio, 1.0)

	// ========================================
	// STEP 10: BUILD SIGNAL WITH PROBABILITY DATA
	// ========================================
	// Populate SMC/VP context
	smcFVGType := ""
	if inFVG {
		smcFVGType = fvgType
	}
	smcOBType := ""
	if inOB {
		smcOBType = obType
	}
	signalType := model.SignalTypeLong
	if signalDir == "SHORT" {
		signalType = model.SignalTypeShort
	}

	techContext := model.TechnicalContext{
		RSI4h:          rsi4h,
		RSI1h:          rsi1h,
		RSI15m:         rsi15m, // Still logging these but score uses less
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
		// New Context
		BTCCorrelation: btcTrend,
		FVGType:        smcFVGType,
		OBType:         smcOBType,
		POC:            vp.POC,
		POCDistance:    pocDist,
	}

	signal := &model.Signal{
		Symbol:           symbol,
		Type:             signalType,
		Tier:             tier,
		EntryPrice:       currentPrice,
		StopLoss:         stopLoss,
		TakeProfit:       takeProfit2, // Main TP for legacy consistency
		TakeProfit1:      takeProfit1,
		TakeProfit2:      takeProfit2,
		RiskRewardRatio:  rrResult.Ratio,
		RecommendedSize:  recommendedSize,
		Regime:           string(regime),
		TechnicalContext: techContext,
		// Probability Fields
		ConfidenceScore:  signalProbability,
		ConfluenceScore:  score,
		BreakEvenWinRate: breakEvenWinRate,
		RiskPercent:      riskPercent,
		RewardPercent:    rewardPercent,
		TP1Percent:       tp1PercentVal,
		TP2Percent:       rewardPercent, // Same as RewardPercent
		NearestLevelDist: nearestLevelDist,
		// Status
		Status:    "ACTIVE",
		Timestamp: time.Now(),
	}

	log.Printf("✨ [Strategy] %s - %s signal! Score: %d (%.0f%% prob), Tier: %s, R:R: %.2f, Entry: %s, SL: %s (%.2f%%)",
		symbol, signalDir, score, signalProbability*100, tier, rrResult.Ratio,
		FormatPrice(currentPrice), FormatPrice(stopLoss), riskPercent)

	return signal, nil
}

// detectRegimePro uses multi-timeframe ADX for better regime detection
func detectRegimePro(adx1h, adx15m, price, ema50 float64) model.MarketRegime {
	avgADX := (adx1h + adx15m) / 2

	if avgADX < 20 {
		return model.RegimeChoppy
	}

	if avgADX < 25 {
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

// calculateConfluenceScore calculates strict weighted score (Max 100)
//
// WEIGTHS DISTRIBUTION (TOTAL: 100):
// 1. Trend Alignment: 		20 (1h + 15m)
// 2. RSI Momentum: 		25 (15m Priority)
// 3. Key Level: 			15
// 4. Volume: 				10
// 5. Order Flow: 			5
// 6. MACD:					5
// 7. SMC (OB/FVG): 		10
// 8. Volume Profile (POC): 5
// 9. BTC Correlation: 		5
func calculateConfluenceScore(
	direction string, regime model.MarketRegime,
	rsi4h, rsi1h, rsi15m float64,
	adx4h, adx1h, adx15m float64,
	histogram, volRatio, orderFlow float64,
	price float64, pivots internalmath.PivotPoints, fibs internalmath.FibonacciLevels,
	btcTrend string, inFVG bool, fvgType string, inOB bool, obType string, pocDist float64,
) int {
	score := 0

	// 1. Trend Alignment (Max 20)
	// Strong trend in both 1H and 15m
	if adx1h > 25 && adx15m > 25 {
		score += 20
	} else if adx1h > 25 || adx15m > 25 {
		score += 10
	}

	// 2. RSI Momentum (Max 25) - 15m Priority for entries
	if direction == "LONG" {
		// Ideal entry: 15m RSI oversold (pullback) in uptrend
		if rsi15m < 45 && rsi15m > 30 {
			score += 25 // Perfect pullback entry
		} else if rsi15m < 60 && rsi1h < 70 {
			score += 15 // Good continuation
		}
	} else {
		// Ideal entry: 15m RSI overbought (pullback) in downtrend
		if rsi15m > 55 && rsi15m < 70 {
			score += 25 // Perfect pullback entry
		} else if rsi15m > 40 && rsi1h > 30 {
			score += 15 // Good continuation
		}
	}

	// 3. Key Level Proximity (Max 15)
	pivotDist := getPivotDistance(price, pivots)
	fibDist := getFibDistance(price, fibs)

	if pivotDist <= 1.5 || fibDist <= 1.5 {
		score += 15
	} else if pivotDist <= 2.5 || fibDist <= 2.5 {
		score += 8
	}

	// 4. Volume (Max 10)
	if volRatio >= 1.5 {
		score += 10
	} else if volRatio >= 1.2 {
		score += 5
	}

	// 5. Order Flow (Max 5)
	if (direction == "LONG" && orderFlow > 0) || (direction == "SHORT" && orderFlow < 0) {
		score += 5
	}

	// 6. MACD (Max 5)
	if (direction == "LONG" && histogram > 0) || (direction == "SHORT" && histogram < 0) {
		score += 5
	}

	// 7. SMC (OB/FVG) (Max 10)
	smcScore := 0
	if inOB {
		if (direction == "LONG" && obType == "BULLISH") || (direction == "SHORT" && obType == "BEARISH") {
			smcScore += 5
		}
	}
	if inFVG {
		if (direction == "LONG" && fvgType == "BULLISH") || (direction == "SHORT" && fvgType == "BEARISH") {
			smcScore += 5
		}
	}
	score += smcScore

	// 8. Volume Profile / POC (Max 5)
	if pocDist <= 2.0 {
		score += 5
	}

	// 9. BTC Correlation (Max 5)
	if btcTrend != "" {
		if (direction == "LONG" && btcTrend == "UP") || (direction == "SHORT" && btcTrend == "DOWN") {
			score += 5
		} else {
			// Penalty for fighting BTC
			score -= 10
		}
	}

	// Penalties
	if volRatio < 0.8 {
		score -= 10 // Low volume penalty
	}

	// Boundary Check
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
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
