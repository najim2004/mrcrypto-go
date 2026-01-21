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

	prompt := fmt.Sprintf(`You are a senior crypto analyst. Analyze this trading signal and provide a score from 0-100 and a brief reason.
IMPORTANT: The "reason" field MUST be in Bengali (Bangla) language.

Symbol: %s
Type: %s
Tier: %s
Entry Price: %s
Regime: %s

Technical Context:
- RSI (4h/1h/15m/5m): %.2f / %.2f / %.2f / %.2f
- ADX (4h/1h/15m): %.2f / %.2f / %.2f
- VWAP: %.6f
- MACD: %.6f, Signal: %.6f, Histogram: %.6f
- Volume: Current=%.2f, Avg=%.2f (%.2fx)
- Order Flow Delta: %.2f

Respond ONLY with a JSON object in this exact format:
{"score": <number>, "reason": "<brief reason in Bangla>"}`,
		signal.Symbol,
		signal.Type,
		signal.Tier,
		FormatPrice(signal.EntryPrice),
		signal.Regime,
		signal.TechnicalContext.RSI4h,
		signal.TechnicalContext.RSI1h,
		signal.TechnicalContext.RSI15m,
		signal.TechnicalContext.RSI5m,
		signal.TechnicalContext.ADX4h,
		signal.TechnicalContext.ADX1h,
		signal.TechnicalContext.ADX15m,
		signal.TechnicalContext.VWAP,
		signal.TechnicalContext.MACD,
		signal.TechnicalContext.Signal,
		signal.TechnicalContext.Histogram,
		signal.TechnicalContext.CurrentVol,
		signal.TechnicalContext.AvgVol,
		signal.TechnicalContext.CurrentVol/signal.TechnicalContext.AvgVol,
		signal.TechnicalContext.OrderFlowDelta,
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

	// Build batch prompt
	prompt := `You are a senior crypto analyst. Analyze these trading signals and provide a score (0-100) and brief reason for EACH signal.
IMPORTANT: The "reason" field MUST be in Bengali (Bangla) language.

Respond with a JSON array in this exact format:
[
  {"signal": 1, "score": <number>, "reason": "<brief reason in Bangla>"},
  {"signal": 2, "score": <number>, "reason": "<brief reason in Bangla>"},
  ...
]

SIGNALS TO ANALYZE:
`

	for idx, signal := range signals {
		prompt += fmt.Sprintf(`
--- Signal %d ---
Symbol: %s
Type: %s
Tier: %s
Entry Price: %s
Regime: %s
Technical Context:
- RSI (4h/1h/15m/5m): %.2f / %.2f / %.2f / %.2f
- ADX (4h/1h/15m): %.2f / %.2f / %.2f
- VWAP: %.6f
- MACD: %.6f, Signal: %.6f, Histogram: %.6f
- Volume: Current=%.2f, Avg=%.2f (%.2fx)
- Order Flow Delta: %.2f

`,
			idx+1,
			signal.Symbol,
			signal.Type,
			signal.Tier,
			FormatPrice(signal.EntryPrice),
			signal.Regime,
			signal.TechnicalContext.RSI4h,
			signal.TechnicalContext.RSI1h,
			signal.TechnicalContext.RSI15m,
			signal.TechnicalContext.RSI5m,
			signal.TechnicalContext.ADX4h,
			signal.TechnicalContext.ADX1h,
			signal.TechnicalContext.ADX15m,
			signal.TechnicalContext.VWAP,
			signal.TechnicalContext.MACD,
			signal.TechnicalContext.Signal,
			signal.TechnicalContext.Histogram,
			signal.TechnicalContext.CurrentVol,
			signal.TechnicalContext.AvgVol,
			signal.TechnicalContext.CurrentVol/signal.TechnicalContext.AvgVol,
			signal.TechnicalContext.OrderFlowDelta,
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
