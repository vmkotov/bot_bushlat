package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/vmkotov/telelog"

	"bushlatinga_bot/bot"
	"bushlatinga_bot/database"
)

func main() {
	// Загружаем конфигурацию
	log.Println("🔧 Запускаю Bushlatinga Bot v3.0 (Модульная архитектура)...")

	if err := godotenv.Load(); err != nil {
		log.Printf("⚠️ Внимание: Файл .env не найден: %v", err)
	}

	// Получаем токен бота
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("❌ TELEGRAM_BOT_TOKEN не найден в .env")
	}

	// Создаем бота
	botAPI, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("❌ Ошибка создания бота: %v", err)
	}

	botAPI.Debug = os.Getenv("DEBUG") == "true"
	log.Printf("✅ Авторизован как @%s (ID: %d)", botAPI.Self.UserName, botAPI.Self.ID)

	// 🔍 Отладочная информация
	log.Printf("🔍 Проверяю переменные окружения:")
	log.Printf("   LOG_CHAT_ID='%s'", os.Getenv("LOG_CHAT_ID"))
	log.Printf("   TELEGRAM_BOT_TOKEN установлен: %v", os.Getenv("TELEGRAM_BOT_TOKEN") != "")
	log.Printf("   DATABASE_URL установлен: %v", os.Getenv("DATABASE_URL") != "")

	// ИНИЦИАЛИЗАЦИЯ TELELOGGER
	var teleLogger telelog.TeleLogger
	var logChatID int64 // Объявляем переменную здесь

	// Получаем ID чата для логов из .env
	logChatIDStr := os.Getenv("LOG_CHAT_ID")

	// ⚠️ ЗАМЕНЕН ХАРДКОД: если не установлено, используем новый ID супергруппы
	if logChatIDStr == "" {
		logChatIDStr = "-1003585352063" // <-- НОВЫЙ ID СУПЕРГРУППЫ
		log.Printf("⚠️ LOG_CHAT_ID не установлен, использую ID супергруппы: %s", logChatIDStr)
	}

	if logChatIDStr != "" {
		var err error
		logChatID, err = strconv.ParseInt(logChatIDStr, 10, 64)
		if err == nil && logChatID != 0 {
			// ✅ ПРАВИЛЬНЫЙ КОНСТРУКТОР для telelog v0.3.0
			teleLogger = telelog.New(telelog.Options{
				Bot:         botAPI,
				LogChatID:   logChatID,
				BotID:       botAPI.Self.ID,
				BotUsername: botAPI.Self.UserName,
			})
			log.Printf("✅ TeleLogger инициализирован для чата ID: %d", logChatID)

			// Проверяем доступность чата
			if teleLogger.IsEnabled() {
				// Отправляем тестовое сообщение для проверки
				testMsg := tgbotapi.NewMessage(logChatID, "🔄 Бот перезапущен. TeleLogger работает!")
				if _, err := botAPI.Send(testMsg); err != nil {
					log.Printf("⚠️ Не удалось отправить тестовое сообщение в чат %d: %v", logChatID, err)
					log.Printf("⚠️ Проверьте: 1) Бот добавлен в группу 2) Бот имеет права на отправку сообщений 3) ID чата правильный")
				} else {
					log.Printf("✅ Тестовое сообщение успешно отправлено в чат %d", logChatID)
				}
			}
		} else {
			log.Printf("⚠️ Неверный LOG_CHAT_ID '%s', использую консольный логгер", logChatIDStr)
			teleLogger = telelog.SimpleNew()
		}
	} else {
		teleLogger = telelog.SimpleNew()
		log.Println("ℹ️ LOG_CHAT_ID не установлен, использую консольный логгер")
	}

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
			log.Printf("❌ Ошибка инициализации обработчика БД: %v", err)
		} else {
			defer dbHandler.Close()
			log.Printf("✅ Обработчик БД инициализирован")
		}
	}

	// Создаем обработчик Telegram с логгером

	// ИНИЦИАЛИЗАЦИЯ MESSAGE FORWARDER
	var messageForwarder *bot.MessageForwarder
	if logChatID != 0 {
		messageForwarder = bot.NewMessageForwarder(botAPI, logChatID)
		log.Printf("✅ MessageForwarder инициализирован для чата ID: %d", logChatID)
	}

	telegramHandler := bot.NewTelegramHandler(botAPI, dbHandler, teleLogger, messageForwarder)

	// Настраиваем HTTP роутер
	http.HandleFunc("/", telegramHandler.HandleWebhook)

	// Получаем порт из переменной окружения
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Отправляем уведомление о запуске
	if teleLogger != nil && teleLogger.IsEnabled() {
		deployInfo := map[string]string{
			"version":     "3.0",
			"environment": getEnvOrDefault("ENVIRONMENT", "production"),
			"branch":      getEnvOrDefault("BRANCH", "main"),
			"commit_hash": getEnvOrDefault("COMMIT_HASH", "unknown"),
			"deployer":    "Bushlatinga Bot",
			"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		}
		teleLogger.SendDeployNotification(deployInfo)
		log.Println("🚀 Уведомление о деплое отправлено")
	} else {
		log.Println("⚠️ TeleLogger не включен, уведомление о деплое не отправлено")
	}

	log.Printf("🌐 Запускаю HTTP сервер на порту %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("❌ Не удалось запустить сервер: %v", err)
	}
}

// getEnvOrDefault возвращает значение переменной окружения или значение по умолчанию
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
