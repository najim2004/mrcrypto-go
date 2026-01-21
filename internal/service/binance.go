package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"my-tool-go/internal/config"
	"my-tool-go/internal/model"
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
	for _, k := range klineData {
		if len(k) < 11 {
			continue
		}

		openTime, _ := k[0].(float64)
		open, _ := strconv.ParseFloat(k[1].(string), 64)
		high, _ := strconv.ParseFloat(k[2].(string), 64)
		low, _ := strconv.ParseFloat(k[3].(string), 64)
		closePrice, _ := strconv.ParseFloat(k[4].(string), 64)
		volume, _ := strconv.ParseFloat(k[5].(string), 64)
		closeTime, _ := k[6].(float64)

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
