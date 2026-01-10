package bot

import (
	"fmt"
	"log"
	"strings"
	"telegram-bot/database"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleUpdate обрабатывает все входящие обновления
func HandleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	// Обрабатываем сообщения
	if update.Message != nil {
		handleMessage(bot, update.Message)
	}

	// Обрабатываем callback-запросы (нажатия на кнопки)
	if update.CallbackQuery != nil {
		handleCallbackQuery(bot, update.CallbackQuery)
	}
}

// handleMessage обрабатывает текстовые сообщения
func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	userID := message.From.ID
	text := message.Text

	// Сохраняем пользователя
	err := database.SaveUser(
		userID,
		message.From.FirstName,
		message.From.LastName,
		message.From.UserName,
	)
	if err != nil {
		log.Printf("Ошибка сохранения пользователя: %v", err)
	}

	// Команды бота
	if strings.HasPrefix(text, "/") {
		handleCommand(bot, message)
		return
	}

	// Сохраняем обычное сообщение
	err = database.SaveMessage(chatID, userID, text)
	if err != nil {
		log.Printf("Ошибка сохранения сообщения: %v", err)
		sendMessage(bot, chatID, "❌ Ошибка сохранения сообщения")
		return
	}

	// Подтверждаем сохранение
	reply := fmt.Sprintf("✅ Сообщение сохранено!\n\n📝 Ваш текст: %s", text)
	sendMessage(bot, chatID, reply)
}

// handleCommand обрабатывает команды бота
func handleCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	command := message.Command()
	args := message.CommandArguments()

	switch command {
	case "start":
		handleStartCommand(bot, message)
	case "help":
		handleHelpCommand(bot, chatID)
	case "get":
		handleGetCommand(bot, chatID, args)
	case "all":
		handleAllCommand(bot, chatID)
	case "count":
		handleCountCommand(bot, chatID)
	case "delete":
		handleDeleteCommand(bot, chatID, args)
	case "clear":
		handleClearCommand(bot, chatID)
	default:
		sendMessage(bot, chatID, "🤔 Неизвестная команда. Используйте /help для списка команд")
	}
}

// handleStartCommand обрабатывает команду /start
func handleStartCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	userName := message.From.FirstName

	welcomeText := fmt.Sprintf(`
👋 Привет, %s!

Я бот для сохранения и управления текстовыми сообщениями.

📌 **Что я умею:**
• Сохранять все ваши текстовые сообщения
• Показывать сохраненные сообщения
• Искать сообщения по ключевым словам
• Удалять сообщения

📋 **Доступные команды:**
/help - Показать справку
/get [N] - Показать последние N сообщений (по умолчанию 5)
/all - Показать все сообщения
/count - Показать количество сохраненных сообщений
/delete [ID] - Удалить сообщение по ID
/clear - Удалить все сообщения (подтверждение)

Просто отправьте мне любой текст, и я его сохрану!
    `, userName)

	sendMessage(bot, chatID, welcomeText)
}

// handleHelpCommand обрабатывает команду /help
func handleHelpCommand(bot *tgbotapi.BotAPI, chatID int64) {
	helpText := `
📋 **Список команд:**

• /start - Начать работу с ботом
• /help - Показать эту справку

💾 **Работа с сообщениями:**
• /get [N] - Показать последние N сообщений
  Пример: /get 10
• /all - Показать все сообщения
• /count - Показать количество сообщений
• /delete [ID] - Удалить сообщение по ID
• /clear - Удалить все сообщения

💡 **Как использовать:**
1. Просто отправьте текст - он сохранится
2. Используйте /get для просмотра
3. Используйте /delete для удаления
    `
	sendMessage(bot, chatID, helpText)
}

// handleGetCommand обрабатывает команду /get
func handleGetCommand(bot *tgbotapi.BotAPI, chatID int64, args string) {
	limit := 5 // по умолчанию 5 сообщений
	if args != "" {
		var n int
		_, err := fmt.Sscanf(args, "%d", &n)
		if err == nil && n > 0 {
			limit = n
		}
	}

	messages, err := database.GetLastMessages(chatID, limit)
	if err != nil {
		log.Printf("Ошибка получения сообщений: %v", err)
		sendMessage(bot, chatID, "❌ Ошибка получения сообщений")
		return
	}

	if len(messages) == 0 {
		sendMessage(bot, chatID, "📭 Сообщений пока нет")
		return
	}

	var response strings.Builder
	response.WriteString(fmt.Sprintf("📜 **Последние %d сообщений:**\n\n", len(messages)))

	for i, msg := range messages {
		timeStr := msg.CreatedAt.Format("02.01.2006 15:04")
		response.WriteString(fmt.Sprintf("**%d.** [ID: %d] %s\n", i+1, msg.ID, timeStr))
		response.WriteString(fmt.Sprintf("```\n%s\n```\n\n", msg.Text))
	}

	sendMessage(bot, chatID, response.String())
}

