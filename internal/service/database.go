package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"my-tool-go/internal/config"
	"my-tool-go/internal/model"

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
