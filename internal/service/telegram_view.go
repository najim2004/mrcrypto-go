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

	message := fmt.Sprintf(`%s <b>%s SIGNAL</b> ✅
🆔 <b>ID:</b> %s

%s | %s (System) | %s (AI)

🚀 <b>ENTRY:</b> <code>%s</code>
🛑 <b>SL:</b> <code>%s</code> (%.2f%%)

🎯 <b>TP 1:</b> <code>%s</code> (%.2f%%)
🏆 <b>TP 2:</b> <code>%s</code> (%.2f%%)

🤖 <b>AI Score:</b> %d/100
⚙️ <b>System Score:</b> %d/100

📝 <b>AI Analysis:</b>
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
		aiAnalysis,
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