// handleAllCommand обрабатывает команду /all
func handleAllCommand(bot *tgbotapi.BotAPI, chatID int64) {
	messages, err := database.GetMessagesByChatID(chatID)
	if err != nil {
		log.Printf("Ошибка получения сообщений: %v", err)
		sendMessage(bot, chatID, "❌ Ошибка получения сообщений")
		return
	}

	if len(messages) == 0 {
		sendMessage(bot, chatID, "📭 Сообщений пока нет")
		return
	}

	// Разбиваем на части, если сообщений много
	const maxMessagesPerPage = 10
	totalPages := (len(messages) + maxMessagesPerPage - 1) / maxMessagesPerPage

	for page := 0; page < totalPages; page++ {
		start := page * maxMessagesPerPage
		end := start + maxMessagesPerPage
		if end > len(messages) {
			end = len(messages)
		}

		var response strings.Builder
		response.WriteString(fmt.Sprintf("📚 **Все сообщения (страница %d/%d):**\n\n", page+1, totalPages))

		for i := start; i < end; i++ {
			msg := messages[i]
			timeStr := msg.CreatedAt.Format("02.01.2006 15:04")
			response.WriteString(fmt.Sprintf("**%d.** [ID: %d] %s\n", i+1, msg.ID, timeStr))
			response.WriteString(fmt.Sprintf("```\n%s\n```\n\n", msg.Text))
		}

		sendMessage(bot, chatID, response.String())
		time.Sleep(100 * time.Millisecond) // Чтобы не превысить лимиты Telegram
	}
}

// handleCountCommand обрабатывает команду /count
func handleCountCommand(bot *tgbotapi.BotAPI, chatID int64) {
	count, err := database.GetMessageCount(chatID)
	if err != nil {
		log.Printf("Ошибка получения количества: %v", err)
		sendMessage(bot, chatID, "❌ Ошибка получения количества сообщений")
		return
	}

	response := fmt.Sprintf("📊 **Статистика:**\n\n✅ Сохранено сообщений: **%d**", count)
	sendMessage(bot, chatID, response)
}

// handleDeleteCommand обрабатывает команду /delete
func handleDeleteCommand(bot *tgbotapi.BotAPI, chatID int64, args string) {
	if args == "" {
		sendMessage(bot, chatID, "❌ Укажите ID сообщения для удаления\nПример: /delete 42")
		return
	}

	var messageID int64
	_, err := fmt.Sscanf(args, "%d", &messageID)
	if err != nil {
		sendMessage(bot, chatID, "❌ Неверный формат ID")
		return
	}

	err = database.DeleteMessage(messageID)
	if err != nil {
		log.Printf("Ошибка удаления сообщения: %v", err)
		sendMessage(bot, chatID, "❌ Ошибка удаления сообщения")
		return
	}

	sendMessage(bot, chatID, fmt.Sprintf("✅ Сообщение с ID %d успешно удалено", messageID))
}

// handleClearCommand обрабатывает команду /clear
func handleClearCommand(bot *tgbotapi.BotAPI, chatID int64) {
	// Создаем клавиатуру с подтверждением
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да, удалить все", "clear_confirm"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Нет, отмена", "clear_cancel"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "⚠️ **Внимание!**\n\nВы действительно хотите удалить ВСЕ сообщения?\nЭто действие нельзя отменить.")
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "Markdown"

	bot.Send(msg)
}

// handleCallbackQuery обрабатывает нажатия на inline-кнопки
func handleCallbackQuery(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery) {
	chatID := query.Message.Chat.ID
	data := query.Data

	// Отвечаем на callback (убираем "часики")
	callback := tgbotapi.NewCallback(query.ID, "")
	bot.Send(callback)

	switch data {
	case "clear_confirm":
		// Удаляем все сообщения для этого чата
		_, err := database.DB.Exec("DELETE FROM messages WHERE chat_id = ?", chatID)
		if err != nil {
			sendMessage(bot, chatID, "❌ Ошибка удаления сообщений")
			return
		}

		// Удаляем кнопки из сообщения
		editMsg := tgbotapi.NewEditMessageReplyMarkup(
			chatID,
			query.Message.MessageID,
			tgbotapi.InlineKeyboardMarkup{},
		)
		bot.Send(editMsg)

		sendMessage(bot, chatID, "✅ Все сообщения успешно удалены")

	case "clear_cancel":
		// Удаляем кнопки из сообщения
		editMsg := tgbotapi.NewEditMessageText(
			chatID,
			query.Message.MessageID,
			"❌ Удаление отменено",
		)
		bot.Send(editMsg)
	}
}

// sendMessage отправляет сообщение пользователю
func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	// Если сообщение слишком длинное, разбиваем его
	if len(text) > 4000 {
		messages := splitMessage(text, 4000)
		for _, part := range messages {
			msg.Text = part
			bot.Send(msg)
			time.Sleep(50 * time.Millisecond)
		}
		return
	}

	bot.Send(msg)
}

// splitMessage разбивает длинное сообщение на части
func splitMessage(text string, maxLength int) []string {
	var parts []string
	for len(text) > maxLength {
		// Ищем последний перенос строки перед maxLength
		splitAt := strings.LastIndex(text[:maxLength], "\n")
		if splitAt == -1 {
			splitAt = maxLength
		}
		parts = append(parts, text[:splitAt])
		text = text[splitAt:]
	}
	parts = append(parts, text)
	return parts
}
