package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"my-tool-go/internal/config"
	"my-tool-go/internal/model"

	"google.golang.org/genai"
)

type AIService struct {
	client *genai.Client
	ctx    context.Context
}

func NewAIService() *AIService {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: config.AppConfig.GeminiAPIKey,
	})
	if err != nil {
		log.Printf("⚠️  Failed to create Gemini client: %v", err)
		return &AIService{
			client: nil,
			ctx:    ctx,
		}
	}

	return &AIService{
		client: client,
		ctx:    ctx,
	}
}

// AIValidationResult contains the AI's assessment
type AIValidationResult struct {
	Score  int    `json:"score"`
	Reason string `json:"reason"`
}

// ValidateSignal sends the signal to Gemini AI for validation with fallback models
func (s *AIService) ValidateSignal(signal *model.Signal) (int, string, error) {
	if s.client == nil {
		return 0, "", fmt.Errorf("gemini client not initialized")
	}

	// Calculate volume ratio safely
	volRatio := 0.0
	if signal.TechnicalContext.AvgVol > 0 {
		volRatio = signal.TechnicalContext.CurrentVol / signal.TechnicalContext.AvgVol
	}

	prompt := fmt.Sprintf(`You are a professional crypto trading analyst. Analyze this trading signal using ALL the provided data to make an accurate decision.

CRITICAL EVALUATION CRITERIA:
1. Confluence Score >= 60 is acceptable, >= 80 is excellent
2. Risk/Reward >= 2.0 is required
3. Entry should be near a key level (pivot or fibonacci)
4. Volume should confirm the move (>= 1.5x average)
5. Signal confidence shows calculated probability of success

🔔 SIGNAL OVERVIEW:
Symbol: %s
Direction: %s (%s Tier)
Market Regime: %s

📈 ENTRY & RISK MANAGEMENT:
- Entry Price: %s
- Stop Loss: %s (Risk: %.2f%%)
- Take Profit: %s (Reward: %.2f%%)
- Risk/Reward Ratio: %.2f
- Break-even Win Rate Required: %.2f%%
- Recommended Position Size: %.2f%% of account

📊 TECHNICAL INDICATORS:
- RSI (4H/1H/15M/5M): %.1f / %.1f / %.1f / %.1f
- ADX (4H/1H/15M): %.1f / %.1f / %.1f
- MACD Histogram: %.6f
- Volume Ratio: %.2fx average
- Order Flow Delta: %.2f
- VWAP: %s

🎯 KEY LEVELS:
- Pivot Point: %s
- Support: S1=%s, S2=%s, S3=%s
- Resistance: R1=%s, R2=%s, R3=%s
- Nearest Pivot: %s
- Fibonacci 38.2%%: %s
- Fibonacci 50.0%%: %s
- Fibonacci 61.8%%: %s
- Nearest Fib: %s
- Distance to Nearest Level: %.2f%%

📐 PROBABILITY METRICS:
- Confluence Score: %d/100
- Signal Confidence: %.1f%%

ANALYSIS INSTRUCTIONS:
1. Check if RSI values indicate overbought (>70) or oversold (<30) conditions
2. Verify ADX > 20 for trend strength confirmation
3. Confirm entry is near a support (for LONG) or resistance (for SHORT)
4. Evaluate if R:R ratio justifies the trade
5. Consider confluence score and confidence probability
6. Volume should confirm the direction

IMPORTANT: Provide reason in Bengali (Bangla) language.

Respond ONLY with JSON:
{"score": <0-100>, "reason": "<detailed analysis in Bangla>"}`,
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
		FormatPrice(signal.TechnicalContext.PivotPoint),
		FormatPrice(signal.TechnicalContext.PivotS1),
		FormatPrice(signal.TechnicalContext.PivotS2),
		FormatPrice(signal.TechnicalContext.PivotS3),
		FormatPrice(signal.TechnicalContext.PivotR1),
		FormatPrice(signal.TechnicalContext.PivotR2),
		FormatPrice(signal.TechnicalContext.PivotR3),
		signal.TechnicalContext.NearestPivot,
		FormatPrice(signal.TechnicalContext.Fib382),
		FormatPrice(signal.TechnicalContext.Fib500),
		FormatPrice(signal.TechnicalContext.Fib618),
		signal.TechnicalContext.NearestFib,
		signal.NearestLevelDist,
		signal.ConfluenceScore,
		signal.ConfidenceScore*100,
	)

	// List of models to try in order (fallback)
	models := []string{
		"gemini-3-flash",
		"gemini-2.5-flash",
		"gemini-2.5-flash-lite",
		"gemini-2.5-flash-tts",
		"gemini-robotics-er-1.5-preview",
		"gemma-3-12b",
		"gemma-3-1b",
		"gemma-3-27b",
		"gemma-3-2b",
	}

	var lastError error

	// Try each model until one succeeds
	for i, modelName := range models {
		result, err := s.client.Models.GenerateContent(
			s.ctx,
			modelName,
			genai.Text(prompt),
			nil,
		)

		if err != nil {
			lastError = err
			log.Printf("⚠️  %s - Model %s failed, trying next model...", signal.Symbol, modelName)

			// If this is not the last model, continue to next
			if i < len(models)-1 {
				continue
			}
			// Last model also failed, return error
			return 0, "", fmt.Errorf("all gemini models failed, last error: %w", lastError)
		}

		// Success! Parse response
		responseText := result.Text()

		var aiResult AIValidationResult
		if err := json.Unmarshal([]byte(responseText), &aiResult); err != nil {
			log.Printf("⚠️  Failed to parse AI response for %s (model: %s): %v", signal.Symbol, modelName, err)
			return 50, responseText, nil
		}

		log.Printf("✅ [AI] %s - Validated! Model: %s, Score: %d/100", signal.Symbol, modelName, aiResult.Score)
		return aiResult.Score, aiResult.Reason, nil
	}

	// Should never reach here, but just in case
	return 0, "", fmt.Errorf("unexpected error: %w", lastError)
}

