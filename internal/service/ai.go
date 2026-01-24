package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"mrcrypto-go/internal/config"
	"mrcrypto-go/internal/model"

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

	prompt := fmt.Sprintf(`তুমি একজন ১০+ বছরের অভিজ্ঞ ক্রিপ্টো ট্রেডিং বিশ্লেষক। নিচের সিগন্যালটি বিশ্লেষণ করে সঠিক সিদ্ধান্ত দাও।

╔══════════════════════════════════════════════════════════════╗
                    🔔 সিগন্যাল ওভারভিউ
╚══════════════════════════════════════════════════════════════╝

📌 সিম্বল: %s
📌 ডিরেকশন: %s
📌 টায়ার: %s
📌 মার্কেট রেজিম: %s

╔══════════════════════════════════════════════════════════════╗
              💰 রিস্ক ম্যানেজমেন্ট ডেটা
╚══════════════════════════════════════════════════════════════╝

🎯 এন্ট্রি প্রাইস: %s
🛑 স্টপ লস: %s (রিস্ক: %.2f%%)
🏆 টেক প্রফিট: %s (রিওয়ার্ড: %.2f%%)

📊 রিস্ক/রিওয়ার্ড রেশিও: %.2f
   → ব্যাখ্যা: প্রতি $১ রিস্কে $%.2f রিওয়ার্ড
   → গ্রহণযোগ্য: >= ২.০

🎲 ব্রেক-ইভেন উইন রেট: %.2f%%
   → ব্যাখ্যা: এই R:R তে প্রফিটেবল থাকতে ন্যূনতম যত %% ট্রেড জিততে হবে
   → এটা যত কম, তত ভালো

💼 প্রস্তাবিত পজিশন সাইজ: %.2f%% (Kelly Criterion অনুযায়ী)

╔══════════════════════════════════════════════════════════════╗
                📊 টেকনিক্যাল ইন্ডিকেটর
╚══════════════════════════════════════════════════════════════╝

━━━ RSI (Relative Strength Index) ━━━
• 4H RSI: %.1f
• 1H RSI: %.1f  
• 15M RSI: %.1f
• 5M RSI: %.1f

📖 RSI ব্যাখ্যা:
   - > 70 = ওভারবট (LONG এ সতর্ক)
   - < 30 = ওভারসোল্ড (SHORT এ সতর্ক)
   - 40-60 = নিউট্রাল জোন
   - LONG এর জন্য আদর্শ: 45-65
   - SHORT এর জন্য আদর্শ: 35-55

━━━ ADX (Average Directional Index) ━━━
• 4H ADX: %.1f
• 1H ADX: %.1f
• 15M ADX: %.1f

📖 ADX ব্যাখ্যা:
   - < 20 = দুর্বল ট্রেন্ড (ট্রেড এড়িয়ে চলুন)
   - 20-25 = মাঝারি ট্রেন্ড
   - 25-30 = শক্তিশালী ট্রেন্ড ✅
   - > 30 = অত্যন্ত শক্তিশালী ট্রেন্ড 🔥

━━━ MACD (Moving Average Convergence Divergence) ━━━
• হিস্টোগ্রাম: %.6f

📖 MACD ব্যাখ্যা:
   - হিস্টোগ্রাম > 0 = বুলিশ মোমেন্টাম (LONG ভালো)
   - হিস্টোগ্রাম < 0 = বিয়ারিশ মোমেন্টাম (SHORT ভালো)

━━━ ভলিউম বিশ্লেষণ ━━━
• বর্তমান ভলিউম: %.2fx গড় ভলিউম

📖 ভলিউম ব্যাখ্যা:
   - < 1.0x = কম ভলিউম (দুর্বল সিগন্যাল)
   - 1.0x-1.5x = স্বাভাবিক
   - 1.5x-2.0x = ভালো কনফার্মেশন ✅
   - > 2.0x = প্রতিষ্ঠানিক আগ্রহ 🔥

━━━ অর্ডার ফ্লো ডেল্টা ━━━
• ডেল্টা: %.2f

📖 অর্ডার ফ্লো ব্যাখ্যা:
   - > 0 = ক্রেতাদের চাপ বেশি (LONG সমর্থন করে)
   - < 0 = বিক্রেতাদের চাপ বেশি (SHORT সমর্থন করে)

━━━ VWAP (Volume Weighted Average Price) ━━━
• VWAP: %s

📖 VWAP ব্যাখ্যা:
   - প্রাইস > VWAP = বুলিশ সেন্টিমেন্ট
   - প্রাইস < VWAP = বিয়ারিশ সেন্টিমেন্ট

╔══════════════════════════════════════════════════════════════╗
                  🎯 কী লেভেল বিশ্লেষণ
╚══════════════════════════════════════════════════════════════╝

━━━ পিভট পয়েন্ট (Daily) ━━━
• R3 (রেজিস্ট্যান্স ৩): %s
• R2 (রেজিস্ট্যান্স ২): %s
• R1 (রেজিস্ট্যান্স ১): %s
• PP (পিভট পয়েন্ট): %s
• S1 (সাপোর্ট ১): %s
• S2 (সাপোর্ট ২): %s
• S3 (সাপোর্ট ৩): %s
• নিকটতম পিভট: %s

📖 পিভট ব্যাখ্যা:
   - LONG: সাপোর্ট (S1/S2) এর কাছে এন্ট্রি ভালো
   - SHORT: রেজিস্ট্যান্স (R1/R2) এর কাছে এন্ট্রি ভালো

━━━ ফিবোনাচ্চি রিট্রেসমেন্ট ━━━
• 38.2%% লেভেল: %s
• 50.0%% লেভেল: %s
• 61.8%% লেভেল: %s (গোল্ডেন রেশিও)
• নিকটতম ফিব: %s
• নিকটতম লেভেল থেকে দূরত্ব: %.2f%%

📖 ফিবোনাচ্চি ব্যাখ্যা:
   - 61.8%% = সবচেয়ে শক্তিশালী রিভার্সাল জোন
   - 50%% = সাইকোলজিক্যাল লেভেল
   - কী লেভেল থেকে ২%% এর মধ্যে এন্ট্রি = ভালো

╔══════════════════════════════════════════════════════════════╗
                📐 প্রোবাবিলিটি মেট্রিক্স
╚══════════════════════════════════════════════════════════════╝

🎯 কনফ্লুয়েন্স স্কোর: %d/100
   → ব্যাখ্যা: কতগুলো ফ্যাক্টর একমত আছে
   → 60+ = গ্রহণযোগ্য
   → 80+ = এক্সিলেন্ট

📊 সিগন্যাল কনফিডেন্স: %.1f%%
   → ব্যাখ্যা: কনফ্লুয়েন্স স্কোর থেকে গণনা করা সম্ভাব্যতা

╔══════════════════════════════════════════════════════════════╗
                ✅ তোমার মূল্যায়ন করতে হবে
╚══════════════════════════════════════════════════════════════╝

নিচের প্রশ্নগুলোর উত্তর দিয়ে সিদ্ধান্ত নাও:

1️⃣ RSI কি ডিরেকশনের সাথে মিলছে?
   - LONG = RSI 40-65 হওয়া উচিত
   - SHORT = RSI 35-55 হওয়া উচিত

2️⃣ ট্রেন্ড কি যথেষ্ট শক্তিশালী?
   - ADX >= 20 হওয়া উচিত
   - আদর্শ: ADX >= 25

3️⃣ এন্ট্রি কি ভালো জায়গায়?
   - কী লেভেল (সাপোর্ট/রেজিস্ট্যান্স) থেকে ২%% এর মধ্যে?

4️⃣ R:R কি যুক্তিসঙ্গত?
   - R:R >= 2.0 হওয়া উচিত

5️⃣ ভলিউম কি কনফার্ম করছে?
   - >= 1.5x গড় ভলিউম থাকলে ভালো

╔══════════════════════════════════════════════════════════════╗
                    📝 তোমার রেসপন্স
╚══════════════════════════════════════════════════════════════╝

⚠️ গুরুত্বপূর্ণ: রেসপন্স বাংলায় দাও।

শুধু JSON ফরম্যাটে উত্তর দাও:
{"score": <0-100>, "reason": "<বিস্তারিত বাংলায় বিশ্লেষণ>"}

স্কোরিং গাইড:
• 80-100 = এক্সিলেন্ট সিগন্যাল (সব ফ্যাক্টর মিলছে)
• 70-79 = ভালো সিগন্যাল (বেশিরভাগ ফ্যাক্টর মিলছে)
• 60-69 = গ্রহণযোগ্য (কিছু ঝুঁকি আছে)
• 40-59 = দুর্বল (অনেক ফ্যাক্টর মিলছে না)
• 0-39 = এড়িয়ে চলুন`,
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
