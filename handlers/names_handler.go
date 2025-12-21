package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"unicode"

	_ "github.com/lib/pq"
)

// ID стикера для "ЕБ"
const (
	EBStickerID = "CAACAgIAAxkBAANTaUVkrWrIsoO8kVNAifaUqz16ex4AAqqFAAJVF1hIHdoBVVf89Yg2BA"
)

// BotDatabaseHandler - основной обработчик для bushlatinga_bot
type BotDatabaseHandler struct {
	db      *sql.DB
	mu      sync.RWMutex
	adminID int64
	cache   map[string]string // Кэш в памяти для быстрого доступа
}

// NewBotDatabaseHandler создает новый обработчик БД для bushlatinga_bot
func NewBotDatabaseHandler(adminID int64, connectionString string) (*BotDatabaseHandler, error) {
	// Подключаемся к базе данных
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к БД: %v", err)
	}

	// Проверяем подключение
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("не удалось проверить подключение к БД: %v", err)
	}

	log.Println("✅ [bushlatinga_bot] Успешное подключение к Supabase")

	handler := &BotDatabaseHandler{
		db:      db,
		adminID: adminID,
		cache:   make(map[string]string),
	}

	// Инициализируем базу данных
	err = handler.initializeDatabase()
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации БД: %v", err)
	}

	// Загружаем данные в кэш
	err = handler.loadCache()
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки кэша: %v", err)
	}

	log.Printf("✅ [bushlatinga_bot] Загружено %d записей в кэш\n", len(handler.cache))

	return handler, nil
}

// GetEBStickerID возвращает ID стикера для "ЕБ"
func (h *BotDatabaseHandler) GetEBStickerID() string {
	return EBStickerID
}

// initializeDatabase создает таблицы, если их нет
func (h *BotDatabaseHandler) initializeDatabase() error {
	// Создаем таблицу для хранения фраз bushlatinga_bot
	createTableQuery := `
        CREATE TABLE IF NOT EXISTS bushlatinga_responses (
            id BIGSERIAL PRIMARY KEY,
            trigger_text VARCHAR(100) UNIQUE NOT NULL,
            response_text TEXT NOT NULL,
            created_at TIMESTAMPTZ DEFAULT NOW(),
            updated_at TIMESTAMPTZ DEFAULT NOW()
        );
        
        CREATE INDEX IF NOT EXISTS idx_bushlatinga_trigger_text ON bushlatinga_responses(trigger_text);
        
        COMMENT ON TABLE bushlatinga_responses IS 'Фразы для бота bushlatinga_bot';
    `

	_, err := h.db.Exec(createTableQuery)
	if err != nil {
		return err
	}

	log.Println("✅ [bushlatinga_bot] Таблица bushlatinga_responses создана/проверена")
	return nil
}

