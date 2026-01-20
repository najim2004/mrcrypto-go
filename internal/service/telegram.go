package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"my-tool-go/internal/config"
	"my-tool-go/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TelegramService struct {
	bot        *tgbotapi.BotAPI
	chatID     int64
	collection *mongo.Collection
}

func NewTelegramService() (*TelegramService, error) {
	bot, err := tgbotapi.NewBotAPI(config.AppConfig.TelegramBotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	// Parse chat ID
	var chatID int64
	fmt.Sscanf(config.AppConfig.TelegramChatID, "%d", &chatID)

	log.Println("✅ Telegram bot authorized:", bot.Self.UserName)

	// Connect to MongoDB for /today command
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.AppConfig.MongoURI))
	if err != nil {
		log.Printf("⚠️  MongoDB connection for Telegram failed: %v", err)
	}

	collection := client.Database("trading-signals").Collection("signals")

	service := &TelegramService{
		bot:        bot,
		chatID:     chatID,
		collection: collection,
	}

	// Start command handler in background
	go service.handleCommands()

	return service, nil
}

// handleCommands listens for and handles telegram commands
func (s *TelegramService) handleCommands() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := s.bot.GetUpdatesChan(u)

	log.Println("✅ Telegram command handler started")

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if !update.Message.IsCommand() {
			continue
		}

		command := update.Message.Command()
		chatID := update.Message.Chat.ID

		switch command {
		case "start":
			s.handleStart(chatID)
		case "status":
			s.handleStatus(chatID)
		case "today":
			s.handleToday(chatID)
		case "help":
			s.handleHelp(chatID)
		default:
			msg := tgbotapi.NewMessage(chatID, "Unknown command. Use /help to see available commands.")
			s.bot.Send(msg)
		}
	}
}

func (s *TelegramService) handleStart(chatID int64) {
	message := `🚀 *Crypto Signal Generator Bot*

I provide 1-3 high-confidence crypto trading signals daily.

*Available Commands:*
/status - Bot and system status
/today - View today's signals
/help - Help and information`

	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "Markdown"
	s.bot.Send(msg)
	log.Println("📱 /start command executed")
}

func (s *TelegramService) handleStatus(chatID int64) {
	message := `📊 *System Status*

✅ Bot: Active
✅ Database: Connected
✅ AI Models: Multi-model fallback ready
⏱️ Polling Interval: 1 minute
🎯 Min Score Threshold: 70/100

*Models:*
• gemini-3-pro-preview
• gemini-3-flash-preview
• gemini-2.5-flash
• gemini-2.5-flash-lite
• gemini-2.5-pro

System running normally 🚀`

	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "Markdown"
	s.bot.Send(msg)
	log.Println("📱 /status command executed")
}

func (s *TelegramService) handleToday(chatID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get today's signals
	today := time.Now().Truncate(24 * time.Hour)

	filter := bson.M{
		"created_at": bson.M{"$gte": today},
	}

	cursor, err := s.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "❌ Error fetching today's signals")
		s.bot.Send(msg)
		return
	}
	defer cursor.Close(ctx)

	var signals []model.Signal
	if err = cursor.All(ctx, &signals); err != nil {
		msg := tgbotapi.NewMessage(chatID, "❌ Error processing signals")
		s.bot.Send(msg)
		return
	}

	if len(signals) == 0 {
		msg := tgbotapi.NewMessage(chatID, "📭 No signals generated today yet.")
		s.bot.Send(msg)
		return
	}

	message := fmt.Sprintf("📅 *Today's Signals (%d)*\n\n", len(signals))
	for idx, sig := range signals {
		message += fmt.Sprintf("%d. %s %s\n", idx+1, sig.Type, sig.Symbol)
		message += fmt.Sprintf("   Entry: %s\n", FormatPrice(sig.EntryPrice))
		message += fmt.Sprintf("   Score: %d/100\n", sig.AIScore)
		message += fmt.Sprintf("   Tier: %s\n\n", sig.Tier)
	}

	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "Markdown"
	s.bot.Send(msg)
	log.Println("📱 /today command executed")
}

func (s *TelegramService) handleHelp(chatID int64) {
	message := `ℹ️ *Help & Information*

*What does this bot do?*
I analyze Binance crypto markets and generate high-probability trading signals using technical indicators + AI.

*Strategy:*
• Multi-timeframe: 4h + 1h + 15m + 5m
• Indicators: RSI, ADX, MACD, VWAP, Bollinger Bands
• AI Scoring: Gemini models (multi-fallback)
• Risk: 3:1 R:R ratio (6% TP, 2% SL)

*Commands:*
/start - Start the bot
/status - System status
/today - Today's signals
/help - This help message

⚠️ *Disclaimer:* Trading is risky. Always use proper risk management.`

	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "Markdown"
	s.bot.Send(msg)
	log.Println("📱 /help command executed")
}

// SendSignal sends a trading signal notification to Telegram
func (s *TelegramService) SendSignal(signal *model.Signal) error {
	message := formatSignalMessage(signal)

	msg := tgbotapi.NewMessage(s.chatID, message)
	msg.ParseMode = "HTML"

	_, err := s.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}

	log.Printf("📲 Telegram notification sent for %s", signal.Symbol)
	return nil
}

