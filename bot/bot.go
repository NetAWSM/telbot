package bot

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"telegram-bot/database"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot представляет Telegram бота
type Bot struct {
	API *tgbotapi.BotAPI
}

// NewBot создает нового бота
func NewBot(token string) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	log.Printf("Авторизован как %s", api.Self.UserName)

	return &Bot{API: api}, nil
}

// Start запускает бота
func (b *Bot) Start() {
	// Настраиваем обновления
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.API.GetUpdatesChan(u)

	log.Println("Бот запущен. Ожидание сообщений...")

	// Обработка сигналов для graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем горутину для обработки сообщений
	go func() {
		for update := range updates {
			HandleUpdate(b.API, update)
		}
	}()

	// Ожидаем сигнал остановки
	<-stop
	log.Println("Остановка бота...")

	// Закрываем соединение с базой данных
	if database.DB != nil {
		database.DB.Close()
	}

	log.Println("Бот остановлен")
}