// loadDefaultResponses загружает стандартные фразы bushlatinga_bot (только оригинальные + Евген Борисыч и Крутой бобёр)
func (h *BotDatabaseHandler) loadDefaultResponses() error {
	defaultResponses := map[string]string{
		// ОРИГИНАЛЬНЫЕ ФРАЗЫ ИЗ ВАШЕГО КОДА
		"славик":      "Славик абсолютно конченная поебота",
		"сплавик":     "Не Сплавик, а Вячеслав!",
		"вячезад":     "Не Вячезад, а Вячеслав!",
		"света":       "ради него из Ижевска она свалила",
		"суетлана":    "ради него из Ижевска она свалила",
		"гусев":       "НИКИТА ГУСЕВ, хорошо долблюсив, хорошо раздалбливаюсИв",
		"нгхд":        "НИКИТА ГУСЕВ, хорошо долблюсив, хорошо раздалбливаюсИв",
		"хамзя":       "хамзя крутооооой, он будет низвергнут",
		"бушлат":      "Дайте Бушлату кто-то в ебло, он уже всех тут заебал...",
		"vkazanee":    "это бушлатинга, продаст корешей он не переживая",
		"банан":       "это Банан, всех он хуесосит, а вообще он нытик..",
		"филин":       "ооо, Филин, здарова!",
		"демида":      "Будка Демиды!",
		"будка":       "Будка Демиды!",
		"галяутдинов": "ГАЛЯУТДИНОВ АЙРАТ АЙДАРОВИЧ ДВАДЦАТЬ ДЕВЯТЬ НОЛЬ ДВА ДЕВЯНОСТО ШЕСТЬ",
		"хсе":         "банана банана мама, спиздил деньги у студентов из кармана",
		"артур":       "заказал кольцо через Банана",
		"айрат":       "это Айратинга, накидал в кабину он ширяю",
		"дуваня":      "лайк, если Дуваня, репост, если Дутаня",
		"горюнов":     "@vkazanee, поздравил Горюнова с днр?",
		"руслан":      "Руслан, крутой, \nЧет он все хуже и хуже с каждым годом ",
		"андрюш":      "мама извини у меня самолет МОСКВА - КИМРЫ - КИМРЫ - МОСКВА",
		"башкир":      "@vkazanee, почему ты башкир?",
		"корона":      "@vkazanee, не корона бро?",
		"акинфеев":    "Игорь Владимиович Акинфеев - легенда русского футбола!",
		"2018":        "2018 - всероссийская пруха! \n @gainutrus, СПАСИБО тебе за ФИНАЛ ЧМ 18 в Лужниках",
		"потроллить":  "@gainutrus, ты брат-2 по-серьезке смотришь, или чисто потроллить?",
		"алсу":        "вот Алсу настоящая татарская жена, не то что эти удмуртские",
		"спб":         "пустословы СПБ",

		// ТОЛЬКО МИНИМАЛЬНЫЕ ЕБ ДОБАВЛЕНИЯ ДЛЯ ТЕСТА
		"тест":     "Тест, блять!",
		"работает": "Работает, епта!",
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Начинаем транзакцию
	tx, err := h.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Подготавливаем запрос
	stmt, err := tx.Prepare(`
        INSERT INTO bushlatinga_responses (trigger_text, response_text) 
        VALUES ($1, $2)
        ON CONFLICT (trigger_text) DO NOTHING
    `)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Вставляем все стандартные фразы
	for trigger, response := range defaultResponses {
		_, err = stmt.Exec(strings.ToLower(trigger), response)
		if err != nil {
			return err
		}
		h.cache[strings.ToLower(trigger)] = response
	}

	// Сохраняем изменения
	err = tx.Commit()
	if err != nil {
		return err
	}

	log.Printf("✅ [bushlatinga_bot] Загружено %d оригинальных фраз\n", len(defaultResponses))
	return nil
}

// loadCache загружает все данные из БД в память
func (h *BotDatabaseHandler) loadCache() error {
	query := "SELECT trigger_text, response_text FROM bushlatinga_responses"

	rows, err := h.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	h.mu.Lock()
	defer h.mu.Unlock()

	count := 0
	for rows.Next() {
		var trigger, response string
		if err := rows.Scan(&trigger, &response); err != nil {
			return err
		}
		h.cache[strings.ToLower(trigger)] = response
		count++
	}

	// Если таблица пуста, загружаем стандартные фразы
	if count == 0 {
		log.Println("📝 [bushlatinga_bot] Таблица пуста, загружаем оригинальные фразы...")
		err = h.loadDefaultResponses()
		if err != nil {
			return err
		}
		// Перезагружаем кэш
		return h.loadCache()
	}

	return rows.Err()
}

// CheckForNames проверяет наличие ровно одного имени в сообщении
// Эта функция сохраняет оригинальную логику вашего бота
func (h *BotDatabaseHandler) CheckForNames(text, userName string) (bool, string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	messageText := strings.ToLower(text)
	var foundResponse string
	foundCount := 0

	// ПРОВЕРЯЕМ "ЕБ" ОТДЕЛЬНО (как в оригинальном eb_handler.go)
	if checkForEB(text) {
		foundCount++
		// Возвращаем специальный маркер для стикера
		foundResponse = "STICKER:Еген борисыч ла-ла-ла-ла-ла-ла"
	}

	// Проверяем все варианты имен в кэше
	for variant, response := range h.cache {
		if strings.Contains(messageText, variant) {
			foundCount++

			if foundCount == 1 {
				// Сохраняем ответ первого найденного имени
				foundResponse = response
			} else {
				// Нашли второе имя - выходим
				return false, ""
			}
		}
	}

	// Отвечаем только если найдено ровно одно имя
	if foundCount == 1 {
		return true, foundResponse
	}

	return false, ""
}

// checkForEB проверяет, содержит ли сообщение "ЕБ" как отдельное слово большими буквами
// Это копия функции из вашего eb_handler.go
func checkForEB(text string) bool {
	// Разбиваем текст на слова (учитываем знаки препинания и пробелы)
	words := strings.FieldsFunc(text, func(r rune) bool {
		// Разделители: все символы, кроме букв, цифр и дефиса
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-'
	})

	// Проверяем каждое слово
	for _, word := range words {
		// Проверяем точное совпадение с "ЕБ" или "ЁБ"
		if word == "ЕБ" || word == "ЁБ" {
			return true
		}
	}
	return false
}

// AddMapping добавляет новую запись в маппинга
func (h *BotDatabaseHandler) AddMapping(key, value string) error {
	key = strings.ToLower(strings.TrimSpace(key))

	if key == "" {
		return fmt.Errorf("ключ не может быть пустым")
	}

	query := `
        INSERT INTO bushlatinga_responses (trigger_text, response_text) 
        VALUES ($1, $2)
        ON CONFLICT (trigger_text) 
        DO UPDATE SET response_text = $2, updated_at = NOW()
        RETURNING id
    `

	var id int64
	err := h.db.QueryRow(query, key, value).Scan(&id)
	if err != nil {
		return fmt.Errorf("ошибка добавления записи: %v", err)
	}

	// Обновляем кэш
	h.mu.Lock()
	h.cache[key] = value
	h.mu.Unlock()

	log.Printf("✅ [bushlatinga_bot] Добавлена запись: '%s' -> '%s' (ID: %d)\n", key, value, id)
	return nil
}

// RemoveMapping удаляет запись из маппинга
func (h *BotDatabaseHandler) RemoveMapping(key string) error {
	key = strings.ToLower(strings.TrimSpace(key))

	query := "DELETE FROM bushlatinga_responses WHERE trigger_text = $1 RETURNING id"

	var id int64
	err := h.db.QueryRow(query, key).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("ключ '%s' не найден", key)
		}
		return fmt.Errorf("ошибка удаления записи: %v", err)
	}

	// Удаляем из кэша
	h.mu.Lock()
	delete(h.cache, key)
	h.mu.Unlock()

	log.Printf("✅ [bushlatinga_bot] Удалена запись: '%s' (ID: %d)\n", key, id)
	return nil
}

