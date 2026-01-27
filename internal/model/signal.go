package model

import "time"

// SignalType represents the direction of the trade signal
type SignalType string

const (
	SignalTypeLong  SignalType = "LONG"
	SignalTypeShort SignalType = "SHORT"
)

// SignalTier represents the quality level of the signal
type SignalTier string

const (
	TierPremium  SignalTier = "PREMIUM"
	TierStandard SignalTier = "STANDARD"
)

// MarketRegime represents the current market condition
type MarketRegime string

const (
	RegimeTrendingUp   MarketRegime = "TRENDING_UP"
	RegimeTrendingDown MarketRegime = "TRENDING_DOWN"
	RegimeRanging      MarketRegime = "RANGING"
	RegimeChoppy       MarketRegime = "CHOPPY"
)

// TechnicalContext holds all technical indicator values
type TechnicalContext struct {
	RSI4h  float64 `json:"rsi_4h"`
	RSI1h  float64 `json:"rsi_1h"`
	RSI15m float64 `json:"rsi_15m"`
	RSI5m  float64 `json:"rsi_5m"`

	ADX4h  float64 `json:"adx_4h"`
	ADX1h  float64 `json:"adx_1h"`
	ADX15m float64 `json:"adx_15m"`

	VWAP       float64 `json:"vwap"`
	CurrentVol float64 `json:"current_vol"`
	AvgVol     float64 `json:"avg_vol"`

	MACD      float64 `json:"macd"`
	Signal    float64 `json:"signal"`
	Histogram float64 `json:"histogram"`

	OrderFlowDelta float64 `json:"order_flow_delta"`
	Regime         string  `json:"regime"`

	// Pivot Points (Daily)
	PivotPoint   float64 `json:"pivot_point"`
	PivotR1      float64 `json:"pivot_r1"`
	PivotR2      float64 `json:"pivot_r2"`
	PivotR3      float64 `json:"pivot_r3"`
	PivotS1      float64 `json:"pivot_s1"`
	PivotS2      float64 `json:"pivot_s2"`
	PivotS3      float64 `json:"pivot_s3"`
	NearestPivot string  `json:"nearest_pivot"`

	// Fibonacci Levels (4H)
	Fib236     float64 `json:"fib_236"`
	Fib382     float64 `json:"fib_382"`
	Fib500     float64 `json:"fib_500"`
	Fib618     float64 `json:"fib_618"`
	Fib786     float64 `json:"fib_786"`
	NearestFib string  `json:"nearest_fib"`

	// New Improvements
	BTCCorrelation string  `json:"btc_correlation"` // UP, DOWN
	FVGType        string  `json:"fvg_type"`        // BULLISH, BEARISH
	OBType         string  `json:"ob_type"`         // BULLISH, BEARISH
	POC            float64 `json:"poc"`
	POCDistance    float64 `json:"poc_distance"`

	// Advanced Analysis
	CandlestickPattern string  `json:"candlestick_pattern"` // Hammer, Doji, etc.
	Divergence         string  `json:"divergence"`          // Bullish, Bearish
	ATR                float64 `json:"atr"`                 // Volatility
	StochRSI           float64 `json:"stoch_rsi"`           // Momentum (K)
	LiquiditySweep     string  `json:"liquidity_sweep"`     // Swing Failure Pattern
	TrendState         string  `json:"trend_state"`         // Golden Cross, Death Cross
}

// Signal represents a trading signal
type Signal struct {
	Symbol           string           `json:"symbol" bson:"symbol"`
	Type             SignalType       `json:"type" bson:"type"`
	Tier             SignalTier       `json:"tier" bson:"tier"`
	EntryPrice       float64          `json:"entry_price" bson:"entry_price"`
	StopLoss         float64          `json:"stop_loss" bson:"stop_loss"`
	TakeProfit       float64          `json:"take_profit" bson:"take_profit"`     // Legacy - same as TP2
	TakeProfit1      float64          `json:"take_profit_1" bson:"take_profit_1"` // TP1: 50% position close
	TakeProfit2      float64          `json:"take_profit_2" bson:"take_profit_2"` // TP2: remaining 50% close
	RiskRewardRatio  float64          `json:"risk_reward_ratio" bson:"risk_reward_ratio"`
	RecommendedSize  float64          `json:"recommended_size" bson:"recommended_size"` // Position size as % of account
	Regime           string           `json:"regime" bson:"regime"`
	TechnicalContext TechnicalContext `json:"technical_context" bson:"technical_context"`
	AIScore          int              `json:"ai_score" bson:"ai_score"`
	AIConfidence     int              `json:"ai_confidence" bson:"ai_confidence"` // 0-100 Confidence
	AITier           string           `json:"ai_tier" bson:"ai_tier"`             // Standard, Premium, or Reject
	AIReason         string           `json:"ai_reason" bson:"ai_reason"`

	// Identity
	ID string `json:"id" bson:"id"` // Short 5-char unique ID

	// Probability Fields
	ConfidenceScore  float64 `json:"confidence_score" bson:"confidence_score"`       // 0.0 - 1.0 probability of success
	ConfluenceScore  int     `json:"confluence_score" bson:"confluence_score"`       // 0-100 confluence points
	BreakEvenWinRate float64 `json:"break_even_win_rate" bson:"break_even_win_rate"` // Required win rate to break even
	RiskPercent      float64 `json:"risk_percent" bson:"risk_percent"`               // % distance to SL
	RewardPercent    float64 `json:"reward_percent" bson:"reward_percent"`           // % distance to TP2
	TP1Percent       float64 `json:"tp1_percent" bson:"tp1_percent"`                 // % distance to TP1
	TP2Percent       float64 `json:"tp2_percent" bson:"tp2_percent"`                 // % distance to TP2
	NearestLevelDist float64 `json:"nearest_level_dist" bson:"nearest_level_dist"`   // % distance to nearest key level

	// Monitoring State
	TPAlertSent       bool      `json:"tp_alert_sent" bson:"tp_alert_sent"`
	SLAlertSent       bool      `json:"sl_alert_sent" bson:"sl_alert_sent"`
	ReversalAlertSent bool      `json:"reversal_alert_sent" bson:"reversal_alert_sent"`
	TrailingAlertSent bool      `json:"trailing_alert_sent" bson:"trailing_alert_sent"`
	LastAlertTime     time.Time `json:"last_alert_time" bson:"last_alert_time"`

	Status      string     `json:"status" bson:"status"`                       // ACTIVE, CLOSED
	CloseReason string     `json:"close_reason,omitempty" bson:"close_reason"` // TP_HIT, SL_HIT, MANUAL, REVERSED
	ClosedAt    *time.Time `json:"closed_at,omitempty" bson:"closed_at"`
	PnL         float64    `json:"pnl,omitempty" bson:"pnl"`               // Profit/Loss percentage
	PnLAmount   float64    `json:"pnl_amount,omitempty" bson:"pnl_amount"` // PnL in USD
	Timestamp   time.Time  `json:"timestamp" bson:"timestamp"`
	CreatedAt   time.Time  `json:"created_at" bson:"created_at"`
}

// Kline represents a candlestick data point
type Kline struct {
	OpenTime  int64
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	CloseTime int64
}
