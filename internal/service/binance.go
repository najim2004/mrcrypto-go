package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"mrcrypto-go/internal/config"
	"mrcrypto-go/internal/model"
)

type BinanceService struct {
	baseURL string
	client  *http.Client
}

func NewBinanceService() *BinanceService {
	return &BinanceService{
		baseURL: config.AppConfig.BinanceBaseURL,
		client:  &http.Client{},
	}
}

// KlineResponse represents Binance API response for klines
type KlineResponse []interface{}

// GetKlines fetches candlestick data from Binance
func (s *BinanceService) GetKlines(symbol, interval string, limit int) ([]model.Kline, error) {
	url := fmt.Sprintf("%s/api/v3/klines?symbol=%s&interval=%s&limit=%d",
		s.baseURL, symbol, interval, limit)

	log.Printf("🌐 [Binance API] Fetching %s klines (%s, limit: %d)...", symbol, interval, limit)
	resp, err := s.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch klines: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("binance API error: %s - %s", resp.Status, string(body))
	}

	var klineData []KlineResponse
	if err := json.NewDecoder(resp.Body).Decode(&klineData); err != nil {
		return nil, fmt.Errorf("failed to decode klines: %w", err)
	}

	klines := make([]model.Kline, 0, len(klineData))
	for idx, k := range klineData {
		// Validate array length
		if len(k) < 11 {
			log.Printf("⚠️  [Binance API] Skipping invalid kline at index %d: insufficient fields (%d/11)", idx, len(k))
			continue
		}

		// Safe type assertions with validation
		openTime := SafeTypeAssertFloat(k[0], 0)
		openStr := SafeTypeAssertString(k[1], "0")
		highStr := SafeTypeAssertString(k[2], "0")
		lowStr := SafeTypeAssertString(k[3], "0")
		closeStr := SafeTypeAssertString(k[4], "0")
		volumeStr := SafeTypeAssertString(k[5], "0")
		closeTime := SafeTypeAssertFloat(k[6], 0)

		// Parse strings to floats with error handling
		open, err1 := strconv.ParseFloat(openStr, 64)
		high, err2 := strconv.ParseFloat(highStr, 64)
		low, err3 := strconv.ParseFloat(lowStr, 64)
		closePrice, err4 := strconv.ParseFloat(closeStr, 64)
		volume, err5 := strconv.ParseFloat(volumeStr, 64)

		// Check for parse errors
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil {
			log.Printf("⚠️  [Binance API] Skipping kline at index %d: parse error", idx)
			continue
		}

		// Validate prices are reasonable
		if !ValidatePrice(open) || !ValidatePrice(high) || !ValidatePrice(low) || !ValidatePrice(closePrice) {
			log.Printf("⚠️  [Binance API] Skipping kline at index %d: invalid price values", idx)
			continue
		}

		// Validate OHLC logic: High >= Low, High >= Open/Close, Low <= Open/Close
		if high < low || high < open || high < closePrice || low > open || low > closePrice {
			log.Printf("⚠️  [Binance API] Skipping kline at index %d: invalid OHLC relationship", idx)
			continue
		}

		klines = append(klines, model.Kline{
			OpenTime:  int64(openTime),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closePrice,
			Volume:    volume,
			CloseTime: int64(closeTime),
		})
	}

	if len(klines) == 0 {
		return nil, fmt.Errorf("no valid klines after parsing")
	}

	log.Printf("✅ [Binance API] Successfully fetched %d %s klines for %s", len(klines), interval, symbol)
	return klines, nil
}

// ExchangeInfoResponse represents Binance exchange info response
type ExchangeInfoResponse struct {
	Symbols []struct {
		Symbol string `json:"symbol"`
		Status string `json:"status"`
	} `json:"symbols"`
}

// GetAllSymbols returns a curated list of major trading pairs
func (s *BinanceService) GetAllSymbols() ([]string, error) {
	// Static list of major coins for focused analysis
	symbols := []string{
		"BTCUSDT",   // Bitcoin
		"ETHUSDT",   // Ethereum
		"SOLUSDT",   // Solana
		"BNBUSDT",   // Binance Coin
		"XRPUSDT",   // Ripple
		"DOGEUSDT",  // Dogecoin
		"ADAUSDT",   // Cardano
		"AVAXUSDT",  // Avalanche
		"TRXUSDT",   // Tron
		"LINKUSDT",  // Chainlink
		"DOTUSDT",   // Polkadot
		"MATICUSDT", // Polygon
		"LTCUSDT",   // Litecoin
		"SHIBUSDT",  // Shiba Inu
		"PEPEUSDT",  // Pepe
		"SUIUSDT",   // Sui
		"ARBUSDT",   // Arbitrum
		"OPUSDT",    // Optimism
		"APTUSDT",   // Aptos
		"INJUSDT",   // Injective
	}

	return symbols, nil
}
