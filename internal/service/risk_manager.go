package service

import (
	"context"
	"log"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"mrcrypto-go/internal/model"
)

// RiskManager handles dynamic position sizing based on recent performance
type RiskManager struct {
	collection *mongo.Collection
}

// NewRiskManager creates a new risk manager instance
func NewRiskManager(db *mongo.Database) *RiskManager {
	return &RiskManager{
		collection: db.Collection("signals"),
	}
}

// PositionSizeInfo contains position sizing recommendation
type PositionSizeInfo struct {
	RecommendedSize float64 // % of account to risk
	Reason          string  // Why this size
	WinStreak       int     // Current win streak
	LoseStreak      int     // Current lose streak
	RecentWinRate   float64 // Win rate of last N trades
	TotalRecent     int     // Total recent trades analyzed
}

// CalculateDynamicPositionSize returns recommended position size based on:
// 1. Recent trade performance (win/lose streak)
// 2. Signal tier (Premium vs Standard)
// 3. Overall win rate
func (rm *RiskManager) CalculateDynamicPositionSize(tier model.SignalTier) *PositionSizeInfo {
	// Get last 10 closed trades
	recentTrades := rm.getRecentClosedTrades(10)

	info := &PositionSizeInfo{
		TotalRecent: len(recentTrades),
	}

	// Calculate streak and win rate
	winCount := 0
	for i, trade := range recentTrades {
		if trade.PnL > 0 {
			winCount++
			if i == 0 {
				info.WinStreak++
			} else if recentTrades[i-1].PnL > 0 {
				info.WinStreak++
			}
		} else {
			if i == 0 {
				info.LoseStreak++
			} else if recentTrades[i-1].PnL <= 0 {
				info.LoseStreak++
			}
		}
	}

	if len(recentTrades) > 0 {
		info.RecentWinRate = float64(winCount) / float64(len(recentTrades)) * 100
	}

	// Base position size based on tier
	baseSize := 1.0 // Default 1%
	if tier == model.TierPremium {
		baseSize = 2.0 // Premium gets 2%
	}

	// Adjust based on streak
	adjustedSize := baseSize

	switch {
	case info.LoseStreak >= 3:
		// 3+ losses in a row - reduce size significantly
		adjustedSize = baseSize * 0.5
		info.Reason = "🔴 ৩টি+ লস স্ট্রিক - রিস্ক কমানো হয়েছে"
	case info.LoseStreak >= 2:
		// 2 losses - slightly reduce
		adjustedSize = baseSize * 0.75
		info.Reason = "🟠 ২টি লস স্ট্রিক - রিস্ক কমানো হয়েছে"
	case info.WinStreak >= 3:
		// 3+ wins - can increase slightly
		adjustedSize = math.Min(baseSize*1.25, 3.0) // Max 3%
		info.Reason = "🟢 ৩টি+ জয় স্ট্রিক - রিস্ক বাড়ানো হয়েছে"
	case info.WinStreak >= 2:
		// 2 wins - slight increase
		adjustedSize = math.Min(baseSize*1.1, 2.5)
		info.Reason = "🟢 ২টি জয় স্ট্রিক"
	default:
		info.Reason = "স্ট্যান্ডার্ড রিস্ক"
	}

	// Additional adjustment based on win rate
	if info.TotalRecent >= 5 {
		if info.RecentWinRate < 40 {
			adjustedSize *= 0.75
			info.Reason += " | Win rate কম"
		} else if info.RecentWinRate > 60 {
			adjustedSize = math.Min(adjustedSize*1.1, 3.0)
			info.Reason += " | Win rate ভালো"
		}
	}

	// Ensure bounds
	info.RecommendedSize = math.Max(0.5, math.Min(adjustedSize, 3.0))

	log.Printf("📊 [RiskManager] Win Streak: %d, Lose Streak: %d, Win Rate: %.1f%%, Size: %.2f%%",
		info.WinStreak, info.LoseStreak, info.RecentWinRate, info.RecommendedSize)

	return info
}

// getRecentClosedTrades fetches last N closed trades
func (rm *RiskManager) getRecentClosedTrades(limit int) []model.Signal {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"status": "CLOSED",
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "closed_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := rm.collection.Find(ctx, filter, opts)
	if err != nil {
		log.Printf("⚠️ [RiskManager] Failed to fetch recent trades: %v", err)
		return nil
	}
	defer cursor.Close(ctx)

	var signals []model.Signal
	if err := cursor.All(ctx, &signals); err != nil {
		log.Printf("⚠️ [RiskManager] Failed to decode trades: %v", err)
		return nil
	}

	return signals
}

// GetTodayStats returns today's trading statistics
func (rm *RiskManager) GetTodayStats() (wins, losses int, totalPnL float64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	today := time.Now().Truncate(24 * time.Hour)
	filter := bson.M{
		"status":    "CLOSED",
		"closed_at": bson.M{"$gte": today},
	}

	cursor, err := rm.collection.Find(ctx, filter)
	if err != nil {
		return 0, 0, 0
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var signal model.Signal
		if err := cursor.Decode(&signal); err == nil {
			totalPnL += signal.PnL
			if signal.PnL > 0 {
				wins++
			} else {
				losses++
			}
		}
	}

	return wins, losses, math.Round(totalPnL*100) / 100
}
