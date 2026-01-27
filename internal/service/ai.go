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
	Score      int    `json:"score"`
	Confidence int    `json:"confidence"`
	Tier       string `json:"tier"` // Standard or Premium
	Reason     string `json:"reason"`
}

// ValidateSignal sends the signal to Gemini AI for validation with fallback models
func (s *AIService) ValidateSignal(signal *model.Signal) (int, int, string, string, error) {
	if len(s.clients) == 0 {
		return 0, 0, "", "", fmt.Errorf("no gemini clients initialized")
	}

	// Calculate volume ratio safely (redundant here but kept for context if needed, otherwise remove. Helper handles it too)
	// Construct the full prompt
	prompt := fmt.Sprintf(`You are a Professional Crypto Trading Analyst and Hedge Fund Manager with 15+ years of experience. Your task is to perform a rigorous analysis of the following trading signal to ensure maximum accuracy.

Remember: A wrong signal leads to significant financial loss. Only provide high scores if there is strong confluence.

%s

╔══════════════════════════════════════════════════════════════╗
                🔍 ANALYSIS GUIDELINES
╚══════════════════════════════════════════════════════════════╝

1. **Multi-Timeframe Alignment:** Are 4H and 1H trends aligned? Does 15M support entry?
2. **Volume & Momentum:** Is Volume > Avg? Does MACD support direction?
3. **Key Level:** Is price reacting to a major level?
4. **Risk/Reward:** Is R:R > 2.0?
5. **Score & Tier:**
   - **PREMIUM (90-100):** Perfect setup. Trend aligned, Volume confirmed, Key Level test.
   - **STANDARD (70-89):** Good setup but maybe 1 factor weak (e.g., weak volume).
   - **REJECT (<70):** Bad risk/reward, choppy market, or contra-trend.

╔══════════════════════════════════════════════════════════════╗
                    📝 RESPONSE FORMAT
╚══════════════════════════════════════════════════════════════╝

Respond ONLY in the following JSON format.
CRITICAL: The "reason" field MUST be written in BENGALI (Bangla).

{"score": <0-100>, "confidence": <0-100>, "tier": "PREMIUM"|"STANDARD"|"REJECT", "reason": "<detailed analysis in BENGALI>"}
`, generateSignalPrompt(signal))

	// List of models to try in order (fallback)
	models := []string{
		"gemini-2.0-flash",
		"gemini-1.5-flash",
		"gemini-1.5-pro",
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
				continue
			}

			// Success! Parse response
			responseText := result.Text()
			jsonText := extractJSONFromMarkdown(responseText)

			var aiResult AIValidationResult
			if err := json.Unmarshal([]byte(jsonText), &aiResult); err != nil {
				log.Printf("⚠️  Failed to parse AI response for %s: %v", signal.Symbol, err)
				return 50, 0, "STANDARD", "AI Parse Error", nil
			}

			// Normalize Tier
			tier := strings.ToUpper(aiResult.Tier)
			if tier != "PREMIUM" && tier != "STANDARD" {
				// Auto-correct tier based on score if missing
				if aiResult.Score >= 90 {
					tier = "PREMIUM"
				} else if aiResult.Score >= 70 {
					tier = "STANDARD"
				} else {
					tier = "REJECT"
				}
			}

			log.Printf("✅ [AI] %s - Validated! Score: %d, Confidence: %d, Tier: %s", signal.Symbol, aiResult.Score, aiResult.Confidence, tier)
			return aiResult.Score, aiResult.Confidence, tier, aiResult.Reason, nil
		}
	}

	return 0, 0, "", "", fmt.Errorf("all AI models failed: %w", lastError)
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
Respond only with a JSON array. 
- "score": 0-100 (90+ = Premium, 70-89 = Standard, <70 = Reject)
- "confidence": 0-100 (Your confidence in this analysis)
- "tier": "PREMIUM" | "STANDARD" | "REJECT"

[
  {"signal": 1, "score": <0-100>, "confidence": <0-100>, "tier": "PREMIUM"|"STANDARD"|"REJECT", "reason": "<Senior Analyst explanation in Bengali>"},
  {"signal": 2, "score": <0-100>, "confidence": <0-100>, "tier": "PREMIUM"|"STANDARD"|"REJECT", "reason": "<Senior Analyst explanation in Bengali>"}
]

