package database

import (
	"database/sql"
	"fmt"
	"log"
	"sync"

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
	cache   map[string]string // Оставим структуру для совместимости, но не будем использовать
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

	// Просто проверяем таблицу (не загружаем кэш)
	err = handler.loadCache()
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки таблицы: %v", err)
	}

	return handler, nil
}

// GetEBStickerID возвращает ID стикера для "ЕБ"
func (h *BotDatabaseHandler) GetEBStickerID() string {
	return EBStickerID
}

// DB возвращает указатель на соединение с БД (для использования в main.go)
func (h *BotDatabaseHandler) DB() *sql.DB {
	return h.db
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
