package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	NodeEnv          string
	Port             string
	MongoURI         string
	BinanceAPIKey    string
	BinanceSecretKey string
	BinanceBaseURL   string
	TelegramBotToken string
	TelegramChatID   string
	GeminiAPIKeys    []string // Supports multiple keys for rotation
}

var AppConfig *Config

// Load reads environment variables and initializes the global config
func Load() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	AppConfig = &Config{
		NodeEnv:          getEnv("NODE_ENV", "development"),
		Port:             getEnv("PORT", "8080"),
		MongoURI:         getEnv("MONGO_URI", "mongodb://localhost:27017/mrcrypto"),
		BinanceAPIKey:    getEnv("BINANCE_API_KEY", ""),
		BinanceSecretKey: getEnv("BINANCE_SECRET_KEY", ""),
		BinanceBaseURL:   getEnv("BINANCE_BASE_URL", "https://api.binance.com"),
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:   getEnv("TELEGRAM_CHAT_ID", ""),
		GeminiAPIKeys:    getEnvAsSlice("GEMINI_API_KEY", ""),
	}

	log.Println("✅ Configuration loaded successfully")
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsSlice(key, defaultValue string) []string {
	value := getEnv(key, defaultValue)
	if value == "" {
		return nil
	}
	// Split by comma
	return strings.Split(value, ",")
}
