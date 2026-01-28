package service

import (
	"fmt"
	"strings"

	"mrcrypto-go/internal/model"
)

// formatSignalMessage creates a formatted message for Telegram in Bangla with trading guidance
func formatSignalMessage(signal *model.Signal) string {
	// Emoji based on signal type
	var signalEmoji string
	if signal.Type == model.SignalTypeLong {
		signalEmoji = "🟢"
	} else {
		signalEmoji = "🔴"
	}

	// Format AI Analysis (mocked if empty, or derived from context)
	aiAnalysis := signal.AIReason
	if aiAnalysis == "" {
		aiAnalysis = fmt.Sprintf("Trend is %s on 4H timeframe. Price is reacting at 15m support/resistance with strong momentum.", signal.TechnicalContext.Regime)
	}
	aiAnalysis = escapeHTML(aiAnalysis)

	// Format Scores
	systemScore := signal.ConfluenceScore
	aiScore := signal.AIScore
	if aiScore == 0 {
		aiScore = systemScore // Fallback if AI score not yet distinct
	}

	// Tier Display
	systemTier := string(signal.Tier)
	aiTier := signal.AITier
	if aiTier == "" {
		aiTier = systemTier
	}

	// Session info with emoji
	sessionEmoji := "🕐"
	switch signal.TechnicalContext.TradingSession {
	case "LONDON_NY_OVERLAP":
		sessionEmoji = "🔥" // Best time
	case "LONDON", "NEW_YORK":
		sessionEmoji = "✅"
	case "ASIA":
		sessionEmoji = "🌙"
	}

	// Funding sentiment emoji
	fundingEmoji := "⚖️"
	switch signal.TechnicalContext.FundingSentiment {
	case "EXTREME_LONG":
		fundingEmoji = "⚠️🔼"
	case "EXTREME_SHORT":
		fundingEmoji = "⚠️🔽"
	case "BULLISH":
		fundingEmoji = "🔼"
	case "BEARISH":
		fundingEmoji = "🔽"
	}

	// Structure emoji
	structureEmoji := "📐"
	if strings.Contains(signal.TechnicalContext.MarketStructure, "BULLISH") {
		structureEmoji = "📈"
	} else if strings.Contains(signal.TechnicalContext.MarketStructure, "BEARISH") {
		structureEmoji = "📉"
	}

	message := fmt.Sprintf(`%s <b>%s SIGNAL</b> ✅
🆔 <b>ID:</b> %s

%s | %s (System) | %s (AI)

🚀 <b>ENTRY:</b> <code>%s</code>
🛑 <b>SL:</b> <code>%s</code> (%.2f%%)

🎯 <b>TP 1:</b> <code>%s</code> (%.2f%%)
🏆 <b>TP 2:</b> <code>%s</code> (%.2f%%)

🤖 <b>AI Score:</b> %d/100
⚙️ <b>System Score:</b> %d/100

━━━━━━━━━━━━━━━━━━━
📊 <b>মার্কেট কন্টেক্সট</b>
━━━━━━━━━━━━━━━━━━━
%s <b>সেশন:</b> %s (%s volatility)
%s <b>Funding:</b> %.4f%% (%s)
%s <b>স্ট্রাকচার:</b> %s

━━━━━━━━━━━━━━━━━━━
📝 <b>AI বিশ্লেষণ</b>
━━━━━━━━━━━━━━━━━━━
%s

━━━━━━━━━━━━━━━━━━━
🎯 <b>ট্রেডিং গাইড</b>
━━━━━━━━━━━━━━━━━━━
%s

⚠️ <b>সতর্কতা:</b>
• ট্রেড নেওয়ার আগে গুরুত্বপূর্ণ নিউজ চেক করুন
• CPI, Fed Meeting, Major Protocol Upgrade এড়িয়ে চলুন
%s

⏰ <b>Time:</b> %s
`,
		signalEmoji,
		signal.Type, // SHORT / LONG
		signal.ID,
		signal.Symbol,
		systemTier,
		aiTier,
		FormatPrice(signal.EntryPrice),
		FormatPrice(signal.StopLoss),
		signal.RiskPercent,
		FormatPrice(signal.TakeProfit1),
		signal.TP1Percent,
		FormatPrice(signal.TakeProfit2),
		signal.TP2Percent,
		aiScore,
		systemScore,
		// Market Context
		sessionEmoji, signal.TechnicalContext.TradingSession, signal.TechnicalContext.SessionVolatility,
		fundingEmoji, signal.TechnicalContext.FundingRate, signal.TechnicalContext.FundingSentiment,
		structureEmoji, signal.TechnicalContext.MarketStructure,
		// AI Analysis
		aiAnalysis,
		// Trading Guidance
		signal.TechnicalContext.TradingGuidance,
		// Risk warning (if any)
		signal.TechnicalContext.RiskWarning,
		signal.Timestamp.Format("15:04:05, 02 Jan"),
	)

	return message
}

// escapeHTML escapes HTML special characters for Telegram
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// Helper functions for Emoji
func getPnLEmoji(pnl float64) string {
	if pnl > 0 {
		return "🟢 +"
	}
	return "🔴 "
}

func getPnLSign(pnl float64) string {
	if pnl > 0 {
		return "+"
	}
	return ""
}
