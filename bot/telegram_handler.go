package bot

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"bushlatinga_bot/database"
)

// TelegramHandler обрабатывает вебхуки от Telegram
type TelegramHandler struct {
	bot               *tgbotapi.BotAPI
	dbHandler         *database.BotDatabaseHandler
	messageProcessor  *MessageProcessor
	commandProcessor  *CommandProcessor
	dbLogger          *DBLogger
}

// NewTelegramHandler создает новый обработчик Telegram
func NewTelegramHandler(bot *tgbotapi.BotAPI, dbHandler *database.BotDatabaseHandler) *TelegramHandler {
	return &TelegramHandler{
		bot:               bot,
		dbHandler:         dbHandler,
		messageProcessor:  NewMessageProcessor(dbHandler),
		commandProcessor:  NewCommandProcessor(dbHandler),
		dbLogger:          NewDBLogger(dbHandler, bot),
	}
}

// HandleWebhook обрабатывает вебхук от Telegram
func (th *TelegramHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Error reading request body: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var update tgbotapi.Update
	if err := json.Unmarshal(body, &update); err != nil {
		log.Printf("❌ Error unmarshaling update: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Обработка сообщения
	if update.Message != nil {
		th.processMessage(&update)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// processMessage обрабатывает сообщение
func (th *TelegramHandler) processMessage(update *tgbotapi.Update) {
	msg := update.Message
	
	chatType := "private"
	if msg.Chat.IsGroup() {
		chatType = "group"
	} else if msg.Chat.IsSuperGroup() {
		chatType = "supergroup"
	}

	log.Printf("📨 Сообщение от @%s в %s: %s",
		msg.From.UserName,
		chatType,
		msg.Text)

	// Логируем в базу данных
	if th.dbLogger != nil {
		th.dbLogger.LogMessage(msg)
	}

	// Обрабатываем команду или сообщение
	if msg.IsCommand() {
		th.commandProcessor.ProcessCommand(th.bot, msg)
	} else {
		th.messageProcessor.ProcessMessage(th.bot, msg)
	}
}