SIGNALS TO SCRUTINIZE:
`

	for _, signal := range signals {
		// Append each signal's details
		prompt += generateSignalPrompt(signal)
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
				SignalNum  int    `json:"signal"`
				Score      int    `json:"score"`
				Confidence int    `json:"confidence"`
				Tier       string `json:"tier"`
				Reason     string `json:"reason"`
			}

			if err := json.Unmarshal([]byte(jsonText), &results); err != nil {
				log.Printf("⚠️  Failed to parse batch AI response (model: %s): %v", modelName, err)
				log.Printf("Response preview: %s", jsonText[:min(len(jsonText), 200)])
				// Return default scores
				defaultResults := make([]AIValidationResult, len(signals))
				for idx := range defaultResults {
					defaultResults[idx] = AIValidationResult{Score: 50, Tier: "STANDARD", Reason: "AI parse error"}
				}
				return defaultResults, nil
			}

			// Convert to AIValidationResult
			validationResults := make([]AIValidationResult, len(signals))
			for idx, res := range results {
				if idx < len(validationResults) {
					tier := strings.ToUpper(res.Tier)

					// Auto-correct tier based on score if missing or invalid
					if tier != "PREMIUM" && tier != "STANDARD" && tier != "REJECT" {
						if res.Score >= 90 {
							tier = "PREMIUM"
						} else if res.Score >= 70 {
							tier = "STANDARD"
						} else {
							tier = "REJECT"
						}
					}

					validationResults[idx] = AIValidationResult{
						Score:      res.Score,
						Confidence: res.Confidence,
						Tier:       tier,
						Reason:     res.Reason,
					}
				}
			}

			log.Printf("✅ [AI Batch] Successfully validated %d signals with model: %s (Client %d)", len(signals), modelName, cIdx+1)
			return validationResults, nil
		}
	}

	return nil, fmt.Errorf("unexpected error: %w", lastError)
}

// validateSignalInternal helper to generate prompt text
func generateSignalPrompt(signal *model.Signal) string {
	// Calculate volume ratio safely
	volRatio := 0.0
	if signal.TechnicalContext.AvgVol > 0 {
		volRatio = signal.TechnicalContext.CurrentVol / signal.TechnicalContext.AvgVol
	}

	return fmt.Sprintf(`
━━━━━━━━━━ SIGNAL CANDIDATE: %s ━━━━━━━━━━
📌 **BASIC INFO:**
- **Symbol:** %s
- **Signal Type:** %s (Proposed Direction)
- **Strategy Tier:** %s
- **Market Regime:** %s (Context: TRENDING_UP/DOWN favors trend following, RANGING/CHOPPY requires caution)

💰 **RISK/REWARD & MONEY MANAGEMENT:**
*Why this matters: A good trade must have high R:R and controlled risk.*
- **Entry Price:** %s
- **Stop Loss:** %s (Distance: %.2f%%)
- **Take Profit:** %s (Distance: %.2f%%)
- **ATR (1H):** %.4f (Volatility context)
- **R:R Ratio:** %.2f (MUST be > 2.0 for validity)
- **Break-Even Win Rate:** %.2f%% (Win rate needed to not lose money)
- **Kelly Position Size:** %.2f%% (Aggressive sizing based on probability)

📊 **MOMENTUM & TREND (The "Engine"):**
*Interpretation: Aligning momentum across timeframes increases success rate.*
- **RSI (Relative Strength Index):**
  - 4H: %.1f | 1H: %.1f | 15m: %.1f | 5m: %.1f
  - *Guide: >70 Overbought (Bearish context), <30 Oversold (Bullish context). 45-55 is neutral.*
- **Stochastic RSI (15m):** %.1f (K-Line)
  - *Guide: >80 is Overbought, <20 is Oversold.*
- **ADX (Trend Strength):**
  - 4H: %.1f | 1H: %.1f | 15m: %.1f
  - *Guide: >25 indicates a STRONG trend. <20 indicates weak/choppy market.*
