package bot

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"bushlatinga_bot/database"
)

// DBLogger логирует сообщения в базу данных и Telegram чат
type DBLogger struct {
	dbHandler *database.BotDatabaseHandler
	bot       *tgbotapi.BotAPI
	logChatID int64
}

// NewDBLogger создает новый логгер БД
func NewDBLogger(dbHandler *database.BotDatabaseHandler, bot *tgbotapi.BotAPI) *DBLogger {
	// Жестко задаем ID чата для логов
	logChatID := int64(-1003585352063)
	
	return &DBLogger{
		dbHandler: dbHandler,
		bot:       bot,
		logChatID: logChatID,
	}
}

// LogMessage логирует сообщение в базу данных и Telegram
func (dl *DBLogger) LogMessage(msg *tgbotapi.Message) {
	if dl.dbHandler == nil || dl.dbHandler.DB() == nil {
		return
	}

	// Проверяем, что это не сообщение от самого бота
	if msg.From != nil && msg.From.ID == dl.bot.Self.ID {
		return
	}

	// Логируем в базу данных
	dl.logToDatabase(msg)
	
	// Логируем в Telegram чат
	dl.logToTelegram(msg)
	
	// Обновляем статистику
	dl.updateBotStats()
}

// logToDatabase логирует сообщение в базу данных
func (dl *DBLogger) logToDatabase(msg *tgbotapi.Message) {
	// Подготавливаем данные
	chatTitle := ""
	if msg.Chat.Title != "" {
		chatTitle = msg.Chat.Title
	}

	chatType := "private"
	if msg.Chat.IsGroup() {
		chatType = "group"
	} else if msg.Chat.IsSuperGroup() {
		chatType = "supergroup"
	} else if msg.Chat.IsChannel() {
		chatType = "channel"
	}

	userName := ""
	if msg.From.FirstName != "" {
		userName = msg.From.FirstName
		if msg.From.LastName != "" {
			userName += " " + msg.From.LastName
		}
	}

	userUsername := ""
	if msg.From.UserName != "" {
		userUsername = msg.From.UserName
	}

	messageText := msg.Text
	messageType := "text"

	hasSticker := false
	stickerEmoji := ""
	if msg.Sticker != nil {
		hasSticker = true
		stickerEmoji = msg.Sticker.Emoji
		messageType = "sticker"
		if messageText == "" {
			messageText = stickerEmoji
		}
	}

	hasPhoto := len(msg.Photo) > 0
	if hasPhoto && messageType == "text" {
		messageType = "photo"
	}

	hasDocument := msg.Document != nil
	if hasDocument && messageType == "text" {
		messageType = "document"
		if messageText == "" {
			messageText = msg.Document.FileName
		}
	}

	replyToMessageID := int64(0)
	replyToUserID := int64(0)
	if msg.ReplyToMessage != nil {
		replyToMessageID = int64(msg.ReplyToMessage.MessageID)
		if msg.ReplyToMessage.From != nil {
			replyToUserID = msg.ReplyToMessage.From.ID
		}
	}

	// Вставляем запись в базу данных
	query := `
		INSERT INTO main.messages_log (
			bot_id, bot_username, chat_id, chat_title, chat_type,
			user_id, user_name, user_username, message_id, message_text,
			message_type, reply_to_message_id, reply_to_user_id,
			has_sticker, sticker_emoji, has_photo, has_document
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (bot_id, chat_id, message_id) DO NOTHING
	`

	_, err := dl.dbHandler.DB().Exec(query,
		dl.bot.Self.ID, dl.bot.Self.UserName, msg.Chat.ID, chatTitle, chatType,
		msg.From.ID, userName, userUsername, msg.MessageID, messageText,
		messageType, replyToMessageID, replyToUserID,
		hasSticker, stickerEmoji, hasPhoto, hasDocument,
	)

	if err != nil {
		log.Printf("❌ Ошибка сохранения лога в БД: %v", err)
	} else {
		log.Printf("✅ Сообщение сохранено в БД: chat_id=%d, user_id=%d", msg.Chat.ID, msg.From.ID)
	}
}

// logToTelegram отправляет лог в Telegram чат
func (dl *DBLogger) logToTelegram(msg *tgbotapi.Message) {
	if dl.logChatID == 0 || dl.bot == nil {
		return
	}

	// Форматируем сообщение БЕЗ Markdown для избежания ошибок парсинга
	chatInfo := dl.formatChatInfo(msg)
	userInfo := dl.formatUserInfo(msg)
	messageInfo := dl.formatMessageInfo(msg)
	botInfo := dl.formatBotInfo()

	text := fmt.Sprintf(
		"🤖 Лог сообщения %s\n\n"+
			"%s\n"+
			"%s\n"+
			"%s\n"+
			"%s",
		msg.Time().Format("15:04:05"),
		chatInfo,
		userInfo,
		messageInfo,
		botInfo,
	)

	// Ограничиваем длину сообщения
	if len(text) > 4000 {
		text = text[:4000] + "\n... (сообщение обрезано)"
	}

	logMsg := tgbotapi.NewMessage(dl.logChatID, text)
	// УБИРАЕМ ParseMode чтобы избежать ошибок Markdown
	// logMsg.ParseMode = "Markdown"

	if _, err := dl.bot.Send(logMsg); err != nil {
		log.Printf("❌ Не удалось отправить логи в чат %d: %v", dl.logChatID, err)
	} else {
		log.Printf("✅ Логи отправлены в Telegram чат %d", dl.logChatID)
	}
}

