package main

import (
	"log"
	"math/rand"
	"time"

	"bushlatinga_bot/handlers"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ Файл .env не найден, используем системные переменные")
	}

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Panic("❌ TELEGRAM_BOT_TOKEN не установлен")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false
	log.Printf("✅ Бот запущен как: %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	u.AllowedUpdates = []string{
		"message",
		"edited_message",
		"channel_post",
	}

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		var message *tgbotapi.Message

		if update.Message != nil {
			message = update.Message
		} else if update.ChannelPost != nil {
			message = update.ChannelPost
		} else if update.EditedMessage != nil {
			message = update.EditedMessage
		}

		if message == nil {
			continue
		}

		// Пропускаем сообщения от самого бота
		if message.From != nil && message.From.ID == bot.Self.ID {
			continue
		}

		messageText := message.Text
		// Если сообщение не текстовое (например, только стикер)
		if messageText == "" {
			continue
		}

		userName := message.From.FirstName
		chatID := message.Chat.ID

		// Логирование для отладки
		chatType := "личные"
		if message.Chat.IsGroup() || message.Chat.IsSuperGroup() {
			chatType = "группа"
		} else if message.Chat.IsChannel() {
			chatType = "канал"
		}

		log.Printf("[%s] %s: %s", chatType, userName, messageText)

		// 🔥 ПРОВЕРКА НА "ЕБ"
		if handlers.CheckForEB(messageText) {
			log.Printf("🎉 Упоминание Евгена Борисыча от %s", userName)

			// Отправляем стикер (БЕЗ цитирования)
			stickerID := handlers.GetStickerID()
			sticker := tgbotapi.NewSticker(chatID, tgbotapi.FileID(stickerID))
			// sticker.ReplyToMessageID = message.MessageID // УБРАЛИ ЭТУ СТРОКУ

			if _, err := bot.Send(sticker); err != nil {
				log.Printf("❌ Ошибка отправки стикера: %v", err)
			}

			// Отправляем текстовый ответ (БЕЗ цитирования)
			response := handlers.GetRandomEBResponse(userName)
			msg := tgbotapi.NewMessage(chatID, response)
			// msg.ReplyToMessageID = message.MessageID // УБРАЛИ ЭТУ СТРОКУ

			if _, err := bot.Send(msg); err != nil {
				log.Printf("❌ Ошибка отправки сообщения: %v", err)
			}

			continue
		}

		// 📝 ПРОВЕРКА НА ИМЕНА
		hasNames, nameResponse := handlers.CheckForNames(messageText, userName)
		if hasNames {
			msg := tgbotapi.NewMessage(chatID, nameResponse)
			// msg.ReplyToMessageID = message.MessageID // УБРАЛИ ЭТУ СТРОКУ

			if _, err := bot.Send(msg); err != nil {
				log.Printf("❌ Ошибка отправки сообщения: %v", err)
			}
			continue
		}
	}
}