// escapeHTML escapes HTML special characters for Telegram
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// formatSignalMessage creates a formatted message for Telegram in Bangla with trading guidance
func formatSignalMessage(signal *model.Signal) string {
	// Emoji based on signal type and tier
	var emoji, tierBadge string
	if signal.Type == model.SignalTypeLong {
		emoji = "🟢"
	} else {
		emoji = "🔴"
	}

	if signal.Tier == model.TierPremium {
		tierBadge = "🔥 প্রিমিয়াম"
	} else {
		tierBadge = "✅ স্ট্যান্ডার্ড"
	}

	// Calculate risk/reward ratio
	risk := calculatePercentChange(signal.EntryPrice, signal.StopLoss)
	reward := calculatePercentChange(signal.EntryPrice, signal.TakeProfit)
	rrRatio := reward / (-risk)

	// Position size recommendation (based on 2% risk rule)
	recommendedPosition := 2.0 / (-risk)

	// Escape AI reason to prevent HTML parsing issues
	aiReason := escapeHTML(signal.AIReason)

	message := fmt.Sprintf(`%s <b>%s সিগন্যাল - %s</b>

━━━━━━━━━━━━━━━━━━
<b>📌 ট্রেড তথ্য</b>
━━━━━━━━━━━━━━━━━━

<b>সিম্বল:</b> %s
<b>টাইপ:</b> %s
<b>মার্কেট রেজিম:</b> %s
<b>টায়ার:</b> %s

━━━━━━━━━━━━━━━━━━
<b>💰 প্রাইস লেভেল</b>
━━━━━━━━━━━━━━━━━━

<b>এন্ট্রি:</b> %s
<b>স্টপ লস:</b> %s (%.2f%%)
<b>টেক প্রফিট:</b> %s (+%.2f%%)

<b>⚖️ রিস্ক/রিওয়ার্ড:</b> 1:%.1f
<b>📊 সম্ভাব্য লস:</b> %.2f%%
<b>📈 সম্ভাব্য প্রফিট:</b> +%.2f%%

━━━━━━━━━━━━━━━━━━
<b>📊 টেকনিক্যাল বিশ্লেষণ</b>
━━━━━━━━━━━━━━━━━━

• <b>RSI (1h/5m):</b> %.1f / %.1f
• <b>ADX (1h):</b> %.1f
• <b>ভলিউম:</b> %.2fx গড়
• <b>MACD:</b> %.6f

<b>🤖 AI স্কোর:</b> %d/100
<b>💭 AI মতামত:</b> %s

━━━━━━━━━━━━━━━━━━
<b>💡 ট্রেডিং গাইড</b>
━━━━━━━━━━━━━━━━━━

<b>১. পজিশন সাইজ:</b>
   আপনার মোট ক্যাপিটালের %.1f%% ব্যবহার করুন
   (২%% রিস্ক রুল অনুযায়ী)

<b>২. এন্ট্রি স্ট্র্যাটেজি:</b>
   • এন্ট্রি প্রাইসের কাছে অপেক্ষা করুন
   • একবারে সব না কিনে ২-৩ ভাগে কিনুন
   • ভলিউম বেশি থাকলে এন্ট্রি নিন

<b>৩. এক্সিট স্ট্র্যাটেজি:</b>
   • টেক প্রফিটে ৫০%% বিক্রি করুন
   • বাকি ৫০%% trailing stop দিয়ে রাখুন
   • স্টপ লস অবশ্যই মেনে চলুন

<b>৪. রিস্ক ম্যানেজমেন্ট:</b>
   • কখনো স্টপ লস মুভ করবেন না
   • একাধিক ট্রেড একসাথে নেবেন না
   • প্রতি ট্রেডে সর্বোচ্চ ২-৩%% রিস্ক নিন

━━━━━━━━━━━━━━━━━━
⚠️ <b>সতর্কতা:</b> ট্রেডিং ঝুঁকিপূর্ণ। 
শুধুমাত্র সেই টাকা ব্যবহার করুন যা হারাতে পারবেন।
━━━━━━━━━━━━━━━━━━
`,
		emoji,
		signal.Type,
		tierBadge,
		signal.Symbol,
		signal.Type,
		signal.Regime,
		tierBadge,
		FormatPrice(signal.EntryPrice),
		FormatPrice(signal.StopLoss),
		risk,
		FormatPrice(signal.TakeProfit),
		reward,
		rrRatio,
		-risk,
		reward,
		signal.TechnicalContext.RSI1h,
		signal.TechnicalContext.RSI5m,
		signal.TechnicalContext.ADX1h,
		signal.TechnicalContext.CurrentVol/signal.TechnicalContext.AvgVol,
		signal.TechnicalContext.Histogram,
		signal.AIScore,
		aiReason,
		recommendedPosition,
	)

	return message
}

// calculatePercentChange calculates percentage change between two prices
func calculatePercentChange(from, to float64) float64 {
	return ((to - from) / from) * 100
}
