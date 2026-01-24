package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"mrcrypto-go/internal/config"
	"mrcrypto-go/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TelegramService struct {
	bot        *tgbotapi.BotAPI
	chatID     int64
	collection *mongo.Collection
	binance    *BinanceService
}

func NewTelegramService(binanceService *BinanceService) (*TelegramService, error) {
	bot, err := tgbotapi.NewBotAPI(config.AppConfig.TelegramBotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	log.Printf("✅ Telegram bot authorized: %s", bot.Self.UserName)

	// MongoDB connection for /today command
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(config.AppConfig.MongoURI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	collection := client.Database("mrcrypto").Collection("signals")

	service := &TelegramService{
		bot:        bot,
		chatID:     parseChatID(config.AppConfig.TelegramChatID),
		collection: collection,
		binance:    binanceService,
	}

	// Start command handler in background
	go service.handleCommands()
	log.Println("✅ Telegram command handler started")

	return service, nil
}

// handleCommands listens for and processes Telegram commands
func (s *TelegramService) handleCommands() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := s.bot.GetUpdatesChan(u)

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
			log.Println("📱 /start command executed")
			s.handleStart(chatID)
		case "status":
			log.Println("📱 /status command executed")
			s.handleStatus(chatID)
		case "today":
			log.Println("📱 /today command executed")
			s.handleToday(chatID)
		case "help":
			log.Println("📱 /help command executed")
			s.handleHelp(chatID)
		case "active":
			log.Println("📱 /active command executed")
			s.handleActive(update.Message)
		case "pnl":
			log.Println("📱 /pnl command executed")
			s.handlePnL(update.Message)
		case "stats":
			log.Println("📱 /stats command executed")
			s.handleStats(update.Message)
		case "closed":
			log.Println("📱 /closed command executed")
			s.handleClosed(update.Message)
		case "price":
			log.Println("📱 /price command executed")
			s.handlePrice(update.Message)
		default:
			msg := tgbotapi.NewMessage(chatID, "Unknown command. Use /help to see available commands.")
			s.bot.Send(msg)
		}
	}
}

// handleStart sends welcome message
func (s *TelegramService) handleStart(chatID int64) {
	message := `🚀 <b>Welcome to MrCrypto Trading Bot!</b>

আমি আপনার জন্য প্রিমিয়াম ট্রেডিং সিগন্যাল generate করি।

<b>Features:</b>
✅ AI-powered signal validation
✅ Multi-timeframe analysis
✅ Real-time market monitoring
✅ Bangla notifications

<b>Commands:</b>
/help - সব command দেখুন
/active - Active signals
/stats - Performance stats

শুভকামনা! 🎯`

	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "HTML"
	s.bot.Send(msg)
}

// handleStatus sends bot status
func (s *TelegramService) handleStatus(chatID int64) {
	message := `✅ <b>Bot Status</b>

🟢 <b>Status:</b> Online
🤖 <b>AI Models:</b> Active
📊 <b>Market Monitoring:</b> Live
⏰ <b>Polling:</b> Every 1 minute

সব কিছু ঠিকঠাক চলছে! 🚀`

	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "HTML"
	s.bot.Send(msg)
}

// handleToday sends today's signals
func (s *TelegramService) handleToday(chatID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	today := time.Now().Truncate(24 * time.Hour)
	cursor, err := s.collection.Find(ctx, bson.M{
		"created_at": bson.M{"$gte": today},
	})

	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "❌ Failed to fetch today's signals")
		s.bot.Send(msg)
		return
	}
	defer cursor.Close(ctx)

	var signals []model.Signal
	if err := cursor.All(ctx, &signals); err != nil {
		msg := tgbotapi.NewMessage(chatID, "❌ Failed to decode signals")
		s.bot.Send(msg)
		return
	}

	if len(signals) == 0 {
		msg := tgbotapi.NewMessage(chatID, "📅 আজ এখনো কোন signal generate হয়নি।")
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
}

