package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"

	"bushlatinga_bot/bot"
	"bushlatinga_bot/database"
)

func main() {
	// Загружаем конфигурацию
	log.Println("🔧 Starting Bushlatinga Bot v3.0 (Modular Architecture)...")

	if err := godotenv.Load(); err != nil {
		log.Printf("⚠️ Warning: No .env file found: %v", err)
	}

	// Получаем токен бота
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("❌ TELEGRAM_BOT_TOKEN not found in .env")
	}

	// Создаем бота
	botAPI, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("❌ Error creating bot: %v", err)
	}

	botAPI.Debug = os.Getenv("DEBUG") == "true"
	log.Printf("✅ Authorized as @%s (ID: %d)", botAPI.Self.UserName, botAPI.Self.ID)

	// Инициализация обработчика БД
	var dbHandler *database.BotDatabaseHandler
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		adminID := int64(266468924)
		if adminEnv := os.Getenv("ADMIN_CHAT_ID"); adminEnv != "" {
			if id, err := strconv.ParseInt(adminEnv, 10, 64); err == nil {
				adminID = id
			}
		}

		var err error
		dbHandler, err = database.NewBotDatabaseHandler(adminID, dbURL)
		if err != nil {
			log.Printf("❌ Error initializing database handler: %v", err)
		} else {
			defer dbHandler.Close()
			log.Printf("✅ Database handler initialized")
		}
	}

	// Создаем обработчик Telegram
	telegramHandler := bot.NewTelegramHandler(botAPI, dbHandler)

	// Настраиваем HTTP роутер
	http.HandleFunc("/", telegramHandler.HandleWebhook)

	// Получаем порт из переменной окружения
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🌐 Starting HTTP server on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}
