package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"

	"bushlatinga_bot/handlers"
	"bushlatinga_bot/logging"
)

func main() {
	// Загружаем конфигурацию
	log.Println("🔧 Starting Bushlatinga Bot...")

	if err := godotenv.Load(); err != nil {
		log.Printf("⚠️ Warning: No .env file found: %v", err)
	} else {
		log.Println("✅ .env file loaded")
	}

	// Получаем токен бота
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("❌ TELEGRAM_BOT_TOKEN not found in .env")
	}

	log.Printf("🔑 Token preview: %s...", token[:min(20, len(token))])

	// Создаем бота
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("❌ Error creating bot: %v", err)
	}

	bot.Debug = os.Getenv("DEBUG") == "true"
	log.Printf("✅ Authorized as @%s (ID: %d)", bot.Self.UserName, bot.Self.ID)
	log.Printf("📝 Bot name: %s", bot.Self.FirstName)

	// Инициализация обработчика БД
	var dbHandler *handlers.BotDatabaseHandler

	// Получаем строку подключения к БД из .env
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Println("⚠️ DATABASE_URL not found in .env, using in-memory only mode")
	} else {
		log.Printf("📊 Database URL found, initializing Supabase connection...")

		// Получаем ID админа из .env
		adminID := int64(266468924) // Значение по умолчанию
		if adminEnv := os.Getenv("ADMIN_CHAT_ID"); adminEnv != "" {
			if id, err := strconv.ParseInt(adminEnv, 10, 64); err == nil {
				adminID = id
			}
		}
		log.Printf("👑 Admin ID: %d", adminID)

		// Создаем обработчик БД
		dbHandler, err = handlers.NewBotDatabaseHandler(adminID, dbURL)
		if err != nil {
			log.Printf("❌ Error initializing database handler: %v", err)
			log.Println("⚠️ Continuing in memory-only mode")
		} else {
			defer dbHandler.Close()
			log.Printf("✅ Database handler initialized with %d records in cache", dbHandler.GetMappingCount())
		}
	}

	// Настраиваем получение обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	log.Println("📡 Getting updates channel...")
	updates := bot.GetUpdatesChan(u)

	// Обработка сигналов для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("🚀 Bushlatinga Bot is running! Press Ctrl+C to stop.")
	log.Println("📱 Open Telegram and search for @bushlatinga_bot")

	// Основной цикл обработки сообщений
	for {
		select {
		case update := <-updates:
			log.Printf("📨 Update received: %+v", update.UpdateID)

			// Обработка сообщений
			if update.Message != nil {
				// Логируем детали сообщения
				chatType := "private"
				if update.Message.Chat.IsGroup() {
					chatType = "group"
				} else if update.Message.Chat.IsSuperGroup() {
					chatType = "supergroup"
				}
				logging.LogMessageDetails(update.Message, chatType)

				// Обработка команд (имеет приоритет)
				if update.Message.IsCommand() {
					handleCommand(bot, update.Message, dbHandler)
					continue
				}

				// Обработка обычных сообщений
				handleMessage(bot, update.Message, dbHandler)
			}

		case <-sigChan:
			log.Println("🛑 Shutting down Bushlatinga Bot...")
			bot.StopReceivingUpdates()
			return
		}
	}
}

func handleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, dbHandler *handlers.BotDatabaseHandler) {
	// Пытаемся найти совпадение в именах через БД (если она подключена)
	if dbHandler != nil {
		found, response := dbHandler.CheckForNames(msg.Text, msg.From.UserName)
		if found {
			log.Printf("✅ Name match found in DB for message: %s", msg.Text)

			// 🔥 ОБРАБОТКА СТИКЕРА ДЛЯ "ЕБ"
			if strings.HasPrefix(response, "STICKER:") {
				// 1. Отправляем стикер
				sticker := tgbotapi.NewSticker(msg.Chat.ID, tgbotapi.FileID(dbHandler.GetEBStickerID()))
				// Стикер тоже без цитирования
				// sticker.ReplyToMessageID = msg.MessageID

				if _, err := bot.Send(sticker); err != nil {
					log.Printf("❌ Error sending sticker: %v", err)
				} else {
					log.Printf("✅ Sticker sent to chat %d", msg.Chat.ID)
				}

				// 2. Отправляем текст (БЕЗ цитирования)
				textResponse := strings.TrimPrefix(response, "STICKER:")
				if textResponse != "" {
					reply := tgbotapi.NewMessage(msg.Chat.ID, textResponse)
					// БЕЗ цитирования
					// reply.ReplyToMessageID = msg.MessageID

					if _, err := bot.Send(reply); err != nil {
						log.Printf("❌ Error sending text after sticker: %v", err)
					}
				}
			} else {
				// Стандартная обработка текстового ответа (БЕЗ цитирования)
				reply := tgbotapi.NewMessage(msg.Chat.ID, response)
				// БЕЗ цитирования
				// reply.ReplyToMessageID = msg.MessageID

				if _, err := bot.Send(reply); err != nil {
					log.Printf("❌ Error sending name response: %v", err)
				}
			}
			return
		}
	}

	// Если не найдено совпадений в именах - НИЧЕГО НЕ ОТВЕЧАЕМ!
	log.Printf("📝 No name match found for message: %s", msg.Text)
}

func handleCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, dbHandler *handlers.BotDatabaseHandler) {
	log.Printf("⚡ Command received: /%s", msg.Command())

	switch msg.Command() {
	case "start":
		reply := tgbotapi.NewMessage(msg.Chat.ID,
			"🌿 *Привет! Я Bushlatinga Bot* — ваш помощник по документам и информации.\n\n"+
				"Я могу:\n"+
				"• Сохранять документы\n"+
				"• Искать информацию\n"+
				"• Помогать с вопросами\n"+
				"• Отвечать на упоминания участников\n\n"+
				"Используй /help для списка команд")
		reply.ParseMode = "Markdown"
		// Админ-команды можно оставить с цитированием для удобства
		// reply.ReplyToMessageID = msg.MessageID
		bot.Send(reply)

	case "help":
		helpText := "🆘 *Доступные команды:*\n\n" +
			"/start - Начать работу\n" +
			"/help - Помощь\n" +
			"/about - О боте\n"

		// Добавляем админ команду, если пользователь админ
		if dbHandler != nil && dbHandler.IsAdmin(msg.From.ID) {
			helpText += "/admin - Команды администратора\n"
		}

		helpText += "\n*Просто напиши мне вопрос или загрузи документ!*"

		reply := tgbotapi.NewMessage(msg.Chat.ID, helpText)
		reply.ParseMode = "Markdown"
		// reply.ReplyToMessageID = msg.MessageID
		bot.Send(reply)

	case "about":
		reply := tgbotapi.NewMessage(msg.Chat.ID,
			"🤖 *Bushlatinga Bot*\n"+
				"Версия: 2.0.0 (с поддержкой БД)\n"+
				"Разработчик: @vmkotov\n"+
				"Технологии: Go + Supabase PostgreSQL\n\n"+
				"Бот для работы с документами и реакцией на упоминания участников.")
		reply.ParseMode = "Markdown"
		// reply.ReplyToMessageID = msg.MessageID
		bot.Send(reply)

	case "admin":
		if dbHandler != nil {
			response := dbHandler.HandleAdminCommand(msg.From.ID, msg.Text)
			reply := tgbotapi.NewMessage(msg.Chat.ID, response)
			reply.ParseMode = "Markdown"
			// Для админ-команд можно оставить цитирование для ясности
			// reply.ReplyToMessageID = msg.MessageID
			bot.Send(reply)
		} else {
			reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ База данных не подключена. Режим работы: только в памяти.")
			// reply.ReplyToMessageID = msg.MessageID
			bot.Send(reply)
		}

	default:
		// Неизвестная команда
		reply := tgbotapi.NewMessage(msg.Chat.ID, "🤔 Неизвестная команда. Используйте /help для списка команд.")
		// reply.ReplyToMessageID = msg.MessageID
		bot.Send(reply)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