- **MACD (5m):** Line: %.6f | Signal: %.6f | Hist: %.6f
  - *Guide: Positive = Bullish momentum, Negative = Bearish momentum.*
- **Trend State:** %s (e.g., Golden Cross = Bullish, Death Cross = Bearish)

🌊 **VOLUME & FLOW (The "Fuel"):**
*Interpretation: Price moves without volume are often fake-outs.*
- **VWAP:** %.4f (Volume Weighted Average Price)
- **Volume Ratio:** %.2fx (Current vs Avg)
  - *Guide: >1.5x confirms breakouts/moves. <1.0x suggests weak participation.*
- **Order Flow Delta:** %.2f
  - *Guide: Positive = Aggressive Buying, Negative = Aggressive Selling.*

🔭 **MARKET STRUCTURE & LEVELS (The "Map"):**
*Interpretation: Price reacts at these key psychological levels.*
- **Pivot Points:** Pivot: %s | S1: %s | R1: %s
- **Nearest Pivot:** %s is %.2f%% away
- **Fibonacci Retracement:** 50%%: %s | 61.8%%: %s | Nearest: %s
- **Nearest Level Proximity:** %.2f%% (Closer to level = Better entry)

🧠 **SMART MONEY CONCEPTS (SMC):**
*Interpretation: Institutional footprints.*
- **Order Block (OB):** %s (Price is inside/near a bank trading zone?)
- **Fair Value Gap (FVG):** %s (Imbalance area acting as magnet?)
- **Liquidity Sweep:** %s (Has a recent high/low been raided for stop losses?)
- **POC (Volume Profile):** %s (Distance: %.2f%%)
  - *Guide: Point of Control is the fair price. Price often reverts to it.*
- **BTC Correlation:** %s (Trading against BTC trend is risky)

🔬 **PATTERN ANALYSIS:**
- **Candlestick Pattern:** %s (Immediate price action trigger)
- **Divergence:** %s (RSI making Highs while Price makes Lows? Reversal signal)

📈 **INTERNAL SYSTEM SCORE:**
- **Confluence Score:** %d/100 (Sum of all technical factors)
- **Internal Confidence:** %.1f%%

---
**YOUR TASK:**
Analyze the data above. Does the narrative make sense?
- If Trend is UP but RSI is Overbought (4H), is it a pullback or top?
- If Volume is low (Ratio < 1), is the move sustainable?
- Is the R:R actually worth the risk given the Pivot/Resistance levels?

Provide your expert verdict.
`,
		signal.Symbol,
		signal.Symbol,
		signal.Type,
		signal.Tier,
		signal.Regime,
		FormatPrice(signal.EntryPrice),
		FormatPrice(signal.StopLoss),
		signal.RiskPercent,
		FormatPrice(signal.TakeProfit),
		signal.RewardPercent,
		signal.TechnicalContext.ATR,
		signal.RiskRewardRatio,
		signal.BreakEvenWinRate,
		signal.RecommendedSize,
		signal.TechnicalContext.RSI4h,
		signal.TechnicalContext.RSI1h,
		signal.TechnicalContext.RSI15m,
		signal.TechnicalContext.RSI5m,
		signal.TechnicalContext.StochRSI,
		signal.TechnicalContext.ADX4h,
		signal.TechnicalContext.ADX1h,
		signal.TechnicalContext.ADX15m,
		signal.TechnicalContext.MACD,
		signal.TechnicalContext.Signal,
		signal.TechnicalContext.Histogram,
		signal.TechnicalContext.TrendState,
		signal.TechnicalContext.VWAP,
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
		signal.NearestLevelDist,
		signal.TechnicalContext.OBType,
		signal.TechnicalContext.FVGType,
		signal.TechnicalContext.LiquiditySweep,
		FormatPrice(signal.TechnicalContext.POC),
		signal.TechnicalContext.POCDistance,
		signal.TechnicalContext.BTCCorrelation,
		signal.TechnicalContext.CandlestickPattern,
		signal.TechnicalContext.Divergence,
		signal.ConfluenceScore,
		signal.ConfidenceScore*100,
	)
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