// formatChatInfo форматирует информацию о чате (без Markdown)
func (dl *DBLogger) formatChatInfo(msg *tgbotapi.Message) string {
	chatType := "личный"
	if msg.Chat.IsGroup() {
		chatType = "группа"
	} else if msg.Chat.IsSuperGroup() {
		chatType = "супергруппа"
	} else if msg.Chat.IsChannel() {
		chatType = "канал"
	}

	chatTitle := "Без названия"
	if msg.Chat.Title != "" {
		chatTitle = msg.Chat.Title
	}

	return fmt.Sprintf(
		"💬 Чат: %s\n"+
			"📌 Тип: %s\n"+
			"🆔 ID: %d",
		chatTitle,
		chatType,
		msg.Chat.ID,
	)
}

// formatUserInfo форматирует информацию о пользователе (без Markdown)
func (dl *DBLogger) formatUserInfo(msg *tgbotapi.Message) string {
	if msg.From == nil {
		return "👤 Пользователь: Неизвестен"
	}

	userName := msg.From.UserName
	if userName == "" {
		userName = "без username"
	}

	fullName := fmt.Sprintf("%s %s", 
		msg.From.FirstName, 
		msg.From.LastName)
	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		fullName = "Без имени"
	}

	return fmt.Sprintf(
		"👤 Пользователь: %s\n"+
			"📛 Имя: %s\n"+
			"�� @%s\n"+
			"🆔 ID: %d",
		fullName,
		msg.From.FirstName,
		userName,
		msg.From.ID,
	)
}

// formatMessageInfo форматирует информацию о сообщении (без Markdown)
func (dl *DBLogger) formatMessageInfo(msg *tgbotapi.Message) string {
	messageText := msg.Text
	if messageText == "" {
		messageText = "⚠️ Без текста"
		
		// Проверяем другие типы контента
		if msg.Sticker != nil {
			messageText = fmt.Sprintf("🎭 Стикер: %s", msg.Sticker.Emoji)
		} else if msg.Photo != nil && len(msg.Photo) > 0 {
			messageText = "🖼️ Фото"
		} else if msg.Video != nil {
			messageText = "🎬 Видео"
		} else if msg.Document != nil {
			messageText = fmt.Sprintf("�� Документ: %s", msg.Document.FileName)
		} else if msg.Audio != nil {
			messageText = "🎵 Аудио"
		} else if msg.Voice != nil {
			messageText = "🎤 Голосовое сообщение"
		} else if msg.Location != nil {
			messageText = "📍 Локация"
		} else if msg.Contact != nil {
			messageText = "👤 Контакт"
		}
	}

	info := fmt.Sprintf("📝 Сообщение:\n%s", messageText)

	// Добавляем информацию о reply, если есть
	if msg.ReplyToMessage != nil {
		replyText := msg.ReplyToMessage.Text
		if replyText == "" {
			replyText = "⬆️ (сообщение без текста)"
		}
		if len(replyText) > 100 {
			replyText = replyText[:100] + "..."
		}
		
		info += fmt.Sprintf("\n\n↩️ Ответ на:\n%s", replyText)
	}

	return info
}

// formatBotInfo форматирует информацию о боте (без Markdown)
func (dl *DBLogger) formatBotInfo() string {
	return fmt.Sprintf(
		"\n🤖 Информация о боте:\n"+
			"Бот: @%s\n"+
			"Bot ID: %d",
		dl.bot.Self.UserName,
		dl.bot.Self.ID,
	)
}

// updateBotStats обновляет статистику бота
func (dl *DBLogger) updateBotStats() {
	if dl.dbHandler == nil || dl.dbHandler.DB() == nil {
		return
	}

	query := `
		INSERT INTO main.bot_stats (bot_id, bot_username, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (bot_id) DO UPDATE SET
			updated_at = NOW(),
			bot_username = EXCLUDED.bot_username
	`

	_, err := dl.dbHandler.DB().Exec(query, dl.bot.Self.ID, dl.bot.Self.UserName)
	if err != nil {
		log.Printf("❌ Ошибка обновления статистики: %v", err)
	}
}