// BatchValidateSignals validates multiple signals in a single AI call (OPTIMIZED)
func (s *AIService) BatchValidateSignals(signals []*model.Signal) ([]AIValidationResult, error) {
	if s.client == nil {
		return nil, fmt.Errorf("gemini client not initialized")
	}

	if len(signals) == 0 {
		return []AIValidationResult{}, nil
	}

	// Build batch prompt with comprehensive data
	prompt := `You are a professional crypto trading analyst. Analyze these trading signals using ALL provided data.

CRITICAL EVALUATION CRITERIA:
1. Confluence Score >= 60 is acceptable, >= 80 is excellent
2. Risk/Reward >= 2.0 is required
3. Entry should be near a key level (pivot or fibonacci)
4. Volume should confirm the move (>= 1.5x average)
5. Consider probability metrics for confidence

IMPORTANT: Provide reason in Bengali (Bangla) language.

Respond with a JSON array:
[
  {"signal": 1, "score": <0-100>, "reason": "<analysis in Bangla>"},
  {"signal": 2, "score": <0-100>, "reason": "<analysis in Bangla>"},
  ...
]

SIGNALS TO ANALYZE:
`

	for idx, signal := range signals {
		// Calculate volume ratio safely
		volRatio := 0.0
		if signal.TechnicalContext.AvgVol > 0 {
			volRatio = signal.TechnicalContext.CurrentVol / signal.TechnicalContext.AvgVol
		}

		prompt += fmt.Sprintf(`
━━━━━━━━━━ SIGNAL %d ━━━━━━━━━━
Symbol: %s | Direction: %s | Tier: %s | Regime: %s

📈 RISK MANAGEMENT:
Entry: %s | SL: %s (%.2f%%) | TP: %s (%.2f%%)
R:R: %.2f | Break-even Win Rate: %.2f%% | Position: %.2f%%

📊 INDICATORS:
RSI (4H/1H/15M/5M): %.1f / %.1f / %.1f / %.1f
ADX (4H/1H/15M): %.1f / %.1f / %.1f
MACD Hist: %.6f | Volume: %.2fx | Order Flow: %.2f

🎯 KEY LEVELS:
Pivot: %s | S1: %s | R1: %s
Nearest: %s (%.2f%% away)
Fib 50%%: %s | Fib 61.8%%: %s

📐 PROBABILITY:
Confluence: %d/100 | Confidence: %.1f%%
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
		result, err := s.client.Models.GenerateContent(
			s.ctx,
			modelName,
			genai.Text(prompt),
			nil,
		)

		if err != nil {
			lastError = err
			log.Printf("⚠️  Batch validation - Model %s failed, trying next...", modelName)

			if i < len(models)-1 {
				continue
			}
			return nil, fmt.Errorf("all gemini models failed for batch validation: %w", lastError)
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

		log.Printf("✅ [AI Batch] Successfully validated %d signals with model: %s", len(signals), modelName)
		return validationResults, nil
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
