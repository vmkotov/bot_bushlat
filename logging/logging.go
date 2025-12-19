package logging

import (
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// LogMessageDetails логирует всю информацию о входящем сообщении
func LogMessageDetails(message *tgbotapi.Message, chatType string) {
	log.Printf("📥 INCOMING MESSAGE:")
	log.Printf("   👤 User: %s %s (ID: %d, Username: @%s, Lang: %s)",
		message.From.FirstName,
		message.From.LastName,
		message.From.ID,
		message.From.UserName,
		message.From.LanguageCode)

	log.Printf("   💬 Chat: %s (ID: %d, Type: %s)",
		getChatTitle(message.Chat),
		message.Chat.ID,
		chatType)

	log.Printf("   📝 Text: %s", message.Text)
	log.Printf("   🆔 Message ID: %d", message.MessageID)
	log.Printf("   📅 Date: %v", time.Unix(int64(message.Date), 0).Format("2006-01-02 15:04:05"))

	// Дополнительные данные
	if message.ReplyToMessage != nil {
		log.Printf("   ↪️  Reply to: %d", message.ReplyToMessage.MessageID)
	}
	if message.ForwardFrom != nil {
		log.Printf("   ↩️  Forwarded from user ID: %d", message.ForwardFrom.ID)
	}
	if message.ForwardFromChat != nil {
		log.Printf("   ↩️  Forwarded from chat: %s (ID: %d)",
			getChatTitle(message.ForwardFromChat),
			message.ForwardFromChat.ID)
	}
	if len(message.Photo) > 0 {
		log.Printf("   📸 Photo: %d sizes, file_id: %s",
			len(message.Photo),
			message.Photo[len(message.Photo)-1].FileID)
	}
	if message.Sticker != nil {
		log.Printf("   🎭 Sticker: %s, emoji: %s",
			message.Sticker.FileUniqueID,
			message.Sticker.Emoji)
	}
	if message.Document != nil {
		log.Printf("   📎 Document: %s, mime: %s",
			message.Document.FileName,
			message.Document.MimeType)
	}
	if message.Location != nil {
		log.Printf("   📍 Location: lat=%.6f, lon=%.6f",
			message.Location.Latitude,
			message.Location.Longitude)
	}
	if message.Voice != nil {
		log.Printf("   🎤 Voice: %d sec", message.Voice.Duration)
	}
}

// getChatTitle возвращает название чата
func getChatTitle(chat *tgbotapi.Chat) string {
	if chat.Title != "" {
		return chat.Title
	}
	if chat.FirstName != "" {
		title := chat.FirstName
		if chat.LastName != "" {
			title += " " + chat.LastName
		}
		return title
	}
	return "Unknown"
}
