package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"mrcrypto-go/internal/config"
	"mrcrypto-go/internal/model"

	"google.golang.org/genai"
)

type AIService struct {
	clients []*genai.Client // Use slice of clients
	ctx     context.Context
}

func NewAIService() *AIService {
	ctx := context.Background()
	var clients []*genai.Client

	keys := config.AppConfig.GeminiAPIKeys
	if len(keys) == 0 {
		log.Printf("⚠️  No Gemini API keys found")
	}

	for _, key := range keys {
		key = strings.TrimSpace(key) // Clean up whitespace
		if key == "" {
			continue
		}
		client, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey: key,
		})
		if err != nil {
			log.Printf("⚠️  Failed to create Gemini client for key ending in ...%s: %v", key[len(key)-4:], err)
			continue
		}
		clients = append(clients, client)
	}

	if len(clients) > 0 {
		log.Printf("✅ Initialized %d Gemini clients for rotation", len(clients))
	} else {
		log.Printf("❌ Failed to initialize any Gemini clients")
	}

	return &AIService{
		clients: clients,
		ctx:     ctx,
	}
}

// AIValidationResult contains the AI's assessment
type AIValidationResult struct {
	Score  int    `json:"score"`
	Reason string `json:"reason"`
}

// ValidateSignal sends the signal to Gemini AI for validation with fallback models
func (s *AIService) ValidateSignal(signal *model.Signal) (int, string, error) {
	if len(s.clients) == 0 {
		return 0, "", fmt.Errorf("no gemini clients initialized")
	}

	// Calculate volume ratio safely
	volRatio := 0.0
	if signal.TechnicalContext.AvgVol > 0 {
		volRatio = signal.TechnicalContext.CurrentVol / signal.TechnicalContext.AvgVol
	}

	prompt := fmt.Sprintf(`You are a Professional Crypto Trading Analyst and Hedge Fund Manager with 15+ years of experience. Your task is to perform a rigorous analysis of the following trading signal to ensure maximum accuracy.

Remember: A wrong signal leads to significant financial loss. Only provide high scores if there is strong confluence.

╔══════════════════════════════════════════════════════════════╗
                    🔔 SIGNAL DETAILS
╚══════════════════════════════════════════════════════════════╝

📌 Pair: %s (Trading Symbol)
📌 Direction: %s (Trade type: BUY/SELL)
📌 Tier: %s (Signal quality classification)
📌 Market Regime: %s (Current market condition: Trending/Ranging/Choppy)

💰 RISK MANAGEMENT:
🎯 Entry: %s (Current Market Price)
🛑 Stop Loss: %s (Exit if price hits this level to limit loss, Risk: %.2f%%)
🏆 Take Profit: %s (Target exit price for profit, Reward: %.2f%%)
📊 R:R Ratio: %.2f (Reward-to-Risk ratio, higher is better)
🎲 Break-Even Win Rate: %.2f%% (Required win rate to stay neutral at this R:R)
💼 Position Size (Kelly): %.2f%% (Recommended allocation based on probability and edge)

📊 TECHNICAL INDICATORS:
• RSI (4H/1H/15M/5M): %.1f / %.1f / %.1f / %.1f (Relative Strength Index: >70 Overbought, <30 Oversold)
• ADX (4H/1H/15M): %.1f / %.1f / %.1f (Average Directional Index: >25 Strong Trend, <20 Weak/Ranging)
• MACD Histogram: %.6f (Moving Average Convergence Divergence: >0 Bullish Momentum, <0 Bearish Momentum)
• Volume Ratio: %.2fx (Current volume vs Average volume: >1.5x indicates high participation)
• Order Flow Delta: %.2f (Net buy/sell volume: Positive is Bullish, Negative is Bearish)
• VWAP: %s (Volume Weighted Average Price: Price above/below indicates sentiment)

🎯 KEY LEVELS (Support & Resistance):
• Pivot Points: %s, %s, %s, PP: %s, %s, %s, %s (Standard daily support/resistance levels)
• Nearest Pivot: %s (The level closest to the current price)
• Fibonacci (38.2/50/61.8): %s, %s, %s (Major retracement levels)
• Nearest Fib: %s (The Fibonacci level closest to the current price)
• Distance from Level: %.2f%% (How close we are to a key structural level)

🏛️ SMC (Smart Money Concepts) & MARKET STRUCTURE:
• BTC Correlation: %s (Current trend of Bitcoin: UP/DOWN)
• Order Block (OB): %s (Institutional interest zone: BULLISH/BEARISH/NONE)
• Fair Value Gap (FVG): %s (Inefficiency zone: BULLISH/BEARISH/NONE)
• Volume POC: %s (Point of Control: Price with highest volume in current range)
• POC Distance: %.2f%% (How far price is from high volume node)

📐 SYSTEM PROBABILITY:
• Confluence Score: %d/100 (Internal system score based on matching indicators)
• System Confidence: %.1f%% (Mathematical probability of success calculated from confluence)

╔══════════════════════════════════════════════════════════════╗
                🔍 ANALYSIS GUIDELINES
╚══════════════════════════════════════════════════════════════╝

As an expert trader, you must deeply verify the following:

1. **Multi-Timeframe Alignment:** Are 4H and 1H trends aligned? Is RSI away from reversal zones?
2. **Volume & Momentum:** Is Volume Ratio > 1.5x? Do MACD and Order Flow support the direction?
3. **Key Level Rejection/Breakout:** Is price near a major Pivot or Fibonacci level? What is the likelihood of rejection/breakout?
4. **Risk/Reward:** Is R:R > 2.0? Is SL at a logical structural level?
5. **Market Regime:** Is this trade appropriate for the current regime (Trending/Ranging)?
6. **SMC Confluence:** Is price reacting to an Order Block or filling a Gap? Is it aligned with BTC?

╔══════════════════════════════════════════════════════════════╗
                    📝 RESPONSE FORMAT
╚══════════════════════════════════════════════════════════════╝

Respond ONLY in the following JSON format.
CRITICAL: The "reason" field MUST be written in BENGALI (Bangla).

{"score": <0-100>, "reason": "<detailed professional analysis in BENGALI>"}

Scoring Rules (Be extremely critical):
• 95-100: Unique setup (All indicators and HTF aligned)
• 85-94: High probability (Minor gaps allowed)
• 75-84: Good setup (Some risk factors present)
• 60-74: Average (Trade with caution)
• Below 60: Direct Reject (Avoid)`,
		signal.Symbol,
		signal.Type,
		signal.Tier,
		signal.Regime,
		FormatPrice(signal.EntryPrice),
		FormatPrice(signal.StopLoss),
		signal.RiskPercent,
		FormatPrice(signal.TakeProfit),
		signal.RewardPercent,
		signal.RiskRewardRatio,
		signal.BreakEvenWinRate,
		signal.RecommendedSize,
		signal.TechnicalContext.RSI4h,
		signal.TechnicalContext.RSI1h,
		signal.TechnicalContext.RSI15m,
		signal.TechnicalContext.RSI5m,
		signal.TechnicalContext.ADX4h,
		signal.TechnicalContext.ADX1h,
		signal.TechnicalContext.ADX15m,
		signal.TechnicalContext.Histogram,
		volRatio,
		signal.TechnicalContext.OrderFlowDelta,
		FormatPrice(signal.TechnicalContext.VWAP),
		FormatPrice(signal.TechnicalContext.PivotR3),
		FormatPrice(signal.TechnicalContext.PivotR2),
		FormatPrice(signal.TechnicalContext.PivotR1),
		FormatPrice(signal.TechnicalContext.PivotPoint),
		FormatPrice(signal.TechnicalContext.PivotS1),
		FormatPrice(signal.TechnicalContext.PivotS2),
		FormatPrice(signal.TechnicalContext.PivotS3),
		signal.TechnicalContext.NearestPivot,
		FormatPrice(signal.TechnicalContext.Fib382),
		FormatPrice(signal.TechnicalContext.Fib500),
		FormatPrice(signal.TechnicalContext.Fib618),
		signal.TechnicalContext.NearestFib,
		signal.NearestLevelDist,
		signal.TechnicalContext.BTCCorrelation,
		signal.TechnicalContext.OBType,
		signal.TechnicalContext.FVGType,
		FormatPrice(signal.TechnicalContext.POC),
		signal.TechnicalContext.POCDistance,
		signal.ConfluenceScore,
		signal.ConfidenceScore*100,
	)

	// List of models to try in order (fallback)
	models := []string{
		"gemini-3-flash",
		"gemini-2.5-flash",
		"gemini-2.5-flash-lite",
		"gemini-2.5-flash-tts",
	}

	var lastError error

	// Try each model until one succeeds
	for _, modelName := range models {
		// Try each client (key) for rotation
		for cIdx, client := range s.clients {
			result, err := client.Models.GenerateContent(
				s.ctx,
				modelName,
				genai.Text(prompt),
				nil,
			)

			if err != nil {
				lastError = err
				log.Printf("⚠️  %s - Model %s (Client %d) failed: %v", signal.Symbol, modelName, cIdx+1, err)

				// If quota exceeded or key expired, try next client immediately
				errStr := err.Error()
				if strings.Contains(errStr, "429") ||
					strings.Contains(errStr, "quota") ||
					strings.Contains(errStr, "expired") ||
					strings.Contains(errStr, "API_KEY_INVALID") ||
					strings.Contains(errStr, "INVALID_ARGUMENT") {
					log.Printf("🔄 Switching to next client due to error: %v", err)
					continue
				}

				// If other error, maybe try next model
				break // Break inner client loop to try next model
			}

			// Success! Parse response
			responseText := result.Text()

			var aiResult AIValidationResult
			if err := json.Unmarshal([]byte(responseText), &aiResult); err != nil {
				log.Printf("⚠️  Failed to parse AI response for %s (model: %s): %v", signal.Symbol, modelName, err)
				// Don't error out on parse error, maybe try next model?
				// For now, return default
				return 50, responseText, nil
			}

			log.Printf("✅ [AI] %s - Validated! Model: %s, Client: %d, Score: %d/100", signal.Symbol, modelName, cIdx+1, aiResult.Score)
			return aiResult.Score, aiResult.Reason, nil
		}
	}

	// Should never reach here, but just in case
	return 0, "", fmt.Errorf("unexpected error: %w", lastError)
}

