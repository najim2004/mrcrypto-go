package service

import (
	"fmt"
	"time"

	"my-tool-go/internal/indicator"
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

// EvaluateSymbol analyzes a symbol and generates a signal if conditions are met
func (s *StrategyService) EvaluateSymbol(symbol string) (*model.Signal, error) {
	// Fetch klines for multiple timeframes
	klines4h, err := s.binance.GetKlines(symbol, "4h", 100)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch 4h klines: %w", err)
	}

	klines1h, err := s.binance.GetKlines(symbol, "1h", 100)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch 1h klines: %w", err)
	}

	klines15m, err := s.binance.GetKlines(symbol, "15m", 100)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch 15m klines: %w", err)
	}

	klines5m, err := s.binance.GetKlines(symbol, "5m", 100)
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

	// Calculate indicators
	rsi4h := indicator.GetLastRSI(closes4h, 14)
	rsi1h := indicator.GetLastRSI(closes1h, 14)
	rsi15m := indicator.GetLastRSI(closes15m, 14)
	rsi5m := indicator.GetLastRSI(closes5m, 14)

	adx4h := indicator.GetLastADX(highs4h, lows4h, closes4h, 14)
	adx1h := indicator.GetLastADX(highs1h, lows1h, closes1h, 14)
	adx15m := indicator.GetLastADX(highs15m, lows15m, closes15m, 14)

	// Validate that we have valid indicator values
	if rsi4h == 0 || rsi1h == 0 || adx4h == 0 || adx1h == 0 {
		return nil, nil // Insufficient data for indicators
	}

	vwap := indicator.GetLastVWAP(highs5m, lows5m, closes5m, volumes5m)
	macd, macdSignal, histogram := indicator.GetLastMACD(closes5m, 12, 26, 9)

	// Calculate volume metrics
	avgVol := calculateAverage(volumes5m)
	currentVol := volumes5m[len(volumes5m)-1]

	// Calculate order flow delta (simplified)
	orderFlowDelta := calculateOrderFlowDelta(klines5m)

	// Get current price and EMA50
	currentPrice := closes4h[len(closes4h)-1]
	ema50 := indicator.CalculateEMA(closes4h, 50)

	// Validate EMA50 calculation
	if len(ema50) == 0 {
		return nil, nil // Not enough data for EMA50
	}

	ema50Value := ema50[len(ema50)-1]

	// Detect market regime
	regime := detectRegime(adx4h, currentPrice, ema50Value)

	// Filter out choppy markets
	if regime == model.RegimeChoppy {
		return nil, nil // No signal for choppy markets
	}

	// Technical context
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
	}

	// Try PREMIUM tier first
	tradingSignal := checkPremiumTier(symbol, currentPrice, regime, techContext, currentVol, avgVol, orderFlowDelta)
	if tradingSignal != nil {
		return tradingSignal, nil
	}

	// Try STANDARD tier
	tradingSignal = checkStandardTier(symbol, currentPrice, regime, techContext, currentVol, avgVol)
	if tradingSignal != nil {
		return tradingSignal, nil
	}

	return nil, nil // No signal generated
}

// detectRegime determines market regime based on ADX and price vs EMA
func detectRegime(adx, price, ema50 float64) model.MarketRegime {
	if adx < 20 {
		return model.RegimeChoppy
	}

	if price > ema50 {
		return model.RegimeTrendingUp
	} else if price < ema50 {
		return model.RegimeTrendingDown
	}

	return model.RegimeRanging
}

