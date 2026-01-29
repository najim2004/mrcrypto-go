package service

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"mrcrypto-go/internal/model"
)

// PatternStats tracks performance of specific indicator patterns
type PatternStats struct {
	Pattern      string
	WinCount     int
	LossCount    int
	TotalCount   int
	WinRate      float64
	IsEnabled    bool
	LastOutcomes []bool // Last 10 outcomes for recent performance
}

// SignalTracker tracks and learns from signal outcomes
type SignalTracker struct {
	patterns map[string]*PatternStats
	mu       sync.RWMutex
}

// NewSignalTracker creates a new signal tracker
func NewSignalTracker() *SignalTracker {
	return &SignalTracker{
		patterns: make(map[string]*PatternStats),
	}
}

// GeneratePatternFingerprint creates a unique pattern ID from signal characteristics
func GeneratePatternFingerprint(signal *model.Signal) string {
	var components []string

	ctx := signal.TechnicalContext

	// ADX strength
	if ctx.ADX1h > 30 {
		components = append(components, "STRONG_TREND")
	} else if ctx.ADX1h > 25 {
		components = append(components, "MODERATE_TREND")
	}

	// RSI condition
	if signal.Type == model.SignalTypeLong {
		if ctx.RSI15m < 40 {
			components = append(components, "RSI_PULLBACK")
		}
	} else {
		if ctx.RSI15m > 60 {
			components = append(components, "RSI_PULLBACK")
		}
	}

	// SMC presence
	if ctx.FVGType != "" {
		components = append(components, fmt.Sprintf("FVG_%s", ctx.FVGType))
	}
	if ctx.OBType != "" {
		components = append(components, fmt.Sprintf("OB_%s", ctx.OBType))
	}

	// Divergence
	if ctx.Divergence != "" {
		components = append(components, ctx.Divergence)
	}

	// Candlestick pattern
	if ctx.CandlestickPattern != "" {
		components = append(components, ctx.CandlestickPattern)
	}

	// Liquidity sweep
	if ctx.LiquiditySweep != "" {
		components = append(components, strings.ReplaceAll(ctx.LiquiditySweep, " ", "_"))
	}

	// Market structure
	if ctx.MarketStructure != "" {
		components = append(components, ctx.MarketStructure)
	}

	// Trading session
	if ctx.TradingSession != "" {
		components = append(components, ctx.TradingSession)
	}

	// Combine into fingerprint
	if len(components) == 0 {
		return "BASIC_SIGNAL"
	}

	return strings.Join(components, "+")
}

// RecordSignalOutcome records whether a signal won or lost
func (st *SignalTracker) RecordSignalOutcome(signal *model.Signal, won bool) {
	pattern := GeneratePatternFingerprint(signal)

	st.mu.Lock()
	defer st.mu.Unlock()

	// Get or create pattern stats
	stats, exists := st.patterns[pattern]
	if !exists {
		stats = &PatternStats{
			Pattern:      pattern,
			IsEnabled:    true,
			LastOutcomes: make([]bool, 0, 10),
		}
		st.patterns[pattern] = stats
	}

	// Update stats
	stats.TotalCount++
	if won {
		stats.WinCount++
	} else {
		stats.LossCount++
	}

	// Update last outcomes (keep last 10)
	stats.LastOutcomes = append(stats.LastOutcomes, won)
	if len(stats.LastOutcomes) > 10 {
		stats.LastOutcomes = stats.LastOutcomes[1:]
	}

	// Calculate win rate
	if stats.TotalCount > 0 {
		stats.WinRate = float64(stats.WinCount) / float64(stats.TotalCount) * 100
	}

	// Auto-disable if performing poorly (after minimum sample size)
	if stats.TotalCount >= 10 && stats.WinRate < 40 {
		stats.IsEnabled = false
		log.Printf("🚫 [Signal Tracker] Pattern DISABLED: %s (Win Rate: %.1f%% after %d trades)",
			pattern, stats.WinRate, stats.TotalCount)
	} else if stats.TotalCount >= 10 && stats.WinRate >= 50 && !stats.IsEnabled {
		// Re-enable if it recovers
		stats.IsEnabled = true
		log.Printf("✅ [Signal Tracker] Pattern RE-ENABLED: %s (Win Rate: %.1f%% after %d trades)",
			pattern, stats.WinRate, stats.TotalCount)
	}

	outcome := "LOSS"
	if won {
		outcome = "WIN"
	}
	log.Printf("📊 [Signal Tracker] Recorded %s - Pattern: %s | Win Rate: %.1f%% (%d/%d)",
		outcome, pattern, stats.WinRate, stats.WinCount, stats.TotalCount)
}

// IsPatternEnabled checks if a pattern is enabled
func (st *SignalTracker) IsPatternEnabled(signal *model.Signal) bool {
	pattern := GeneratePatternFingerprint(signal)

	st.mu.RLock()
	defer st.mu.RUnlock()

	stats, exists := st.patterns[pattern]
	if !exists {
		// New pattern - enabled by default
		return true
	}

	return stats.IsEnabled
}

// GetPatternStats returns stats for a specific pattern
func (st *SignalTracker) GetPatternStats(pattern string) *PatternStats {
	st.mu.RLock()
	defer st.mu.RUnlock()

	return st.patterns[pattern]
}

// GetAllPatternStats returns all pattern statistics
func (st *SignalTracker) GetAllPatternStats() map[string]*PatternStats {
	st.mu.RLock()
	defer st.mu.RUnlock()

	// Return a copy to avoid race conditions
	statsCopy := make(map[string]*PatternStats)
	for k, v := range st.patterns {
		statsCopy[k] = v
	}

	return statsCopy
}

// LogPerformanceSummary logs a summary of all patterns
func (st *SignalTracker) LogPerformanceSummary() {
	st.mu.RLock()
	defer st.mu.RUnlock()

	log.Println("==========================================")
	log.Println("📈 SIGNAL PATTERN PERFORMANCE SUMMARY")
	log.Println("==========================================")

	totalPatterns := len(st.patterns)
	enabledPatterns := 0
	disabledPatterns := 0

	for _, stats := range st.patterns {
		if stats.IsEnabled {
			enabledPatterns++
		} else {
			disabledPatterns++
		}

		status := "✅ ENABLED"
		if !stats.IsEnabled {
			status = "🚫 DISABLED"
		}

		log.Printf("%s | %s | Win Rate: %.1f%% (%d/%d trades)",
			status, stats.Pattern, stats.WinRate, stats.WinCount, stats.TotalCount)
	}

	log.Println("------------------------------------------")
	log.Printf("Total Patterns: %d | Enabled: %d | Disabled: %d",
		totalPatterns, enabledPatterns, disabledPatterns)
	log.Println("==========================================")
}
