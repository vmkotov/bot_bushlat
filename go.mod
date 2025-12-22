module bushlatinga_bot

go 1.21.1

toolchain go1.21.3

require (
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9
)

require github.com/vmkotov/telelog v0.0.0-20251222152736-38edbb74f8b2

// Добавляем replace для локальной разработки (если telelog лежит рядом)

replace github.com/vmkotov/telelog => ../telelog
