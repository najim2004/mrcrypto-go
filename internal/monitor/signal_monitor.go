package monitor

import (
	"context"
	"fmt"
	"log"
	"time"

	"my-tool-go/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type SignalMonitor struct {
	collection *mongo.Collection
}

func NewSignalMonitor(db *mongo.Database) *SignalMonitor {
	return &SignalMonitor{
		collection: db.Collection("active_signals"),
	}
}

// AddActiveSignal adds a signal to monitor
func (sm *SignalMonitor) AddActiveSignal(signal *model.Signal) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := sm.collection.InsertOne(ctx, signal)
	if err != nil {
		return err
	}

	log.Printf("📌 Added %s to active monitoring", signal.Symbol)
	return nil
}

// CheckTPSLHits checks if any active signals hit TP or SL
func (sm *SignalMonitor) CheckTPSLHits(currentPrices map[string]float64) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Find all active signals
	cursor, err := sm.collection.Find(ctx, bson.M{"status": "ACTIVE"})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var notifications []string

	for cursor.Next(ctx) {
		var signal model.Signal
		if err := cursor.Decode(&signal); err != nil {
			continue
		}

		currentPrice, exists := currentPrices[signal.Symbol]
		if !exists {
			continue
		}

		// Check TP/SL based on signal type
		if signal.Type == model.SignalTypeLong {
			if currentPrice >= signal.TakeProfit {
				notification := formatTPHit(signal, currentPrice)
				notifications = append(notifications, notification)
				sm.markSignalClosed(signal.Symbol, "TP_HIT")
			} else if currentPrice <= signal.StopLoss {
				notification := formatSLHit(signal, currentPrice)
				notifications = append(notifications, notification)
				sm.markSignalClosed(signal.Symbol, "SL_HIT")
			}
		} else { // SHORT
			if currentPrice <= signal.TakeProfit {
				notification := formatTPHit(signal, currentPrice)
				notifications = append(notifications, notification)
				sm.markSignalClosed(signal.Symbol, "TP_HIT")
			} else if currentPrice >= signal.StopLoss {
				notification := formatSLHit(signal, currentPrice)
				notifications = append(notifications, notification)
				sm.markSignalClosed(signal.Symbol, "SL_HIT")
			}
		}
	}

	return notifications, nil
}

func (sm *SignalMonitor) markSignalClosed(symbol, reason string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"status":       "CLOSED",
			"close_reason": reason,
			"closed_at":    time.Now(),
		},
	}

	sm.collection.UpdateOne(ctx, bson.M{"symbol": symbol, "status": "ACTIVE"}, update)
	log.Printf("🔒 Closed %s signal: %s", symbol, reason)
}

func formatTPHit(signal model.Signal, currentPrice float64) string {
	return "🎯 TAKE PROFIT HIT!\n" +
		"Symbol: " + signal.Symbol + "\n" +
		"Type: " + string(signal.Type) + "\n" +
		"Entry: " + formatPrice(signal.EntryPrice) + "\n" +
		"Exit: " + formatPrice(currentPrice) + "\n" +
		"Target: " + formatPrice(signal.TakeProfit)
}

func formatSLHit(signal model.Signal, currentPrice float64) string {
	return "🛑 STOP LOSS HIT!\n" +
		"Symbol: " + signal.Symbol + "\n" +
		"Type: " + string(signal.Type) + "\n" +
		"Entry: " + formatPrice(signal.EntryPrice) + "\n" +
		"Exit: " + formatPrice(currentPrice) + "\n" +
		"Stop: " + formatPrice(signal.StopLoss)
}

func formatPrice(price float64) string {
	if price < 0.001 {
		return fmt.Sprintf("%.8f", price)
	} else if price < 1 {
		return fmt.Sprintf("%.6f", price)
	} else if price < 10 {
		return fmt.Sprintf("%.4f", price)
	}
	return fmt.Sprintf("%.2f", price)
}
