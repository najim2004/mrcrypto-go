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

// ========================================
// ORDER BOOK DEPTH ANALYSIS
// ========================================

// OrderBookDepth represents bid/ask volume analysis
type OrderBookDepth struct {
	BidVolume float64
	AskVolume float64
	Imbalance float64 // (Bid - Ask) / (Bid + Ask) * 100
	Signal    string  // "Buy Pressure" / "Sell Pressure" / "Balanced"
}

// DepthResponse represents Binance order book depth response
type DepthResponse struct {
	Bids [][]interface{} `json:"bids"`
	Asks [][]interface{} `json:"asks"`
}

// GetOrderBookDepth fetches and analyzes order book depth
// limit: 100 for detailed analysis, 500 for comprehensive (max allowed by Binance)
func (s *BinanceService) GetOrderBookDepth(symbol string, limit int) (*OrderBookDepth, error) {
	url := fmt.Sprintf("%s/api/v3/depth?symbol=%s&limit=%d", s.baseURL, symbol, limit)

	resp, err := s.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch depth: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("binance API error: %s - %s", resp.Status, string(body))
	}

	var depthData DepthResponse
	if err := json.NewDecoder(resp.Body).Decode(&depthData); err != nil {
		return nil, fmt.Errorf("failed to decode depth: %w", err)
	}

	// Calculate total bid and ask volume
	bidVolume := 0.0
	askVolume := 0.0

	for _, bid := range depthData.Bids {
		if len(bid) >= 2 {
			qty := SafeTypeAssertString(bid[1], "0")
			vol, _ := strconv.ParseFloat(qty, 64)
			bidVolume += vol
		}
	}

	for _, ask := range depthData.Asks {
		if len(ask) >= 2 {
			qty := SafeTypeAssertString(ask[1], "0")
			vol, _ := strconv.ParseFloat(qty, 64)
			askVolume += vol
		}
	}

	// Calculate imbalance
	totalVolume := bidVolume + askVolume
	imbalance := 0.0
	signal := "Balanced"

	if totalVolume > 0 {
		imbalance = ((bidVolume - askVolume) / totalVolume) * 100

		if imbalance > 20 {
			signal = "Strong Buy Pressure"
		} else if imbalance > 10 {
			signal = "Buy Pressure"
		} else if imbalance < -20 {
			signal = "Strong Sell Pressure"
		} else if imbalance < -10 {
			signal = "Sell Pressure"
		}
	}

	log.Printf("📚 [Order Book] %s - Bid: %.2f | Ask: %.2f | Imbalance: %.1f%% (%s)",
		symbol, bidVolume, askVolume, imbalance, signal)

	return &OrderBookDepth{
		BidVolume: bidVolume,
		AskVolume: askVolume,
		Imbalance: imbalance,
		Signal:    signal,
	}, nil
}

// ========================================
// PERP VS SPOT DIVERGENCE
// ========================================

// PerpSpotDivergence represents futures vs spot price comparison
type PerpSpotDivergence struct {
	PerpPrice float64
	SpotPrice float64
	Premium   float64 // (Perp - Spot) / Spot * 100
	Sentiment string  // "Overheated Longs" / "Neutral" / "Bearish Discount"
}

// TickerPriceResponse represents Binance ticker price response
type TickerPriceResponse struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

// GetSpotPrice fetches current spot price
func (s *BinanceService) GetSpotPrice(symbol string) (float64, error) {
	url := fmt.Sprintf("%s/api/v3/ticker/price?symbol=%s", s.baseURL, symbol)

	resp, err := s.client.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch spot price: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("binance API error: %s - %s", resp.Status, string(body))
	}

	var tickerData TickerPriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&tickerData); err != nil {
		return 0, fmt.Errorf("failed to decode spot price: %w", err)
	}

	price, err := strconv.ParseFloat(tickerData.Price, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse spot price: %w", err)
	}

	return price, nil
}

// GetPerpSpotDivergence calculates perpetual vs spot price divergence
func (s *BinanceService) GetPerpSpotDivergence(perpSymbol string, perpPrice float64) (*PerpSpotDivergence, error) {
	// Convert BTCUSDT (perp) to BTCUSDT (spot) - same symbol for Binance
	spotPrice, err := s.GetSpotPrice(perpSymbol)
	if err != nil {
		return nil, err
	}

	// Calculate premium/discount
	premium := ((perpPrice - spotPrice) / spotPrice) * 100
	sentiment := "Neutral"

	if premium > 0.5 {
		sentiment = "Overheated Longs"
	} else if premium > 0.2 {
		sentiment = "Bullish Premium"
	} else if premium < -0.5 {
		sentiment = "Bearish Discount"
	} else if premium < -0.2 {
		sentiment = "Oversold Shorts"
	}

	log.Printf("💱 [Perp-Spot] %s - Perp: $%.2f | Spot: $%.2f | Premium: %.3f%% (%s)",
		perpSymbol, perpPrice, spotPrice, premium, sentiment)

	return &PerpSpotDivergence{
		PerpPrice: perpPrice,
		SpotPrice: spotPrice,
		Premium:   premium,
		Sentiment: sentiment,
	}, nil
}
