package service

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"mrcrypto-go/internal/config"
	"mrcrypto-go/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DatabaseService struct {
	client     *mongo.Client
	collection *mongo.Collection
}

func NewDatabaseService() (*DatabaseService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.AppConfig.MongoURI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	collection := client.Database("mrcrypto").Collection("signals")

	log.Println("✅ MongoDB connected successfully")

	return &DatabaseService{
		client:     client,
		collection: collection,
	}, nil
}

// SaveSignal saves a trading signal to the database
func (s *DatabaseService) SaveSignal(signal *model.Signal) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("💾 [Database] Saving signal for %s...", signal.Symbol)
	signal.CreatedAt = time.Now()

	_, err := s.collection.InsertOne(ctx, signal)
	if err != nil {
		return fmt.Errorf("failed to save signal: %w", err)
	}

	log.Printf("💾 Signal saved to database: %s %s", signal.Symbol, signal.Type)
	return nil
}

// GetLastSignalTime retrieves the timestamp of the last signal for a symbol
func (s *DatabaseService) GetLastSignalTime(symbol string) (time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"symbol": symbol}
	opts := options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})

	var result model.Signal
	err := s.collection.FindOne(ctx, filter, opts).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return time.Time{}, nil // No previous signal
		}
		return time.Time{}, fmt.Errorf("failed to get last signal: %w", err)
	}

	return result.CreatedAt, nil
}

// CheckCooldown checks if enough time has passed since the last signal
func (s *DatabaseService) CheckCooldown(symbol string, duration time.Duration) bool {
	log.Printf("⏳ [Database] Checking cooldown for %s...", symbol)
	lastTime, err := s.GetLastSignalTime(symbol)
	if err != nil {
		log.Printf("⚠️  Error checking cooldown for %s: %v", symbol, err)
		return false
	}

	if lastTime.IsZero() {
		return false // No previous signal, no cooldown
	}

	timeSince := time.Since(lastTime)
	if timeSince < duration {
		log.Printf("⏱️  Cooldown active for %s: %.1f minutes remaining",
			symbol, (duration - timeSince).Minutes())
		return true
	}

	return false
}

// CheckDuplicateActiveSignal checks if an active signal already exists for symbol+type
// Returns TRUE if duplicate (should skip), FALSE if allowed (e.g. price difference > 1.5%)
func (s *DatabaseService) CheckDuplicateActiveSignal(symbol string, signalType model.SignalType, newEntryPrice float64) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find the latest active signal for this symbol and type
	filter := bson.M{
		"symbol": symbol,
		"type":   signalType,
		"status": "ACTIVE",
	}
	// Sort by newest first to compare with latest entry
	opts := options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})

	var existingSignal model.Signal
	err := s.collection.FindOne(ctx, filter, opts).Decode(&existingSignal)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false // No active signal found, safe to proceed
		}
		log.Printf("⚠️  Error checking duplicate for %s %s: %v", symbol, signalType, err)
		return false // Assume safe on error to avoid blocking valid signals
	}

	// Active signal exists. Check price difference.
	// Logic: If price has moved significantly (> 1.5%), allow "Scaling In"
	priceDiff := math.Abs(existingSignal.EntryPrice - newEntryPrice)
	percentDiff := (priceDiff / existingSignal.EntryPrice) * 100

	if percentDiff > 1.5 {
		log.Printf("✅ %s %s - Scaling In Allowed (Price diff: %.2f%% from prev entry)", symbol, signalType, percentDiff)
		return false // Not a duplicate (conceptually), allow new signal
	}

	log.Printf("⏭️  %s %s - Active signal exists at similar price (%.2f%% diff) - Duplicate prevented", symbol, signalType, percentDiff)
	return true
}

// Close closes the database connection
func (s *DatabaseService) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect from MongoDB: %w", err)
	}

	log.Println("🔌 MongoDB connection closed")
	return nil
}

// CloseAllActiveSignals closes all currently active signals (e.g. for daily cleanup)
func (s *DatabaseService) CloseAllActiveSignals(reason string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"status": "ACTIVE"}
	update := bson.M{
		"$set": bson.M{
			"status":       "CLOSED",
			"close_reason": reason,
			"closed_at":    time.Now(),
		},
	}

	result, err := s.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, fmt.Errorf("failed to close active signals: %w", err)
	}

	if result.ModifiedCount > 0 {
		log.Printf("🧹 [Database] Closed %d active signals (Reason: %s)", result.ModifiedCount, reason)
	}
	return result.ModifiedCount, nil
}

// GetDB returns the MongoDB database instance
func (s *DatabaseService) GetDB() *mongo.Database {
	return s.client.Database("mrcrypto")
}