// BatchValidateSignals validates multiple signals in a single AI call (OPTIMIZED)
func (s *AIService) BatchValidateSignals(signals []*model.Signal) ([]AIValidationResult, error) {
	if len(s.clients) == 0 {
		return nil, fmt.Errorf("no gemini clients initialized")
	}

	if len(signals) == 0 {
		return []AIValidationResult{}, nil
	}

	// Build batch prompt with comprehensive data
	prompt := `You are a Tier-1 Crypto Trading Floor Manager with 15+ years of experience. Analyze these potential signals with extreme scrutiny. 
Discard any setups that lack proper technical alignment or have poor risk management.

STRICT CRITERIA:
1. Multi-TF Alignment: 4H and 1H trends MUST align for high scores.
2. Volume Confirmation: Real breakouts need >= 1.5x average volume.
3. Key Level Integrity: Respect major Pivot and Fibonacci levels.
4. Risk Management: If R:R < 2.0, the signal is INVALID.

BENGALI ONLY REASONING:
Explain your decision like a senior mentor teaching a junior trader. You MUST write the "reason" in BENGALI (Bangla).

RESPONSE FORMAT:
Respond only with a JSON array:
[
  {"signal": 1, "score": <0-100>, "reason": "<Senior Analyst explanation in Bengali>"},
  {"signal": 2, "score": <0-100>, "reason": "<Senior Analyst explanation in Bengali>"}
]

SIGNALS TO SCRUTINIZE:
`

	for idx, signal := range signals {
		// Calculate volume ratio safely
		volRatio := 0.0
		if signal.TechnicalContext.AvgVol > 0 {
			volRatio = signal.TechnicalContext.CurrentVol / signal.TechnicalContext.AvgVol
		}

		prompt += fmt.Sprintf(`
━━━━━━━━━━ SIGNAL %d ━━━━━━━━━━
📌 Symbol: %s | Direction: %s | Tier: %s | Regime: %s (Market Cycle)

💰 RISK & REWARD (Position Management):
- Entry Price: %s (Current Market Level)
- Stop Loss: %s (Exit if hit, Risk: %.2f%%)
- Take Profit: %s (Target exit, Reward: %.2f%%)
- R:R Ratio: %.2f (Reward-to-Risk)
- Break-Even Win Rate: %.2f%% (Statistically required)
- Rec. Position Size: %.2f%% (Kelly Criterion allocation)

📊 TECHNICAL INDICATORS (Momentum & Trend):
- RSI (4H/1H/15M/5M): %.1f/%.1f/%.1f/%.1f (Strength: >70 Overbought, <30 Oversold)
- ADX (4H/1H/15M): %.1f/%.1f/%.1f (Trend Intensity: >25 Strong)
- MACD Histogram: %.6f (Momentum Direction)
- Volume Ratio: %.2fx (Relative volume vs Average)
- Order Flow: %.2f (Net buying/selling pressure)

🎯 LEVELS & MARKET STRUCTURE (SMC):
- Pivot Levels: Pivot: %s | S1: %s | R1: %s (Support/Resistance)
- Nearest Pivot: %s (Price distance: %.2f%%)
- Fibonacci: 50.0%%: %s | 61.8%%: %s | Nearest: %s (Retracement zones)
- BTC Trend: %s (Overall market correlation)
- Smart Money: OB: %s (Order Block) | FVG: %s (Fair Value Gap)
- Volume Profile: POC: %s (Dist: %.2f%%) (Point of Control)
- System Confluence: %d/100 | Confidence: %.1f%% (Internal probability)

SENIOR ANALYST DECISION (Rigorous Bengali Analysis):
`,
			idx+1,
			signal.Symbol,
			signal.Type,
			signal.Tier,
			signal.Regime,
			FormatPrice(signal.EntryPrice),
			FormatPrice(signal.StopLoss),
			signal.RiskPercent,
			FormatPrice(signal.TakeProfit),
			signal.RewardPercent,
			signal.RiskRewardRatio,
			signal.BreakEvenWinRate,
			signal.RecommendedSize,
			signal.TechnicalContext.RSI4h,
			signal.TechnicalContext.RSI1h,
			signal.TechnicalContext.RSI15m,
			signal.TechnicalContext.RSI5m,
			signal.TechnicalContext.ADX4h,
			signal.TechnicalContext.ADX1h,
			signal.TechnicalContext.ADX15m,
			signal.TechnicalContext.Histogram,
			volRatio,
			signal.TechnicalContext.OrderFlowDelta,
			FormatPrice(signal.TechnicalContext.PivotPoint),
			FormatPrice(signal.TechnicalContext.PivotS1),
			FormatPrice(signal.TechnicalContext.PivotR1),
			signal.TechnicalContext.NearestPivot,
			signal.NearestLevelDist,
			FormatPrice(signal.TechnicalContext.Fib500),
			FormatPrice(signal.TechnicalContext.Fib618),
			signal.TechnicalContext.NearestFib,
			signal.TechnicalContext.BTCCorrelation,
			signal.TechnicalContext.OBType,
			signal.TechnicalContext.FVGType,
			FormatPrice(signal.TechnicalContext.POC),
			signal.TechnicalContext.POCDistance,
			signal.ConfluenceScore,
			signal.ConfidenceScore*100,
		)
	}

	// List of models to try in order (fallback)
	models := []string{
		"gemini-3-pro-preview",
		"gemini-3-flash-preview",
		"gemini-2.5-flash",
		"gemini-2.5-flash-lite",
		"gemini-2.5-pro",
	}

	log.Printf("🤖 [AI Batch] Validating %d signals (trying %d models)...", len(signals), len(models))

	var lastError error

	// Try each model until one succeeds
	for i, modelName := range models {
		log.Printf("⏳ [AI Batch] Trying model: %s (%d/%d)...", modelName, i+1, len(models))

		// Try each client (key) for rotation
		for cIdx, client := range s.clients {
			result, err := client.Models.GenerateContent(
				s.ctx,
				modelName,
				genai.Text(prompt),
				nil,
			)

			if err != nil {
				lastError = err
				log.Printf("⚠️  Batch validation - Model %s (Client %d) failed: %v", modelName, cIdx+1, err)

				// If quota exceeded or key expired, try next client
				errStr := err.Error()
				if strings.Contains(errStr, "429") ||
					strings.Contains(errStr, "quota") ||
					strings.Contains(errStr, "expired") ||
					strings.Contains(errStr, "API_KEY_INVALID") ||
					strings.Contains(errStr, "INVALID_ARGUMENT") {
					log.Printf("🔄 Switching to next client due to error: %v", err)
					continue
				}
				break // Try next model on other errors
			}

			// Success! Parse response
			responseText := result.Text()

			// Extract JSON from markdown code blocks if present
			jsonText := extractJSONFromMarkdown(responseText)

			// Try to parse as JSON array
			var results []struct {
				SignalNum int    `json:"signal"`
				Score     int    `json:"score"`
				Reason    string `json:"reason"`
			}

			if err := json.Unmarshal([]byte(jsonText), &results); err != nil {
				log.Printf("⚠️  Failed to parse batch AI response (model: %s): %v", modelName, err)
				log.Printf("Response preview: %s", jsonText[:min(len(jsonText), 200)])
				// Return default scores
				defaultResults := make([]AIValidationResult, len(signals))
				for idx := range defaultResults {
					defaultResults[idx] = AIValidationResult{Score: 50, Reason: "AI parse error"}
				}
				return defaultResults, nil
			}

			// Convert to AIValidationResult
			validationResults := make([]AIValidationResult, len(signals))
			for idx, res := range results {
				if idx < len(validationResults) {
					validationResults[idx] = AIValidationResult{
						Score:  res.Score,
						Reason: res.Reason,
					}
				}
			}

			log.Printf("✅ [AI Batch] Successfully validated %d signals with model: %s (Client %d)", len(signals), modelName, cIdx+1)
			return validationResults, nil
		}
	}

	return nil, fmt.Errorf("unexpected error: %w", lastError)
}

// extractJSONFromMarkdown removes markdown code block markers
func extractJSONFromMarkdown(text string) string {
	// Check if wrapped in markdown code blocks
	if len(text) > 7 && text[:3] == "```" {
		// Find the end of the opening code fence
		start := 0
		for i := 3; i < len(text); i++ {
			if text[i] == '\n' {
				start = i + 1
				break
			}
		}

		// Find the closing code fence
		end := len(text)
		for i := len(text) - 1; i >= start+3; i-- {
			if i >= 2 && text[i-2:i+1] == "```" {
				end = i - 2
				break
			}
		}

		return text[start:end]
	}

	return text
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
