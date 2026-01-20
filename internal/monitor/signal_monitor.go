package monitor

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"my-tool-go/internal/model"
	"my-tool-go/internal/service"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type SignalMonitor struct {
	collection *mongo.Collection
	binance    *service.BinanceService
	telegram   *service.TelegramService
}

func NewSignalMonitor(db *mongo.Database, binance *service.BinanceService, telegram *service.TelegramService) *SignalMonitor {
	return &SignalMonitor{
		collection: db.Collection("signals"),
		binance:    binance,
		telegram:   telegram,
	}
}

// MonitorActiveSignals checks all active signals for TP/SL hits and reversals
func (sm *SignalMonitor) MonitorActiveSignals() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Find all active signals
	cursor, err := sm.collection.Find(ctx, bson.M{"status": "ACTIVE"})
	if err != nil {
		log.Printf("❌ Failed to fetch active signals: %v", err)
		return
	}
	defer cursor.Close(ctx)

	var signals []model.Signal
	if err := cursor.All(ctx, &signals); err != nil {
		log.Printf("❌ Failed to decode signals: %v", err)
		return
	}

	if len(signals) == 0 {
		return // No active signals to monitor
	}

	log.Printf("👀 Monitoring %d active signals...", len(signals))

	// Check each signal
	for _, signal := range signals {
		sm.checkSignal(&signal)
	}
}

// checkSignal checks individual signal for TP/SL/Reversal
func (sm *SignalMonitor) checkSignal(signal *model.Signal) {
	// Fetch current price from Binance
	klines, err := sm.binance.GetKlines(signal.Symbol, "1m", 1)
	if err != nil {
		log.Printf("⚠️  Failed to fetch price for %s: %v", signal.Symbol, err)
		return
	}

	if len(klines) == 0 {
		return
	}

	currentPrice := klines[0].Close

	// Check TP/SL based on signal type
	if signal.Type == model.SignalTypeLong {
		sm.checkLongSignal(signal, currentPrice)
	} else {
		sm.checkShortSignal(signal, currentPrice)
	}
}

// checkLongSignal checks LONG signal conditions
func (sm *SignalMonitor) checkLongSignal(signal *model.Signal, currentPrice float64) {
	// Take Profit Hit
	if currentPrice >= signal.TakeProfit {
		pnl := ((currentPrice - signal.EntryPrice) / signal.EntryPrice) * 100
		sm.closeSignal(signal, "TP_HIT", currentPrice, pnl)
		sm.sendTPAlert(signal, currentPrice, pnl)
		return
	}

	// Stop Loss Hit
	if currentPrice <= signal.StopLoss {
		pnl := ((currentPrice - signal.EntryPrice) / signal.EntryPrice) * 100
		sm.closeSignal(signal, "SL_HIT", currentPrice, pnl)
		sm.sendSLAlert(signal, currentPrice, pnl)
		return
	}

	// Quick Reversal Detection (price dropped 1% from entry within 5 min)
	if signal.Timestamp.Add(5 * time.Minute).After(time.Now()) {
		drop := ((signal.EntryPrice - currentPrice) / signal.EntryPrice) * 100
		if drop >= 1.0 && drop < 2.0 { // Between 1-2% (before SL)
			sm.sendReversalWarning(signal, currentPrice, drop)
		}
	}

	// Trailing Stop Recommendation (price up 3%+)
	profit := ((currentPrice - signal.EntryPrice) / signal.EntryPrice) * 100
	if profit >= 3.0 && profit < 6.0 {
		sm.sendTrailingStopSuggestion(signal, currentPrice, profit)
	}
}

// checkShortSignal checks SHORT signal conditions
func (sm *SignalMonitor) checkShortSignal(signal *model.Signal, currentPrice float64) {
	// Take Profit Hit (price goes down)
	if currentPrice <= signal.TakeProfit {
		pnl := ((signal.EntryPrice - currentPrice) / signal.EntryPrice) * 100
		sm.closeSignal(signal, "TP_HIT", currentPrice, pnl)
		sm.sendTPAlert(signal, currentPrice, pnl)
		return
	}

	// Stop Loss Hit (price goes up)
	if currentPrice >= signal.StopLoss {
		pnl := ((signal.EntryPrice - currentPrice) / signal.EntryPrice) * 100
		sm.closeSignal(signal, "SL_HIT", currentPrice, pnl)
		sm.sendSLAlert(signal, currentPrice, pnl)
		return
	}

	// Quick Reversal Detection
	if signal.Timestamp.Add(5 * time.Minute).After(time.Now()) {
		rise := ((currentPrice - signal.EntryPrice) / signal.EntryPrice) * 100
		if rise >= 1.0 && rise < 2.0 {
			sm.sendReversalWarning(signal, currentPrice, rise)
		}
	}

	// Trailing Stop Recommendation
	profit := ((signal.EntryPrice - currentPrice) / signal.EntryPrice) * 100
	if profit >= 3.0 && profit < 6.0 {
		sm.sendTrailingStopSuggestion(signal, currentPrice, profit)
	}
}

// closeSignal updates signal status in MongoDB
func (sm *SignalMonitor) closeSignal(signal *model.Signal, reason string, exitPrice, pnl float64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"status":       "CLOSED",
			"close_reason": reason,
			"closed_at":    now,
			"pnl":          pnl,
		},
	}

	_, err := sm.collection.UpdateOne(
		ctx,
		bson.M{"symbol": signal.Symbol, "status": "ACTIVE", "timestamp": signal.Timestamp},
		update,
	)

	if err != nil {
		log.Printf("❌ Failed to close signal %s: %v", signal.Symbol, err)
		return
	}

	log.Printf("🔒 Closed %s signal: %s (PnL: %.2f%%)", signal.Symbol, reason, pnl)
}

