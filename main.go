package main

import (
	"bushlatinga_bot/handlers"
	"bushlatinga_bot/logging"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

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

	// 🔧 ID целевого чата для пересылки ВСЕХ сообщений
	targetChatID := int64(-5094399861)
	log.Printf("🔄 Target chat ID: %d", targetChatID)

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

		// ЛОГИРОВАНИЕ - ВЫПОЛНЯЕТСЯ ДЛЯ ВСЕХ СООБЩЕНИЙ
		chatType := "личные"
		if message.Chat.IsGroup() || message.Chat.IsSuperGroup() {
			chatType = "группа"
		} else if message.Chat.IsChannel() {
			chatType = "канал"
		}
		logging.LogMessageDetails(message, chatType)

		// 🔧 ПЕРЕСЫЛКА ВСЕХ СООБЩЕНИЙ В ЦЕЛЕВОЙ ЧАТ
		forwardMsg := tgbotapi.NewForward(targetChatID, message.Chat.ID, message.MessageID)

		// Добавляем задержку, чтобы не превышать лимиты API
		time.Sleep(100 * time.Millisecond)

		sentMsg, err := bot.Send(forwardMsg)
		if err != nil {
			log.Printf("❌ Ошибка пересылки сообщения %d в чат %d: %v",
				message.MessageID, targetChatID, err)
			log.Printf("   Отправитель: %d, Текст: %s",
				message.Chat.ID, message.Text)

			// Проверяем конкретные ошибки
			errStr := err.Error()
			switch {
			case errStr == "Forbidden: bot was kicked from the group chat":
				log.Printf("   ⚠️ Бота кикнули из чата %d", targetChatID)
			case errStr == "Forbidden: bot is not a member of the group chat":
				log.Printf("   ⚠️ Бот не добавлен в чат %d", targetChatID)
			case errStr == "Bad Request: chat not found":
				log.Printf("   ⚠️ Чат %d не найден", targetChatID)
			case errStr == "Forbidden: bot can't send messages to bots":
				log.Printf("   ⚠️ Бот не может отправлять сообщения другим ботам")
			case errStr == "Forbidden: user is deactivated":
				log.Printf("   ⚠️ Пользователь деактивирован")
			}

			// Альтернатива: отправка копии сообщения вместо пересылки
			if message.Text != "" {
				msg := tgbotapi.NewMessage(targetChatID,
					fmt.Sprintf("📨 От %s (@%s): %s",
						message.From.FirstName,
						message.From.UserName,
						message.Text))

				if _, err2 := bot.Send(msg); err2 != nil {
					log.Printf("❌ Ошибка отправки копии: %v", err2)
				} else {
					log.Printf("📝 Копия отправлена в чат %d", targetChatID)
				}
			} else if message.Sticker != nil {
				// Для стикеров
				sticker := tgbotapi.NewSticker(targetChatID, tgbotapi.FileID(message.Sticker.FileID))
				if _, err2 := bot.Send(sticker); err2 != nil {
					log.Printf("❌ Ошибка отправки стикера: %v", err2)
				}
			}
		} else {
			log.Printf("✅ Сообщение %d переслано в чат %d (ID пересланного: %d)",
				message.MessageID, targetChatID, sentMsg.MessageID)
		}

		messageText := message.Text
		// Если сообщение не текстовое (например, только стикер)
		if messageText == "" {
			continue
		}

		userName := message.From.FirstName
		chatID := message.Chat.ID

		// 🔥 ПРОВЕРКА НА "ЕБ"
		if handlers.CheckForEB(messageText) {
			log.Printf("🎉 Упоминание Евгена Борисыча от %s", userName)

			// Отправляем стикер (БЕЗ цитирования)
			stickerID := handlers.GetStickerID()
			sticker := tgbotapi.NewSticker(chatID, tgbotapi.FileID(stickerID))

			if _, err := bot.Send(sticker); err != nil {
				log.Printf("❌ Ошибка отправки стикера: %v", err)
			}

			// Отправляем текстовый ответ (БЕЗ цитирования)
			response := handlers.GetRandomEBResponse(userName)
			msg := tgbotapi.NewMessage(chatID, response)

			if _, err := bot.Send(msg); err != nil {
				log.Printf("❌ Ошибка отправки сообщения: %v", err)
			}

			continue
		}

		// 📝 ПРОВЕРКА НА ИМЕНА
		hasNames, nameResponse := handlers.CheckForNames(messageText, userName)
		if hasNames {
			msg := tgbotapi.NewMessage(chatID, nameResponse)

			if _, err := bot.Send(msg); err != nil {
				log.Printf("❌ Ошибка отправки сообщения: %v", err)
			}
			continue
		}
	}
}
