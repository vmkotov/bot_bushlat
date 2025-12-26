package bot

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vmkotov/telelog"
	"bushlatinga_bot/database"
)

// TelegramHandler обрабатывает вебхуки от Telegram
type TelegramHandler struct {
	bot               *tgbotapi.BotAPI
	dbHandler         *database.BotDatabaseHandler
	messageProcessor  *MessageProcessor
	commandProcessor  *CommandProcessor
	dbLogger          *DBLogger
	teleLogger        telelog.TeleLogger
	messageForwarder *MessageForwarder
}

// NewTelegramHandler создает новый обработчик Telegram
func NewTelegramHandler(
	bot *tgbotapi.BotAPI, 
	dbHandler *database.BotDatabaseHandler,
	teleLogger telelog.TeleLogger,
	messageForwarder *MessageForwarder,
) *TelegramHandler {
	return &TelegramHandler{
		bot:               bot,
		dbHandler:         dbHandler,
		messageProcessor:  NewMessageProcessor(dbHandler, teleLogger),
		commandProcessor:  NewCommandProcessor(dbHandler, teleLogger),
		dbLogger:          NewDBLogger(dbHandler, bot),
		teleLogger:        teleLogger,
		messageForwarder: messageForwarder,
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

	// Используем telelog для логирования
	if th.teleLogger != nil && th.teleLogger.IsEnabled() {
		th.teleLogger.LogMessage(msg, chatType)
	}

	// Логируем в базу данных
	if th.dbLogger != nil {
		th.dbLogger.LogMessage(msg)
	}

	// Пересылаем сообщение если forwarder инициализирован
	if th.messageForwarder != nil {
		th.messageForwarder.Forward(msg)
	}

	// Обрабатываем команду или сообщение
	if msg.IsCommand() {
		th.commandProcessor.ProcessCommand(th.bot, msg)
	} else {
		th.messageProcessor.ProcessMessage(th.bot, msg)
	}
}
