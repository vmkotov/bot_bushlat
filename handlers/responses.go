package handlers

import (
	"math/rand"
)

// GetRandomEBResponse возвращает случайный ответ на упоминание Евгена Борисыча
func GetRandomEBResponse(userName string) string {
	responses := []string{
		"О, Евген Борисыч!",
		"крутой бобёр",
	}

	if len(responses) == 0 {
		return "Евген Борисыч упомянут! 🎉"
	}

	randomIndex := rand.Intn(len(responses))
	return responses[randomIndex]
}

// GetStickerID возвращает ID единственного стикера для "ЕБ"
func GetStickerID() string {
	// Правильный FileID стикера
	return "CAACAgIAAxkBAANTaUVkrWrIsoO8kVNAifaUqz16ex4AAqqFAAJVF1hIHdoBVVf89Yg2BA"
}
