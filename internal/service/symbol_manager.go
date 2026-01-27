package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SymbolManager struct {
	collection *mongo.Collection
}

type WatchedSymbol struct {
	Symbol   string    `bson:"symbol"`
	AddedAt  time.Time `bson:"added_at"`
	IsActive bool      `bson:"is_active"`
}

func NewSymbolManager(db *mongo.Database) *SymbolManager {
	collection := db.Collection("watchlist")

	sm := &SymbolManager{
		collection: collection,
	}

	// Seed initialization if empty
	sm.initializeDefaults()

	return sm
}

// initializeDefaults adds default critical symbols if DB is empty
func (sm *SymbolManager) initializeDefaults() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := sm.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Printf("⚠️ Failed to check watchlist count: %v", err)
		return
	}

	if count == 0 {
		log.Println("🌱 Seeding default watchlist...")
		defaults := []string{
			"BTCUSDT", "ETHUSDT", "SOLUSDT", "BNBUSDT", "XRPUSDT",
			"DOGEUSDT", "ADAUSDT", "AVAXUSDT", "TRXUSDT", "LINKUSDT",
			"DOTUSDT", "MATICUSDT", "LTCUSDT", "SHIBUSDT", "PEPEUSDT",
		}

		for _, s := range defaults {
			sm.AddSymbol(s)
		}
	}
}

// AddSymbol adds a symbol to the watchlist
func (sm *SymbolManager) AddSymbol(symbol string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if !strings.HasSuffix(symbol, "USDT") {
		return fmt.Errorf("symbol must end with USDT")
	}

	filter := bson.M{"symbol": symbol}
	update := bson.M{
		"$set": bson.M{
			"symbol":    symbol,
			"is_active": true,
		},
		"$setOnInsert": bson.M{
			"added_at": time.Now(),
		},
	}
	opts := options.Update().SetUpsert(true)

	_, err := sm.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to add symbol: %w", err)
	}

	log.Printf("✅ Added %s to watchlist", symbol)
	return nil
}

// RemoveSymbol removes a symbol from the watchlist
func (sm *SymbolManager) RemoveSymbol(symbol string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	// Hard delete or soft delete? Let's do hard delete for now as per user request
	_, err := sm.collection.DeleteOne(ctx, bson.M{"symbol": symbol})
	if err != nil {
		return fmt.Errorf("failed to remove symbol: %w", err)
	}

	log.Printf("🗑️ Removed %s from watchlist", symbol)
	return nil
}

// GetWatchlist returns all active symbols
func (sm *SymbolManager) GetWatchlist() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := sm.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []WatchedSymbol
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	var symbols []string
	for _, s := range results {
		symbols = append(symbols, s.Symbol)
	}

	return symbols, nil
}