// sendTPAlert sends Take Profit hit notification in Bangla
func (sm *SignalMonitor) sendTPAlert(signal *model.Signal, exitPrice, pnl float64) {
	emoji := "🎯"
	if pnl > 10 {
		emoji = "🚀💰"
	}

	message := fmt.Sprintf(`%s <b>টেক প্রফিট হিট!</b>

<b>সিম্বল:</b> %s
<b>টাইপ:</b> %s
<b>টায়ার:</b> %s

<b>💰 এন্ট্রি:</b> %s
<b>🎯 এক্সিট:</b> %s
<b>💵 টার্গেট:</b> %s

<b>📈 প্রফিট:</b> +%.2f%%

🎉 <b>অভিনন্দন!</b> আপনার ট্রেড সফল হয়েছে।
এখন ৫০%% বিক্রি করুন এবং বাকি ৫০%% trailing stop দিয়ে রাখুন।
`,
		emoji,
		signal.Symbol,
		signal.Type,
		signal.Tier,
		formatPrice(signal.EntryPrice),
		formatPrice(exitPrice),
		formatPrice(signal.TakeProfit),
		pnl,
	)

	sm.telegram.SendMessage(message)
}

// sendSLAlert sends Stop Loss hit notification in Bangla
func (sm *SignalMonitor) sendSLAlert(signal *model.Signal, exitPrice, pnl float64) {
	message := fmt.Sprintf(`🛑 <b>স্টপ লস হিট!</b>

<b>সিম্বল:</b> %s
<b>টাইপ:</b> %s
<b>টায়ার:</b> %s

<b>💰 এন্ট্রি:</b> %s
<b>🛑 এক্সিট:</b> %s
<b>⛔ স্টপ লস:</b> %s

<b>📉 লস:</b> %.2f%%

💡 <b>শিক্ষা:</b> স্টপ লস মেনে চলাই স্মার্ট ট্রেডিং।
পরবর্তী সিগন্যালের জন্য অপেক্ষা করুন।
`,
		signal.Symbol,
		signal.Type,
		signal.Tier,
		formatPrice(signal.EntryPrice),
		formatPrice(exitPrice),
		formatPrice(signal.StopLoss),
		pnl,
	)

	sm.telegram.SendMessage(message)
}

// sendReversalWarning sends quick reversal warning
func (sm *SignalMonitor) sendReversalWarning(signal *model.Signal, currentPrice, movePercent float64) {
	message := fmt.Sprintf(`⚠️ <b>সতর্কতা: এন্ট্রি রিভার্স হচ্ছে!</b>

<b>সিম্বল:</b> %s
<b>এন্ট্রি প্রাইস:</b> %s
<b>বর্তমান প্রাইস:</b> %s
<b>মুভমেন্ট:</b> %.2f%% (বিপরীত দিকে)

⚠️ প্রাইস এন্ট্রি থেকে দ্রুত নিচে নামছে।
বিবেচনা করুন:
• ট্রেড ক্লোজ করুন যদি breakdown নিশ্চিত হয়
• অথবা break-even এ স্টপ লস মুভ করুন

<b>স্টপ লস:</b> %s
`,
		signal.Symbol,
		formatPrice(signal.EntryPrice),
		formatPrice(currentPrice),
		movePercent,
		formatPrice(signal.StopLoss),
	)

	sm.telegram.SendMessage(message)
	log.Printf("⚠️  Reversal warning sent for %s (%.2f%%)", signal.Symbol, movePercent)
}

// sendTrailingStopSuggestion sends trailing stop suggestion
func (sm *SignalMonitor) sendTrailingStopSuggestion(signal *model.Signal, currentPrice, profit float64) {
	newStopLoss := signal.EntryPrice // Break-even
	if profit >= 4.0 {
		newStopLoss = signal.EntryPrice * 1.02 // 2% profit locked
	}

	message := fmt.Sprintf(`📊 <b>ট্রেইলিং স্টপ সাজেশন</b>

<b>সিম্বল:</b> %s
<b>বর্তমান প্রফিট:</b> +%.2f%%

💡 <b>সাজেশন:</b> 
আপনার স্টপ লস মুভ করুন:
<b>পুরোনো SL:</b> %s
<b>নতুন SL:</b> %s (Break-even)

এভাবে প্রফিট protect করুন!
`,
		signal.Symbol,
		profit,
		formatPrice(signal.StopLoss),
		formatPrice(newStopLoss),
	)

	sm.telegram.SendMessage(message)
	log.Printf("💡 Trailing stop suggestion sent for %s (+%.2f%%)", signal.Symbol, profit)
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

// GetActiveSignalsCount returns count of active signals
func (sm *SignalMonitor) GetActiveSignalsCount() int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := sm.collection.CountDocuments(ctx, bson.M{"status": "ACTIVE"})
	if err != nil {
		return 0
	}

	return int(count)
}

// GetTodayPnL calculates today's total PnL
func (sm *SignalMonitor) GetTodayPnL() float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	today := time.Now().Truncate(24 * time.Hour)
	cursor, err := sm.collection.Find(ctx, bson.M{
		"status":     "CLOSED",
		"created_at": bson.M{"$gte": today},
	})

	if err != nil {
		return 0
	}
	defer cursor.Close(ctx)

	totalPnL := 0.0
	for cursor.Next(ctx) {
		var signal model.Signal
		if err := cursor.Decode(&signal); err == nil {
			totalPnL += signal.PnL
		}
	}

	return math.Round(totalPnL*100) / 100
}