func (s *TelegramService) handleHelp(chatID int64) {
	message := `🤖 <b>MrCrypto Bot - Help</b>

<b>📊 Signal Commands:</b>
/active - সব active signals দেখুন
/closed - Recently closed signals
/pnl - Profit &amp; Loss summary
/stats - Performance statistics
/price SYMBOL - Current price check

<b>📈 Info Commands:</b>
/status - Bot status
/today - আজকের signals

<b>❓ Help:</b>
/start - Welcome message
/help - এই help message

💡 <b>Tips:</b>
• প্রতিটি signal এ trading guide দেওয়া আছে
• Risk management অবশ্যই মানুন
• Stop loss কখনো মুভ করবেন না

যেকোনো সমস্যায় support এ যোগাযোগ করুন।`

	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "HTML"
	s.bot.Send(msg)
}

// handleActive shows all active signals
func (s *TelegramService) handleActive(msg *tgbotapi.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := s.collection.Find(ctx, bson.M{"status": "ACTIVE"})
	if err != nil {
		s.sendMessage(msg.Chat.ID, "❌ Failed to fetch active signals")
		return
	}
	defer cursor.Close(ctx)

	var signals []model.Signal
	if err := cursor.All(ctx, &signals); err != nil {
		s.sendMessage(msg.Chat.ID, "❌ Failed to decode signals")
		return
	}

	if len(signals) == 0 {
		message := `📊 <b>Active Signals</b>

কোন active signal নেই।
নতুন signal এর জন্য অপেক্ষা করুন।`
		s.sendMessage(msg.Chat.ID, message)
		return
	}

	message := fmt.Sprintf("<b>📊 Active Signals (%d)</b>\n\n", len(signals))

	for i, signal := range signals {
		emoji := "🟢"
		if signal.Type == model.SignalTypeShort {
			emoji = "🔴"
		}

		message += fmt.Sprintf(`%s <b>%s - %s</b>
Entry: %s
TP: %s | SL: %s
⏰ %s

`, emoji, signal.Symbol, signal.Type,
			FormatPrice(signal.EntryPrice),
			FormatPrice(signal.TakeProfit),
			FormatPrice(signal.StopLoss),
			signal.Timestamp.Format("15:04, 02 Jan"))

		if i >= 9 { // Limit to 10 signals
			message += fmt.Sprintf("\n... এবং আরো %d টি signal", len(signals)-10)
			break
		}
	}

	s.sendMessage(msg.Chat.ID, message)
}

// handlePnL shows PnL summary
func (s *TelegramService) handlePnL(msg *tgbotapi.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Today's PnL
	today := time.Now().Truncate(24 * time.Hour)
	todayCursor, _ := s.collection.Find(ctx, bson.M{
		"status":     "CLOSED",
		"created_at": bson.M{"$gte": today},
	})
	defer todayCursor.Close(ctx)

	var todaySignals []model.Signal
	todayCursor.All(ctx, &todaySignals)

	todayPnL, todayWins, todayLosses := 0.0, 0, 0
	for _, sig := range todaySignals {
		todayPnL += sig.PnL
		if sig.PnL > 0 {
			todayWins++
		} else {
			todayLosses++
		}
	}

	// This week
	weekStart := time.Now().AddDate(0, 0, -7)
	weekCursor, _ := s.collection.Find(ctx, bson.M{
		"status":     "CLOSED",
		"created_at": bson.M{"$gte": weekStart},
	})
	defer weekCursor.Close(ctx)

	var weekSignals []model.Signal
	weekCursor.All(ctx, &weekSignals)

	weekPnL := 0.0
	for _, sig := range weekSignals {
		weekPnL += sig.PnL
	}

	winRate := 0.0
	totalTrades := todayWins + todayLosses
	if totalTrades > 0 {
		winRate = (float64(todayWins) / float64(totalTrades)) * 100
	}

	message := fmt.Sprintf(`💰 <b>Profit &amp; Loss Summary</b>

📅 <b>Today:</b> %s%.2f%% (%d trades)
  ✅ Wins: %d
  ❌ Losses: %d
  📊 Win Rate: %.1f%%

📅 <b>This Week:</b> %s%.2f%% (%d trades)

💡 আপনার পারফরম্যান্স দেখতে /stats ব্যবহার করুন
`,
		getPnLEmoji(todayPnL), todayPnL, len(todaySignals),
		todayWins, todayLosses, winRate,
		getPnLEmoji(weekPnL), weekPnL, len(weekSignals))

	s.sendMessage(msg.Chat.ID, message)
}

