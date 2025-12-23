package database

import (
	"fmt"
	"strings"
)

// HandleAdminCommand обрабатывает команды администратора для bushlatinga_bot
func (h *BotDatabaseHandler) HandleAdminCommand(userID int64, command string) string {
	// Проверяем права администратора
	if !h.IsAdmin(userID) {
		return "❌ У вас нет прав для выполнения этой команды"
	}

	// Убираем "/admin " из команды
	argsStr := strings.TrimSpace(strings.TrimPrefix(command, "/admin"))
	parts := strings.Fields(argsStr)

	if len(parts) == 0 {
		return h.showAdminHelp()
	}

	subCommand := strings.ToLower(parts[0])

	switch subCommand {
	case "add", "добавить":
		if len(parts) < 3 {
			return "❌ Использование: /admin add <ключ> <значение>\nПример: /admin add привет Привет!"
		}
		key := parts[1]
		value := strings.Join(parts[2:], " ")

		if err := h.AddMapping(key, value); err != nil {
			return fmt.Sprintf("❌ Ошибка: %v", err)
		}
		return fmt.Sprintf("✅ Добавлено:\n`%s` → `%s`", key, value)

	case "remove", "удалить", "del":
		if len(parts) < 2 {
			return "❌ Использование: /admin remove <ключ>"
		}
		key := parts[1]

		if err := h.RemoveMapping(key); err != nil {
			return fmt.Sprintf("❌ Ошибка: %v", err)
		}
		return fmt.Sprintf("✅ Удалено: `%s`", key)

	case "list", "список", "все":
		mapping := h.GetMapping()
		if len(mapping) == 0 {
			return "📭 База данных пуста. Добавьте фразы через /admin add"
		}

		var result strings.Builder
		result.WriteString(fmt.Sprintf("📋 Всего записей: %d\n\n", len(mapping)))

		count := 0
		for k, v := range mapping {
			count++
			safeKey := strings.ReplaceAll(k, "`", "'")
			safeValue := strings.ReplaceAll(v, "`", "'")
			result.WriteString(fmt.Sprintf("%d. `%s`\n   → %s\n\n", count, safeKey, safeValue))
			if count >= 30 {
				result.WriteString(fmt.Sprintf("\n... и еще %d записей\n", len(mapping)-count))
				break
			}
		}
		return result.String()

	case "search", "найти", "поиск":
		if len(parts) < 2 {
			return "❌ Использование: /admin search <текст>"
		}
		searchText := strings.Join(parts[1:], " ")
		results := h.SearchInValues(searchText)

		if len(results) == 0 {
			return fmt.Sprintf("🔍 Не найдено записей содержащих '%s'", searchText)
		}

		var result strings.Builder
		result.WriteString(fmt.Sprintf("🔍 Найдено %d записей:\n\n", len(results)))

		count := 0
		for k, v := range results {
			count++
			safeKey := strings.ReplaceAll(k, "`", "'")
			safeValue := strings.ReplaceAll(v, "`", "'")
			result.WriteString(fmt.Sprintf("%d. `%s`\n   → %s\n\n", count, safeKey, safeValue))
		}
		return result.String()

	case "count", "количество":
		count := h.GetMappingCount()
		return fmt.Sprintf("📊 Статистика:\n• Всего фраз: %d\n• Админ ID: %d", count, h.adminID)

	case "help", "помощь":
		return h.showAdminHelp()

	case "export", "экспорт":
		mapping := h.GetMapping()
		var result strings.Builder
		result.WriteString("�� Экспорт данных:\n\n")
		for k, v := range mapping {
			safeKey := strings.ReplaceAll(k, "`", "'")
			safeValue := strings.ReplaceAll(v, "`", "'")
			result.WriteString(fmt.Sprintf("`%s` → `%s`\n", safeKey, safeValue))
		}
		return result.String()

	case "info", "инфо":
		return "🤖 *bushlatinga_bot v2.0*\n\n" +
			"• База данных: Supabase PostgreSQL\n" +
			"• Схема: bushlatinga_bot (фразы), main (логи)\n" +
			"• Админ команды: /admin help\n" +
			"• Фразы сохраняются в облаке\n" +
			"• Работа напрямую с БД (без кэша)\n" +
			"• Отвечает только на одно совпадение\n" +
			"• ЕБ-детектор активен\n\n" +
			"Используйте /admin help для списка команд"

	case "test", "тест":
		// Тестовая команда для проверки ЕБ
		testMessage := "Тест ЕБ функции"
		ebFound := CheckForEB(testMessage)
		ebResult := "❌ Не найдено"
		if ebFound {
			ebResult = "✅ Найдено"
		}

		return fmt.Sprintf("🧪 Тест ЕБ-детектора:\n"+
			"Сообщение: '%s'\n"+
			"Результат: %s\n\n"+
			"Тест: напишите 'ЕБ' большими буквами", testMessage, ebResult)

	default:
		return "❌ Неизвестная команда. Используйте /admin help для списка команд"
	}
}

func (h *BotDatabaseHandler) showAdminHelp() string {
	return `🛠️ Команды администратора:

📝 Добавление/удаление:
/admin add <ключ> <значение> - Добавить новую запись
/admin remove <ключ> - Удалить запись

🔍 Поиск и просмотр:
/admin list - Показать все записи (первые 30)
/admin search <текст> - Найти текст в значениях
/admin count - Показать количество записей

📁 Экспорт и информация:
/admin export - Показать все записи для экспорта
/admin info - Информация о боте
/admin test - Протестировать ЕБ-детектор
/admin help - Эта справка

Примеры:
/admin add славик Славик абсолютно конченная поебота
/admin remove славик
/admin search спасибо
/admin test

📌 Примечания:
• Бот отвечает только на ОДНО совпадение в сообщении!
• "ЕБ" проверяется как отдельное слово большими буквами
• Все фразы хранятся только в БД (никаких фраз по умолчанию)!
• Работа напрямую с БД (без кэша в памяти)!`
}
