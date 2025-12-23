package bot

import (
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vmkotov/telelog"
	"bushlatinga_bot/database"
)

// MessageProcessor обрабатывает сообщения
type MessageProcessor struct {
	dbHandler  *database.BotDatabaseHandler
	teleLogger *telelog.TeleLogger
}

// NewMessageProcessor создает новый процессор сообщений
func NewMessageProcessor(dbHandler *database.BotDatabaseHandler, teleLogger *telelog.TeleLogger) *MessageProcessor {
	return &MessageProcessor{
		dbHandler:  dbHandler,
		teleLogger: teleLogger,
	}
}

// ProcessMessage обрабатывает входящее сообщение
func (mp *MessageProcessor) ProcessMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	// Пытаемся найти совпадение в именах через БД (если она подключена)
	if mp.dbHandler != nil {
		found, response := mp.dbHandler.CheckForNames(msg.Text, msg.From.UserName)
		if found {
			log.Printf("✅ Name match found in DB for message: %s", msg.Text)

			// 🔥 ОБРАБОТКА СТИКЕРА ДЛЯ "ЕБ"
			if strings.HasPrefix(response, "STICKER:") {
				// 1. Отправляем стикер
				sticker := tgbotapi.NewSticker(msg.Chat.ID, tgbotapi.FileID(mp.dbHandler.GetEBStickerID()))

				if _, err := bot.Send(sticker); err != nil {
					log.Printf("❌ Error sending sticker: %v", err)
				} else {
					log.Printf("✅ Sticker sent to chat %d", msg.Chat.ID)
				}

				// 2. Отправляем текст
				textResponse := strings.TrimPrefix(response, "STICKER:")
				if textResponse != "" {
					reply := tgbotapi.NewMessage(msg.Chat.ID, textResponse)

					if _, err := bot.Send(reply); err != nil {
						log.Printf("❌ Error sending text after sticker: %v", err)
					}
				}
			} else {
				// Стандартная обработка текстового ответа
				reply := tgbotapi.NewMessage(msg.Chat.ID, response)

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