// handleStats shows performance statistics
func (s *TelegramService) handleStats(msg *tgbotapi.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, _ := s.collection.Find(ctx, bson.M{"status": "CLOSED"})
	defer cursor.Close(ctx)

	var allSignals []model.Signal
	cursor.All(ctx, &allSignals)

	if len(allSignals) == 0 {
		s.sendMessage(msg.Chat.ID, "📊 এখনো কোন closed signal নেই।")
		return
	}

	wins, losses := 0, 0
	totalWinPnL, totalLossPnL := 0.0, 0.0
	bestTrade, worstTrade := model.Signal{PnL: -999}, model.Signal{PnL: 999}

	for _, sig := range allSignals {
		if sig.PnL > 0 {
			wins++
			totalWinPnL += sig.PnL
			if sig.PnL > bestTrade.PnL {
				bestTrade = sig
			}
		} else {
			losses++
			totalLossPnL += sig.PnL
			if sig.PnL < worstTrade.PnL {
				worstTrade = sig
			}
		}
	}

	totalTrades := wins + losses
	winRate := (float64(wins) / float64(totalTrades)) * 100
	avgWin := 0.0
	avgLoss := 0.0
	if wins > 0 {
		avgWin = totalWinPnL / float64(wins)
	}
	if losses > 0 {
		avgLoss = totalLossPnL / float64(losses)
	}

	profitFactor := 0.0
	if totalLossPnL != 0 {
		profitFactor = -totalWinPnL / totalLossPnL
	}

	message := fmt.Sprintf(`📊 <b>Performance Statistics</b>

🎯 <b>Win Rate:</b> %.1f%% (%d/%d)
💎 <b>Profit Factor:</b> %.2f
📈 <b>Average Win:</b> +%.2f%%
📉 <b>Average Loss:</b> %.2f%%

🏆 <b>Best Trade:</b> +%.2f%% (%s)
💀 <b>Worst Trade:</b> %.2f%% (%s)

📊 <b>Total Trades:</b> %d
✅ <b>Wins:</b> %d
❌ <b>Losses:</b> %d
`,
		winRate, wins, totalTrades,
		profitFactor,
		avgWin,
		avgLoss,
		bestTrade.PnL, bestTrade.Symbol,
		worstTrade.PnL, worstTrade.Symbol,
		totalTrades, wins, losses)

	s.sendMessage(msg.Chat.ID, message)
}

// handleClosed shows recently closed signals
func (s *TelegramService) handleClosed(msg *tgbotapi.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Last 24 hours
	yesterday := time.Now().AddDate(0, 0, -1)
	opts := options.Find().SetSort(bson.D{{Key: "closed_at", Value: -1}}).SetLimit(10)

	cursor, err := s.collection.Find(ctx, bson.M{
		"status":    "CLOSED",
		"closed_at": bson.M{"$gte": yesterday},
	}, opts)

	if err != nil {
		s.sendMessage(msg.Chat.ID, "❌ Failed to fetch closed signals")
		return
	}
	defer cursor.Close(ctx)

	var signals []model.Signal
	cursor.All(ctx, &signals)

	if len(signals) == 0 {
		s.sendMessage(msg.Chat.ID, "📊 গত ২৪ ঘণ্টায় কোন signal close হয়নি।")
		return
	}

	message := fmt.Sprintf("<b>📊 Recently Closed Signals (%d)</b>\n\n", len(signals))

	for _, signal := range signals {
		emoji := "✅"
		if signal.PnL < 0 {
			emoji = "❌"
		}

		reasonEmoji := ""
		switch signal.CloseReason {
		case "TP_HIT":
			reasonEmoji = "🎯"
		case "SL_HIT":
			reasonEmoji = "🛑"
		}

		closedTime := time.Now()
		if signal.ClosedAt != nil {
			closedTime = *signal.ClosedAt
		}

		message += fmt.Sprintf(`%s <b>%s</b> %s%s
PnL: %s%.2f%% | Reason: %s
⏰ %s

`, emoji, signal.Symbol, signal.Type, reasonEmoji,
			getPnLSign(signal.PnL), signal.PnL, signal.CloseReason,
			closedTime.Format("15:04, 02 Jan"))
	}

	s.sendMessage(msg.Chat.ID, message)
}

