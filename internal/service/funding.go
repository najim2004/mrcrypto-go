package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

// FundingRateInfo contains funding rate data
type FundingRateInfo struct {
	Symbol      string
	FundingRate float64 // Percentage (e.g., 0.01 = 0.01%)
	NextFunding time.Time
	Sentiment   string // BULLISH, BEARISH, NEUTRAL, EXTREME_LONG, EXTREME_SHORT
	Warning     string
	RiskLevel   string // LOW, MEDIUM, HIGH
}

// BinanceFundingResponse represents Binance API response
type BinanceFundingResponse struct {
	Symbol      string `json:"symbol"`
	FundingRate string `json:"fundingRate"`
	FundingTime int64  `json:"fundingTime"`
}

// GetFundingRate fetches current funding rate from Binance Futures API (FREE, no API key needed)
func GetFundingRate(symbol string) (*FundingRateInfo, error) {
	url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/fundingRate?symbol=%s&limit=1", symbol)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch funding rate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binance API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var fundingData []BinanceFundingResponse
	if err := json.Unmarshal(body, &fundingData); err != nil {
		return nil, fmt.Errorf("failed to parse funding rate: %w", err)
	}

	if len(fundingData) == 0 {
		return nil, fmt.Errorf("no funding data returned")
	}

	rate, err := strconv.ParseFloat(fundingData[0].FundingRate, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse funding rate value: %w", err)
	}

	// Convert to percentage
	ratePercent := rate * 100

	info := &FundingRateInfo{
		Symbol:      symbol,
		FundingRate: ratePercent,
		NextFunding: time.UnixMilli(fundingData[0].FundingTime),
	}

	// Analyze funding rate sentiment
	analyzeFundingRate(info)

	return info, nil
}

// analyzeFundingRate determines sentiment and risk from funding rate
func analyzeFundingRate(info *FundingRateInfo) {
	rate := info.FundingRate

	switch {
	case rate > 0.1:
		// Extreme positive funding - too many longs
		info.Sentiment = "EXTREME_LONG"
		info.RiskLevel = "HIGH"
		info.Warning = "⚠️ Funding অত্যন্ত বেশি! LONG ঝুঁকিপূর্ণ, SHORT সুযোগ হতে পারে।"
	case rate > 0.05:
		// High positive funding
		info.Sentiment = "BULLISH"
		info.RiskLevel = "MEDIUM"
		info.Warning = "Funding বেশি। LONG এ সাবধান।"
	case rate < -0.1:
		// Extreme negative funding - too many shorts
		info.Sentiment = "EXTREME_SHORT"
		info.RiskLevel = "HIGH"
		info.Warning = "⚠️ Funding অত্যন্ত কম! SHORT ঝুঁকিপূর্ণ, LONG সুযোগ হতে পারে।"
	case rate < -0.05:
		// High negative funding
		info.Sentiment = "BEARISH"
		info.RiskLevel = "MEDIUM"
		info.Warning = "Funding কম। SHORT এ সাবধান।"
	default:
		// Neutral funding
		info.Sentiment = "NEUTRAL"
		info.RiskLevel = "LOW"
		info.Warning = ""
	}
}

// CalculateFundingScore calculates score adjustment using already-fetched funding info
// This avoids duplicate API calls
func CalculateFundingScore(info *FundingRateInfo, direction string) int {
	if info == nil {
		return 0 // No penalty if funding info not available
	}

	// Score adjustment based on funding vs trade direction
	switch {
	case direction == "LONG" && info.Sentiment == "EXTREME_LONG":
		return -15 // Heavy penalty - going long when everyone is long
	case direction == "SHORT" && info.Sentiment == "EXTREME_SHORT":
		return -15 // Heavy penalty - going short when everyone is short
	case direction == "LONG" && info.Sentiment == "EXTREME_SHORT":
		return 10 // Bonus - contrarian long in extreme short
	case direction == "SHORT" && info.Sentiment == "EXTREME_LONG":
		return 10 // Bonus - contrarian short in extreme long
	case direction == "LONG" && info.Sentiment == "BULLISH":
		return -5 // Slight penalty
	case direction == "SHORT" && info.Sentiment == "BEARISH":
		return -5 // Slight penalty
	default:
		return 0 // Neutral
	}
}

// GetFundingScore returns confluence score adjustment based on funding
// direction: "LONG" or "SHORT"
// NOTE: This makes an API call - use CalculateFundingScore if you already have funding info
func GetFundingScore(symbol, direction string) int {
	info, err := GetFundingRate(symbol)
	if err != nil {
		log.Printf("⚠️ Failed to fetch funding for %s: %v", symbol, err)
		return 0 // No penalty if API fails
	}

	log.Printf("📊 [Funding] %s: %.4f%% (%s)", symbol, info.FundingRate, info.Sentiment)
	return CalculateFundingScore(info, direction)
}

// IsFundingRisky checks if trade is risky based on funding
func IsFundingRisky(symbol, direction string) (bool, string) {
	info, err := GetFundingRate(symbol)
	if err != nil {
		return false, ""
	}

	if direction == "LONG" && info.Sentiment == "EXTREME_LONG" {
		return true, info.Warning
	}
	if direction == "SHORT" && info.Sentiment == "EXTREME_SHORT" {
		return true, info.Warning
	}

	return false, info.Warning
}
