package database

import (
	"fmt"
	"log"
)

// initializeDatabase создает схемы и таблицы, если их нет
func (h *BotDatabaseHandler) initializeDatabase() error {
	// 1. Создаем схему для бота, если не существует
	createBotSchemaQuery := `
		CREATE SCHEMA IF NOT EXISTS bushlatinga_bot;
		
		COMMENT ON SCHEMA bushlatinga_bot IS 'Схема для данных бота Bushlatinga Bot';
	`

	// 2. Создаем схему main для логов, если не существует
	createMainSchemaQuery := `
		CREATE SCHEMA IF NOT EXISTS main;
		
		COMMENT ON SCHEMA main IS 'Основная схема для логов и системной информации';
	`

	// 3. Создаем таблицу для хранения фраз bushlatinga_bot в его схеме
	createResponsesTableQuery := `
		CREATE TABLE IF NOT EXISTS bushlatinga_bot.bushlatinga_responses (
			id BIGSERIAL PRIMARY KEY,
			trigger_text VARCHAR(100) UNIQUE NOT NULL,
			response_text TEXT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);
		
		CREATE INDEX IF NOT EXISTS idx_bushlatinga_trigger_text 
		ON bushlatinga_bot.bushlatinga_responses(trigger_text);
		
		COMMENT ON TABLE bushlatinga_bot.bushlatinga_responses IS 'Фразы для бота bushlatinga_bot';
	`

	// 4. Создаем таблицу для логов сообщений в схеме main
	createMessagesTableQuery := `
		CREATE TABLE IF NOT EXISTS main.messages_log (
			id BIGSERIAL PRIMARY KEY,
			bot_id BIGINT NOT NULL,
			bot_username VARCHAR(100),
			chat_id BIGINT NOT NULL,
			chat_title VARCHAR(255),
			chat_type VARCHAR(50),
			user_id BIGINT NOT NULL,
			user_name VARCHAR(255),
			user_username VARCHAR(100),
			message_id BIGINT NOT NULL,
			message_text TEXT,
			message_type VARCHAR(50),
			reply_to_message_id BIGINT,
			reply_to_user_id BIGINT,
			has_sticker BOOLEAN DEFAULT FALSE,
			sticker_emoji VARCHAR(100),
			has_photo BOOLEAN DEFAULT FALSE,
			has_document BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			
			CONSTRAINT unique_bot_message UNIQUE(bot_id, chat_id, message_id)
		);
		
		CREATE INDEX IF NOT EXISTS idx_messages_bot_id ON main.messages_log(bot_id);
		CREATE INDEX IF NOT EXISTS idx_messages_chat_id ON main.messages_log(chat_id);
		CREATE INDEX IF NOT EXISTS idx_messages_user_id ON main.messages_log(user_id);
		CREATE INDEX IF NOT EXISTS idx_messages_created_at ON main.messages_log(created_at);
		
		COMMENT ON TABLE main.messages_log IS 'Логи всех сообщений, полученных ботом';
	`

	// 5. Создаем таблицу для статистики бота
	createStatsTableQuery := `
		CREATE TABLE IF NOT EXISTS main.bot_stats (
			id BIGSERIAL PRIMARY KEY,
			bot_id BIGINT NOT NULL,
			bot_username VARCHAR(100),
			total_messages BIGINT DEFAULT 0,
			total_commands BIGINT DEFAULT 0,
			total_name_matches BIGINT DEFAULT 0,
			total_eb_matches BIGINT DEFAULT 0,
			unique_chats BIGINT DEFAULT 0,
			unique_users BIGINT DEFAULT 0,
			last_message_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			
			CONSTRAINT unique_bot_stats UNIQUE(bot_id)
		);
		
		CREATE INDEX IF NOT EXISTS idx_bot_stats_bot_id ON main.bot_stats(bot_id);
		
		COMMENT ON TABLE main.bot_stats IS 'Статистика по ботам';
	`

	// Выполняем все запросы в транзакции
	tx, err := h.db.Begin()
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %v", err)
	}
	defer tx.Rollback()

	// Создаем схемы
	if _, err := tx.Exec(createBotSchemaQuery); err != nil {
		return fmt.Errorf("ошибка создания схемы бота: %v", err)
	}
	log.Println("✅ Схема 'bushlatinga_bot' создана/проверена")

	if _, err := tx.Exec(createMainSchemaQuery); err != nil {
		return fmt.Errorf("ошибка создания схемы main: %v", err)
	}
	log.Println("✅ Схема 'main' создана/проверена")

	// Создаем таблицы
	if _, err := tx.Exec(createResponsesTableQuery); err != nil {
		return fmt.Errorf("ошибка создания таблицы ответов: %v", err)
	}
	log.Println("✅ Таблица 'bushlatinga_bot.bushlatinga_responses' создана/проверена")

	if _, err := tx.Exec(createMessagesTableQuery); err != nil {
		return fmt.Errorf("ошибка создания таблицы логов: %v", err)
	}
	log.Println("✅ Таблица 'main.messages_log' создана/проверена")

	if _, err := tx.Exec(createStatsTableQuery); err != nil {
		return fmt.Errorf("ошибка создания таблицы статистики: %v", err)
	}
	log.Println("✅ Таблица 'main.bot_stats' создана/проверена")

	// Коммитим транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %v", err)
	}

	log.Println("✅ Все схемы и таблицы успешно инициализированы")
	return nil
}
