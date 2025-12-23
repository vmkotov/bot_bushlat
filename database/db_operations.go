package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// CheckForNames проверяет наличие ровно одного имени в сообщении (ПРЯМО ИЗ БД!)
func (h *BotDatabaseHandler) CheckForNames(text, userName string) (bool, string) {
	messageText := strings.ToLower(text)
	
	// ПРОВЕРЯЕМ "ЕБ" ОТДЕЛЬНО
	if CheckForEB(text) {
		// Возвращаем специальный маркер для стикера
		return true, "STICKER:Еген борисыч ла-ла-ла-ла-ла-ла"
	}

	// Получаем все записи из БД
	query := "SELECT trigger_text, response_text FROM bushlatinga_bot.bushlatinga_responses"
	rows, err := h.db.Query(query)
	if err != nil {
		log.Printf("❌ Ошибка запроса к БД: %v", err)
		return false, ""
	}
	defer rows.Close()

	var foundResponse string
	foundCount := 0

	// Проверяем каждую запись из БД
	for rows.Next() {
		var trigger, response string
		if err := rows.Scan(&trigger, &response); err != nil {
			log.Printf("❌ Ошибка чтения строки: %v", err)
			continue
		}

		if strings.Contains(messageText, strings.ToLower(trigger)) {
			foundCount++

			if foundCount == 1 {
				// Сохраняем ответ первого найденного имени
				foundResponse = response
			} else {
				// Нашли второе имя - выходим
				rows.Close()
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

// AddMapping добавляет новую запись в маппинг
func (h *BotDatabaseHandler) AddMapping(key, value string) error {
	key = strings.ToLower(strings.TrimSpace(key))

	if key == "" {
		return fmt.Errorf("ключ не может быть пустым")
	}

	query := `
        INSERT INTO bushlatinga_bot.bushlatinga_responses (trigger_text, response_text) 
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

	log.Printf("✅ [bushlatinga_bot] Добавлена запись: '%s' -> '%s' (ID: %d)\n", key, value, id)
	return nil
}

// RemoveMapping удаляет запись из маппинга
func (h *BotDatabaseHandler) RemoveMapping(key string) error {
	key = strings.ToLower(strings.TrimSpace(key))

	query := "DELETE FROM bushlatinga_bot.bushlatinga_responses WHERE trigger_text = $1 RETURNING id"

	var id int64
	err := h.db.QueryRow(query, key).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("ключ '%s' не найден", key)
		}
		return fmt.Errorf("ошибка удаления записи: %v", err)
	}

	log.Printf("✅ [bushlatinga_bot] Удалена запись: '%s' (ID: %d)\n", key, id)
	return nil
}

// GetMapping возвращает все записи из БД
func (h *BotDatabaseHandler) GetMapping() map[string]string {
	query := "SELECT trigger_text, response_text FROM bushlatinga_bot.bushlatinga_responses"
	rows, err := h.db.Query(query)
	if err != nil {
		log.Printf("❌ Ошибка получения маппинга: %v", err)
		return make(map[string]string)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		result[key] = value
	}

	return result
}

// SearchInValues ищет текст в значениях маппинга
func (h *BotDatabaseHandler) SearchInValues(searchText string) map[string]string {
	query := "SELECT trigger_text, response_text FROM bushlatinga_bot.bushlatinga_responses WHERE LOWER(response_text) LIKE $1"
	
	rows, err := h.db.Query(query, "%"+strings.ToLower(searchText)+"%")
	if err != nil {
		log.Printf("❌ Ошибка поиска: %v", err)
		return make(map[string]string)
	}
	defer rows.Close()

	results := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		results[key] = value
	}

	return results
}

// GetMappingCount возвращает количество записей в маппинге
func (h *BotDatabaseHandler) GetMappingCount() int {
	query := "SELECT COUNT(*) FROM bushlatinga_bot.bushlatinga_responses"
	
	var count int
	err := h.db.QueryRow(query).Scan(&count)
	if err != nil {
		log.Printf("❌ Ошибка подсчета записей: %v", err)
		return 0
	}
	
	return count
}

// loadCache теперь только проверяет наличие таблицы
func (h *BotDatabaseHandler) loadCache() error {
	// Просто проверяем, что таблица существует
	query := "SELECT COUNT(*) FROM bushlatinga_bot.bushlatinga_responses"
	
	var count int
	err := h.db.QueryRow(query).Scan(&count)
	if err != nil {
		// Если таблицы нет, это нормально - она создастся при первой записи
		log.Println("📝 Таблица bushlatinga_responses еще не содержит данных")
		return nil
	}
	
	log.Printf("✅ В таблице bushlatinga_responses найдено %d записей", count)
	return nil
}