// handlePrice shows current price for a symbol
func (s *TelegramService) handlePrice(msg *tgbotapi.Message) {
	// Extract symbol from command (e.g., "/price BTCUSDT")
	parts := strings.Fields(msg.Text)
	if len(parts) < 2 {
		s.sendMessage(msg.Chat.ID, `💡 <b>Usage:</b> /price BTCUSDT

Example: /price ETHUSDT`)
		return
	}

	symbol := strings.ToUpper(parts[1])

	// Fetch current 1m kline data
	klines, err := s.binance.GetKlines(symbol, "1m", 1)
	if err != nil {
		s.sendMessage(msg.Chat.ID, fmt.Sprintf(`❌ <b>Error</b>

Failed to fetch price for %s
Symbol টি সঠিক আছে কিনা চেক করুন।`, symbol))
		return
	}

	if len(klines) == 0 {
		s.sendMessage(msg.Chat.ID, fmt.Sprintf(`❌ <b>No Data</b>

%s এর জন্য কোন data পাওয়া যায়নি।`, symbol))
		return
	}

	currentPrice := klines[0].Close
	openPrice := klines[0].Open
	highPrice := klines[0].High
	lowPrice := klines[0].Low
	volume := klines[0].Volume

	// Fetch 24h kline to calculate 24h change
	klines24h, err := s.binance.GetKlines(symbol, "1d", 1)
	change24h := 0.0
	if err == nil && len(klines24h) > 0 {
		price24hAgo := klines24h[0].Open
		change24h = ((currentPrice - price24hAgo) / price24hAgo) * 100
	}

	// Determine emoji based on 24h change
	changeEmoji := "📊"
	if change24h > 0 {
		changeEmoji = "📈"
	} else if change24h < 0 {
		changeEmoji = "📉"
	}

	message := fmt.Sprintf(`💰 <b>%s Price</b>

<b>Current Price:</b> %s
<b>24h Change:</b> %s%.2f%%

━━━━━━━━━━━━━━━━━━
<b>📊 1min Candle:</b>
• Open: %s
• High: %s
• Low: %s
• Volume: %.2f

<b>Last Update:</b> %s
`,
		symbol,
		FormatPrice(currentPrice),
		changeEmoji, change24h,
		FormatPrice(openPrice),
		FormatPrice(highPrice),
		FormatPrice(lowPrice),
		volume,
		time.Now().Format("15:04:05"))

	s.sendMessage(msg.Chat.ID, message)
}

// Helper functions
func getPnLEmoji(pnl float64) string {
	if pnl > 0 {
		return "🟢 +"
	}
	return "🔴 "
}

func getPnLSign(pnl float64) string {
	if pnl > 0 {
		return "+"
	}
	return ""
}

func (s *TelegramService) sendMessage(chatID int64, message string) {
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ParseMode = "HTML"
	s.bot.Send(msg)
}

