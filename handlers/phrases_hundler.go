package handlers

import (
	"strings"
)

// CheckForPhrases проверяет наличие фраз в сообщении
func CheckForPhrases(text, userName string) (bool, string) {
	messageText := strings.ToLower(text)

	isGreeting := checkGreeting(messageText)
	isFarewell := checkFarewell(messageText)
	isHowAreYou := checkHowAreYou(messageText)

	var response string

	switch {
	case isGreeting && isFarewell && isHowAreYou:
		response = "Ты и здороваешься, и прощаешься, и спрашиваешь как дела! Ладно, привет, " + userName + "!"
	case isGreeting && isFarewell:
		response = "Ты и здравствуешь, и прощаешься? Ну привет и пока, " + userName + "!"
	case isGreeting && isHowAreYou:
		response = "Привет! У меня хорошо, " + userName + "!"
	case isGreeting:
		response = "Привет, " + userName + "!"
	case isFarewell && isHowAreYou:
		response = "Пока! И у меня всё хорошо, " + userName + "!"
	case isFarewell:
		response = "Пока, " + userName + "!"
	case isHowAreYou:
		response = "Хорошо, " + userName + "!"
	default:
		return false, ""
	}

	return true, response
}

// checkGreeting проверяет на приветствия
func checkGreeting(text string) bool {
	greetings := []string{
		"привет", "здравствуй", "здрасте", "здаров",
		"добрый день", "доброе утро", "добрый вечер",
		"хай", "hi", "hello", "салют", "прив",
	}

	for _, greeting := range greetings {
		if strings.Contains(text, greeting) {
			return true
		}
	}
	return false
}

// checkFarewell проверяет на прощания
func checkFarewell(text string) bool {
	farewells := []string{
		"пока", "до свидания", "прощай", "счастливо",
		"до встречи", "увидимся", "бай", "bye", "goodbye",
		"чао", "покеда", "всего хорошего",
	}

	for _, farewell := range farewells {
		if strings.Contains(text, farewell) {
			return true
		}
	}
	return false
}

// checkHowAreYou проверяет на вопрос "как дела"
func checkHowAreYou(text string) bool {
	phrases := []string{
		"как дела", "как ты", "как жизнь", "как сам",
		"как поживаешь", "how are you",
	}

	for _, phrase := range phrases {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return false
}
