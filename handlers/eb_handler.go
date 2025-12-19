package handlers

import (
	"strings"
	"unicode"
)

// CheckForEB проверяет, содержит ли сообщение "ЕБ" как отдельное слово большими буквами
func CheckForEB(text string) bool {
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