// SendSignal sends a trading signal notification to Telegram
func (s *TelegramService) SendSignal(signal *model.Signal) error {
	log.Printf("📤 [Telegram] Sending signal notification for %s...", signal.Symbol)
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

// SendMessage sends a generic message to Telegram
func (s *TelegramService) SendMessage(message string) error {
	msg := tgbotapi.NewMessage(s.chatID, message)
	msg.ParseMode = "HTML"

	_, err := s.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}

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
	var signalEmoji, tierEmoji, directionText string
	if signal.Type == model.SignalTypeLong {
		signalEmoji = "🟢"
		directionText = "লং (BUY)"
	} else {
		signalEmoji = "🔴"
		directionText = "শর্ট (SELL)"
	}

	if signal.Tier == model.TierPremium {
		tierEmoji = "🔥"
	} else {
		tierEmoji = "✅"
	}

	// Calculate volume ratio safely
	volRatio := 0.0
	if signal.TechnicalContext.AvgVol > 0 {
		volRatio = signal.TechnicalContext.CurrentVol / signal.TechnicalContext.AvgVol
	}

	// Regime translation
	regimeText := signal.Regime
	switch signal.Regime {
	case "TRENDING_UP":
		regimeText = "📈 আপট্রেন্ড"
	case "TRENDING_DOWN":
		regimeText = "📉 ডাউনট্রেন্ড"
	case "RANGING":
		regimeText = "↔️ রেঞ্জিং"
	}

	// RSI status
	rsiStatus := "স্বাভাবিক"
	if signal.TechnicalContext.RSI1h > 70 {
		rsiStatus = "⚠️ ওভারবট"
	} else if signal.TechnicalContext.RSI1h < 30 {
		rsiStatus = "⚠️ ওভারসোল্ড"
	} else if signal.TechnicalContext.RSI1h >= 50 && signal.TechnicalContext.RSI1h <= 65 {
		rsiStatus = "✅ বুলিশ জোন"
	} else if signal.TechnicalContext.RSI1h >= 35 && signal.TechnicalContext.RSI1h < 50 {
		rsiStatus = "✅ বিয়ারিশ জোন"
	}

	// ADX status
	adxStatus := "দুর্বল"
	if signal.TechnicalContext.ADX1h >= 30 {
		adxStatus = "🔥 অত্যন্ত শক্তিশালী"
	} else if signal.TechnicalContext.ADX1h >= 25 {
		adxStatus = "💪 শক্তিশালী"
	} else if signal.TechnicalContext.ADX1h >= 20 {
		adxStatus = "✅ মাঝারি"
	}

	// Volume status
	volStatus := "কম"
	if volRatio >= 2.0 {
		volStatus = "🔥 উচ্চ"
	} else if volRatio >= 1.5 {
		volStatus = "✅ ভালো"
	} else if volRatio >= 1.0 {
		volStatus = "স্বাভাবিক"
	}

	// Escape AI reason
	aiReason := escapeHTML(signal.AIReason)

	// Confidence level
	confidenceLevel := "মাঝারি"
	if signal.ConfidenceScore >= 0.75 {
		confidenceLevel = "🔥 উচ্চ"
	} else if signal.ConfidenceScore >= 0.60 {
		confidenceLevel = "✅ ভালো"
	}

	message := fmt.Sprintf(`%s <b>%s সিগন্যাল</b> %s

╔═══════════════════════════════╗
         <b>%s</b>
         <b>%s</b>
╚═══════════════════════════════╝

┏━━━ 💰 <b>ট্রেড সেটআপ</b> (ট্যাপ করে কপি করুন) ━━━┓

🎯 <b>এন্ট্রি:</b>     <code>%s</code>
🛑 <b>স্টপ লস:</b>     <code>%s</code> (−%.2f%%)

🏆 <b>TP 1:</b>      <code>%s</code> (+%.2f%%)
   <i>(৫০%% বুক করুন & SL ব্রেক-ইভেনে)</i>

🏆 <b>TP 2:</b>      <code>%s</code> (+%.2f%%)
   <i>(বাকি ৫০%% বুক করুন)</i>

┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

┏━━━ 📊 <b>রিস্ক ম্যানেজমেন্ট</b> ━━━┓

⚖️ <b>R:R রেশিও:</b>      1:%.1f (সর্বোচ্চ)
🎲 <b>ব্রেক-ইভেন WR:</b>  %.1f%%
💼 <b>পজিশন সাইজ:</b>    %.1f%%

┗━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

┏━━━ 📈 <b>টেকনিক্যাল ডেটা</b> ━━━┓

<b>RSI:</b> 4H: %.1f | 1H: %.1f | 15M: %.1f | 5M: %.1f
       %s

<b>ADX:</b> 4H: %.1f | 1H: %.1f | 15M: %.1f
       %s

<b>ভলিউম:</b> %.2fx গড় (%s)
<b>MACD:</b> %.6f
<b>অর্ডার ফ্লো:</b> %.2f

┗━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

┏━━━ 🎯 <b>কী লেভেল</b> ━━━┓

<b>পিভট:</b>
• R3: <code>%s</code>
• R2: <code>%s</code>
• R1: <code>%s</code>
• PP: <code>%s</code>
• S1: <code>%s</code>
• S2: <code>%s</code>
• S3: <code>%s</code>
• নিকটতম: %s

<b>ফিবোনাচ্চি:</b>
• 38.2%%: <code>%s</code>
• 50.0%%: <code>%s</code>
• 61.8%%: <code>%s</code>
• নিকটতম: %s (%.2f%% দূরে)

┗━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

┏━━━ 🧠 <b>AI বিশ্লেষণ</b> ━━━┓

🎯 <b>কনফ্লুয়েন্স:</b> %d/100
📊 <b>কনফিডেন্স:</b> %.0f%% (%s)
🤖 <b>AI স্কোর:</b> %d/100

💭 %s

┗━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

┏━━━ ⚡ <b>ট্রেডিং গাইড</b> ━━━┓

✅ মার্কেট অর্ডারে এন্ট্রি নিন
✅ স্টপ লস সাথে সাথে সেট করুন
✅ TP 1 হিট করলে ৫০%% প্রফিট বুক করুন
✅ বাকি ৫০%% এর জন্য SL এন্ট্রি প্রাইসে আনুন
✅ TP 2 তে সম্পূর্ণ ক্লোজ করুন

┗━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

<b>মার্কেট:</b> %s
<b>টায়ার:</b> %s %s
<b>সময়:</b> %s

#%s #%s #MrCrypto
`,
		signalEmoji,
		signal.Type,
		tierEmoji,
		signal.Symbol,
		directionText,
		FormatPrice(signal.EntryPrice),
		FormatPrice(signal.StopLoss),
		signal.RiskPercent,
		FormatPrice(signal.TakeProfit1),
		signal.TP1Percent,
		FormatPrice(signal.TakeProfit2),
		signal.TP2Percent,
		signal.RiskRewardRatio,
		signal.BreakEvenWinRate,
		signal.RecommendedSize,
		signal.TechnicalContext.RSI4h,
		signal.TechnicalContext.RSI1h,
		signal.TechnicalContext.RSI15m,
		signal.TechnicalContext.RSI5m,
		rsiStatus,
		signal.TechnicalContext.ADX4h,
		signal.TechnicalContext.ADX1h,
		signal.TechnicalContext.ADX15m,
		adxStatus,
		volRatio,
		volStatus,
		signal.TechnicalContext.Histogram,
		signal.TechnicalContext.OrderFlowDelta,
		FormatPrice(signal.TechnicalContext.PivotR3),
		FormatPrice(signal.TechnicalContext.PivotR2),
		FormatPrice(signal.TechnicalContext.PivotR1),
		FormatPrice(signal.TechnicalContext.PivotPoint),
		FormatPrice(signal.TechnicalContext.PivotS1),
		FormatPrice(signal.TechnicalContext.PivotS2),
		FormatPrice(signal.TechnicalContext.PivotS3),
		signal.TechnicalContext.NearestPivot,
		FormatPrice(signal.TechnicalContext.Fib382),
		FormatPrice(signal.TechnicalContext.Fib500),
		FormatPrice(signal.TechnicalContext.Fib618),
		signal.TechnicalContext.NearestFib,
		signal.NearestLevelDist,
		signal.ConfluenceScore,
		signal.ConfidenceScore*100,
		confidenceLevel,
		signal.AIScore,
		aiReason,
		regimeText,
		tierEmoji,
		signal.Tier,
		signal.Timestamp.Format("15:04, 02 Jan 2006"),
		signal.Symbol,
		signal.Type,
	)

	return message
}

func calculatePercentChange(from, to float64) float64 {
	return ((to - from) / from) * 100
}

// parseChatID converts string chat ID to int64
func parseChatID(chatIDStr string) int64 {
	var chatID int64
	fmt.Sscanf(chatIDStr, "%d", &chatID)
	return chatID
}
