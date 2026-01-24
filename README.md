# my-tool-go

> AI-Powered Crypto Trading Signal Tool written in Golang

Professional cryptocurrency trading signal generator that analyzes market conditions, applies technical indicators, validates signals with AI, and sends notifications via Telegram.

## Features

- 📊 **Multi-Timeframe Analysis**: Analyzes 4h, 1h, 15m, and 5m candlesticks
- 🎯 **Technical Indicators**: RSI, ADX, VWAP, MACD, Bollinger Bands, and more
- 🧠 **AI Validation**: Google Gemini AI validates each signal (minimum 70/100 score)
- ⚡ **Dual-Tier Signals**: PREMIUM (strict criteria) and STANDARD (relaxed)
- 🚫 **Market Regime Detection**: Filters out choppy markets automatically
- 📲 **Telegram Notifications**: Real-time signal delivery
- 💾 **MongoDB Persistence**: Signal history with 4-hour cooldown per symbol
- 🔄 **Concurrent Processing**: Worker pool using goroutines for efficiency
- ⏰ **Automated Polling**: Runs every 1 minute

## Prerequisites

- Go >= 1.21
- MongoDB (local or cloud)
- Telegram Bot Token
- Google Gemini API Key

## Installation

### 1. Install Go

```bash
# Ubuntu/Debian
sudo snap install go --classic

# Or download from https://go.dev/dl/
```

### 2. Clone/Setup Project

```bash
cd /home/najim/Desktop/projects/my-tool-go
```

### 3. Install Dependencies

```bash
go mod tidy
```

### 4. Configure Environment

Copy `.env.example` to `.env` and fill in your credentials:

```bash
cp .env.example .env
nano .env
```

Required variables:
- `MONGO_URI`: MongoDB connection string
- `BINANCE_API_KEY`: Binance API key (optional for public endpoints)
- `TELEGRAM_BOT_TOKEN`: Your Telegram bot token
- `TELEGRAM_CHAT_ID`: Your Telegram chat ID
- `GEMINI_API_KEY`: Google Gemini API key

## Usage

### Run Development Mode

```bash
go run cmd/server/main.go
```

### Build for Production

```bash
go build -o mrcrypto-go cmd/server/main.go
./mrcrypto-go
```

### Build for Linux (Cross-compile from any OS)

```bash
GOOS=linux GOARCH=amd64 go build -o mrcrypto-go-linux cmd/server/main.go
```

## Project Structure

```
mrcrypto-go/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── config/
│   │   └── config.go            # Environment configuration
│   ├── model/
│   │   └── signal.go            # Data structures
│   ├── service/
│   │   ├── binance.go           # Binance API client
│   │   ├── strategy.go          # Strategy evaluation
│   │   ├── ai.go                # Gemini AI validation
│   │   ├── telegram.go          # Telegram notifications
│   │   └── database.go          # MongoDB operations
│   ├── indicator/
│   │   ├── rsi.go               # RSI calculation
│   │   ├── adx.go               # ADX calculation
│   │   ├── vwap.go              # VWAP calculation
│   │   ├── macd.go              # MACD calculation
│   │   └── bollinger.go         # Bollinger Bands
│   ├── worker/
│   │   └── pool.go              # Worker pool manager
│   └── loader/
│       └── loader.go            # Main scheduler
├── .env.example                  # Environment template
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

## How It Works

### 1. Data Collection
Every minute, the system fetches all USDT trading pairs from Binance.

### 2. Parallel Processing
A worker pool (10 goroutines) processes symbols concurrently:
- Fetches klines for 4h, 1h, 15m, 5m timeframes
- Calculates technical indicators

### 3. Market Regime Detection
Determines if the market is:
- **TRENDING_UP**: Price > EMA50, ADX > 20
- **TRENDING_DOWN**: Price < EMA50, ADX > 20
- **RANGING**: ADX between 15-20
- **CHOPPY**: ADX < 20 (filtered out)

### 4. Signal Tier Evaluation

**PREMIUM** (🔥 Strict):
- ADX > 25
- Volume > 2x average
- Positive order flow delta
- Stricter RSI ranges

**STANDARD** (✅ Relaxed):
- ADX > 20
- Volume > 1x average
- Wider RSI ranges

### 5. AI Validation
Each signal is sent to Google Gemini AI for validation:
- Provides technical context
- Receives score (0-100) and reasoning
- Only signals with score ≥ 70 proceed

### 6. Cooldown Check
Ensures no duplicate signals for the same symbol within 4 hours.

### 7. Notification
Valid signals are:
- Saved to MongoDB
- Sent to Telegram with formatted message

## Signal Format Example

```
🟢 LONG Signal - 🔥 PREMIUM

Symbol: BTCUSDT
Type: LONG
Regime: TRENDING_UP

💰 Entry Price: 45230.50
🛑 Stop Loss: 44325.89 (-2.00%)
🎯 Take Profit: 47944.33 (+6.00%)

📊 Technical Context:
• RSI (1h/5m): 58.3 / 62.1
• ADX (1h): 27.5
• Volume: 2.3x avg
• MACD Histogram: 0.002345

🤖 AI Score: 85/100
💭 AI Reason: Strong uptrend with good volume confirmation...
```

## Dependencies

```go
require (
    github.com/joho/godotenv v1.5.1
    github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
    github.com/robfig/cron/v3 v3.0.1
    go.mongodb.org/mongo-driver v1.17.3
)
```

## Comparison with TypeScript Version

| Feature | TypeScript | Golang |
|---------|-----------|---------|
| Performance | Good | **Excellent** |
| Concurrency | Workers | **Goroutines** |
| Memory | ~100MB | **~20MB** |
| Deployment | node_modules | **Single binary** |
| Startup | ~2s | **~0.5s** |

## License

MIT

## Contact

- **Developer**: Najim
- **Email**: itsnajim.mail@gmail.com
- **GitHub**: [https://github.com/najim2004](https://github.com/najim2004)

---

Happy Trading! 🚀