// checkPremiumTier checks if conditions meet PREMIUM tier criteria
func checkPremiumTier(symbol string, currentPrice float64, regime model.MarketRegime,
	techContext model.TechnicalContext, currentVol, avgVol, orderFlowDelta float64) *model.Signal {

	// PREMIUM requirements: ADX > 25, stricter RSI, 2x volume, positive order flow
	if techContext.ADX1h < 25 {
		return nil
	}

	if currentVol < avgVol*2.0 {
		return nil
	}

	// Check for LONG signal
	if regime == model.RegimeTrendingUp &&
		techContext.RSI1h > 50 && techContext.RSI1h < 65 &&
		techContext.RSI5m > 40 && techContext.RSI5m < 70 &&
		orderFlowDelta > 0 &&
		techContext.Histogram > 0 {

		return createSignal(symbol, model.SignalTypeLong, model.TierPremium,
			currentPrice, regime, techContext)
	}

	// Check for SHORT signal
	if regime == model.RegimeTrendingDown &&
		techContext.RSI1h > 35 && techContext.RSI1h < 50 &&
		techContext.RSI5m > 30 && techContext.RSI5m < 60 &&
		orderFlowDelta < 0 &&
		techContext.Histogram < 0 {

		return createSignal(symbol, model.SignalTypeShort, model.TierPremium,
			currentPrice, regime, techContext)
	}

	return nil
}

// checkStandardTier checks if conditions meet STANDARD tier criteria
func checkStandardTier(symbol string, currentPrice float64, regime model.MarketRegime,
	techContext model.TechnicalContext, currentVol, avgVol float64) *model.Signal {

	// STANDARD requirements: ADX > 20, wider RSI, 1x volume
	if techContext.ADX1h < 20 {
		return nil
	}

	if currentVol < avgVol {
		return nil
	}

	// Check for LONG signal
	if regime == model.RegimeTrendingUp &&
		techContext.RSI1h > 40 && techContext.RSI1h < 70 &&
		techContext.RSI5m > 35 && techContext.RSI5m < 75 &&
		techContext.Histogram > 0 {

		return createSignal(symbol, model.SignalTypeLong, model.TierStandard,
			currentPrice, regime, techContext)
	}

	// Check for SHORT signal
	if regime == model.RegimeTrendingDown &&
		techContext.RSI1h > 30 && techContext.RSI1h < 60 &&
		techContext.RSI5m > 25 && techContext.RSI5m < 65 &&
		techContext.Histogram < 0 {

		return createSignal(symbol, model.SignalTypeShort, model.TierStandard,
			currentPrice, regime, techContext)
	}

	return nil
}

// createSignal creates a signal with calculated stop loss and take profit
func createSignal(symbol string, signalType model.SignalType, tier model.SignalTier,
	entryPrice float64, regime model.MarketRegime, techContext model.TechnicalContext) *model.Signal {

	var stopLoss, takeProfit float64

	if signalType == model.SignalTypeLong {
		stopLoss = entryPrice * 0.98   // 2% stop loss
		takeProfit = entryPrice * 1.06 // 6% take profit (3:1 R/R)
	} else {
		stopLoss = entryPrice * 1.02   // 2% stop loss
		takeProfit = entryPrice * 0.94 // 6% take profit (3:1 R/R)
	}

	return &model.Signal{
		Symbol:           symbol,
		Type:             signalType,
		Tier:             tier,
		EntryPrice:       entryPrice,
		StopLoss:         stopLoss,
		TakeProfit:       takeProfit,
		Regime:           string(regime),
		TechnicalContext: techContext,
		Timestamp:        time.Now(),
	}
}

// Helper functions to extract data from klines
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
	// Simplified order flow: assume buying pressure when close > open
	delta := 0.0
	for i := len(klines) - 10; i < len(klines); i++ {
		if i < 0 {
			continue
		}
		k := klines[i]
		if k.Close > k.Open {
			delta += k.Volume
		} else {
			delta -= k.Volume
		}
	}
	return delta
}

// CalculateDynamicDecimals returns appropriate decimal places based on price
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

// FormatPrice formats price with dynamic decimals
func FormatPrice(price float64) string {
	decimals := CalculateDynamicDecimals(price)
	format := fmt.Sprintf("%%.%df", decimals)
	return fmt.Sprintf(format, price)
}