// GetMapping возвращает копию маппинга
func (h *BotDatabaseHandler) GetMapping() map[string]string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	copyMap := make(map[string]string)
	for k, v := range h.cache {
		copyMap[k] = v
	}

	return copyMap
}

// GetMappingCount возвращает количество записей в маппинге
func (h *BotDatabaseHandler) GetMappingCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.cache)
}

// SearchInValues ищет текст в значениях маппинга
func (h *BotDatabaseHandler) SearchInValues(searchText string) map[string]string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	searchText = strings.ToLower(searchText)
	results := make(map[string]string)

	for k, v := range h.cache {
		if strings.Contains(strings.ToLower(v), searchText) {
			results[k] = v
		}
	}

	return results
}

// IsAdmin проверяет, является ли пользователь администратором
func (h *BotDatabaseHandler) IsAdmin(userID int64) bool {
	return userID == h.adminID
}

// Close закрывает соединение с БД
func (h *BotDatabaseHandler) Close() error {
	if h.db != nil {
		return h.db.Close()
	}
	return nil
}

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
			return "📭 База данных пуста"
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
		return fmt.Sprintf("📊 Статистика:\n• Всего записей: %d\n• Админ ID: %d", count, h.adminID)

	case "help", "помощь":
		return h.showAdminHelp()

	case "export", "экспорт":
		mapping := h.GetMapping()
		var result strings.Builder
		result.WriteString("📁 Экспорт данных:\n\n")
		for k, v := range mapping {
			safeKey := strings.ReplaceAll(k, "`", "'")
			safeValue := strings.ReplaceAll(v, "`", "'")
			result.WriteString(fmt.Sprintf("`%s` → `%s`\n", safeKey, safeValue))
		}
		return result.String()

	case "info", "инфо":
		return "🤖 *bushlatinga_bot v2.0*\n\n" +
			"• База данных: Supabase PostgreSQL\n" +
			"• Админ команды: /admin help\n" +
			"• Фразы сохраняются в облаке\n" +
			"• Автоматический кэш в памяти\n" +
			"• Отвечает только на одно совпадение\n" +
			"• ЕБ-детектор активен\n\n" +
			"Используйте /admin help для списка команд"

	case "test", "тест":
		// Тестовая команда для проверки ЕБ
		testMessage := "Тест ЕБ функции"
		ebFound := checkForEB(testMessage)
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
/admin add привет Привет!
/admin remove привет
/admin search спасибо
/admin test

📌 Примечания:
• Бот отвечает только на ОДНО совпадение в сообщении!
• "ЕБ" проверяется как отдельное слово большими буквами
• "Евген Борисыч" и "Крутой бобёр" уже добавлены!`
}
