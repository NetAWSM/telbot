package main

import (
	"log"
	"os"
	"telegram-bot/bot"
	"telegram-bot/database"
)

func main() {
	// Получаем токен бота из переменной окружения
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN не установлен")
	}

	// Инициализируем базу данных
	err := database.InitDB("messages.db")
	if err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}
	defer database.DB.Close()

	// Создаем бота
	telegramBot, err := bot.NewBot(botToken)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	// Запускаем бота
	telegramBot.Start()
}
