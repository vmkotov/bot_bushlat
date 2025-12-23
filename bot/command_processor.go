package bot

import (
	"log"

	"bushlatinga_bot/database"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vmkotov/telelog"
)

// CommandProcessor обрабатывает команды
type CommandProcessor struct {
	dbHandler  *database.BotDatabaseHandler
	teleLogger *telelog.TeleLogger
}

// NewCommandProcessor создает новый процессор команд
func NewCommandProcessor(dbHandler *database.BotDatabaseHandler, teleLogger *telelog.TeleLogger) *CommandProcessor {
	return &CommandProcessor{
		dbHandler:  dbHandler,
		teleLogger: teleLogger,
	}
}

// ProcessCommand обрабатывает команду
func (cp *CommandProcessor) ProcessCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	log.Printf("⚡ Command received: /%s", msg.Command())

	// Логируем команду через telelog
	if cp.teleLogger != nil && cp.teleLogger.IsEnabled() {
		cp.teleLogger.LogCommand(msg, msg.Command())
	}

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
		bot.Send(reply)

	case "help":
		helpText := cp.getHelpText(msg.From.ID)
		reply := tgbotapi.NewMessage(msg.Chat.ID, helpText)
		reply.ParseMode = "Markdown"
		bot.Send(reply)

	case "about":
		reply := tgbotapi.NewMessage(msg.Chat.ID,
			"🤖 *Bushlatinga Bot*\n"+
				"Версия: 3.0.0 (модульная архитектура)\n"+
				"Разработчик: @vmkotov\n"+
				"Технологии: Go + Supabase PostgreSQL\n\n"+
				"Бот для работы с документами и реакцией на упоминания участников.")
		reply.ParseMode = "Markdown"
		bot.Send(reply)

	case "admin":
		cp.processAdminCommand(bot, msg)

	default:
		reply := tgbotapi.NewMessage(msg.Chat.ID, "🤔 Неизвестная команда. Используйте /help для списка команд.")
		bot.Send(reply)
	}
}

// getHelpText возвращает текст помощи
func (cp *CommandProcessor) getHelpText(userID int64) string {
	helpText := "🆘 *Доступные команды:*\n\n" +
		"/start - Начать работу\n" +
		"/help - Помощь\n" +
		"/about - О боте\n"

	// Добавляем админ команду, если пользователь админ
	if cp.dbHandler != nil && cp.dbHandler.IsAdmin(userID) {
		helpText += "/admin - Команды администратора\n"
	}

	helpText += "\n*Просто напиши мне вопрос или загрузи документ!*"
	return helpText
}

// processAdminCommand обрабатывает админ команды
func (cp *CommandProcessor) processAdminCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	if cp.dbHandler != nil {
		response := cp.dbHandler.HandleAdminCommand(msg.From.ID, msg.Text)
		reply := tgbotapi.NewMessage(msg.Chat.ID, response)
		reply.ParseMode = "Markdown"
		bot.Send(reply)
	} else {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ База данных не подключена. Режим работы: только в памяти.")
		bot.Send(reply)
	}
}
