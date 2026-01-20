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
}

// Signal represents a trading signal
type Signal struct {
	Symbol           string           `json:"symbol" bson:"symbol"`
	Type             SignalType       `json:"type" bson:"type"`
	Tier             SignalTier       `json:"tier" bson:"tier"`
	EntryPrice       float64          `json:"entry_price" bson:"entry_price"`
	StopLoss         float64          `json:"stop_loss" bson:"stop_loss"`
	TakeProfit       float64          `json:"take_profit" bson:"take_profit"`
	Regime           string           `json:"regime" bson:"regime"`
	TechnicalContext TechnicalContext `json:"technical_context" bson:"technical_context"`
	AIScore          int              `json:"ai_score" bson:"ai_score"`
	AIReason         string           `json:"ai_reason" bson:"ai_reason"`
	Status           string           `json:"status" bson:"status"`                       // ACTIVE, CLOSED
	CloseReason      string           `json:"close_reason,omitempty" bson:"close_reason"` // TP_HIT, SL_HIT, MANUAL, REVERSED
	ClosedAt         *time.Time       `json:"closed_at,omitempty" bson:"closed_at"`
	PnL              float64          `json:"pnl,omitempty" bson:"pnl"` // Profit/Loss percentage
	Timestamp        time.Time        `json:"timestamp" bson:"timestamp"`
	CreatedAt        time.Time        `json:"created_at" bson:"created_at"`
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
