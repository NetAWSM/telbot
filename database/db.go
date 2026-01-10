package database

import (
	"database/sql"
	"log"
	"telegram-bot/models"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

// InitDB инициализирует базу данных SQLite
func InitDB(dbPath string) error {
	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	// Проверяем соединение
	if err = DB.Ping(); err != nil {
		return err
	}

	// Создаем таблицы
	if err = createTables(); err != nil {
		return err
	}

	log.Println("База данных успешно подключена")
	return nil
}

// createTables создает необходимые таблицы
func createTables() error {
	// Таблица пользователей
	createUsersTable := `
    CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY,
        first_name TEXT,
        last_name TEXT,
        username TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    )`

	// Таблица сообщений
	createMessagesTable := `
    CREATE TABLE IF NOT EXISTS messages (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        chat_id INTEGER NOT NULL,
        user_id INTEGER NOT NULL,
        text TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES users (id)
    )`

	// Создаем таблицу пользователей
	if _, err := DB.Exec(createUsersTable); err != nil {
		return err
	}

	// Создаем таблицу сообщений
	if _, err := DB.Exec(createMessagesTable); err != nil {
		return err
	}

	// Создаем индекс для быстрого поиска сообщений по chat_id
	createIndex := `
    CREATE INDEX IF NOT EXISTS idx_messages_chat_id 
    ON messages (chat_id)`

	if _, err := DB.Exec(createIndex); err != nil {
		return err
	}

	log.Println("Таблицы успешно созданы")
	return nil
}

// SaveUser сохраняет или обновляет пользователя
func SaveUser(userID int64, firstName, lastName, username string) error {
	// Проверяем, существует ли пользователь
	var exists bool
	err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", userID).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		// Обновляем существующего пользователя
		query := `UPDATE users SET first_name = ?, last_name = ?, username = ? WHERE id = ?`
		_, err = DB.Exec(query, firstName, lastName, username, userID)
	} else {
		// Создаем нового пользователя
		query := `INSERT INTO users (id, first_name, last_name, username) VALUES (?, ?, ?, ?)`
		_, err = DB.Exec(query, userID, firstName, lastName, username)
	}

	return err
}

// SaveMessage сохраняет сообщение в базу данных
func SaveMessage(chatID, userID int64, text string) error {
	query := `INSERT INTO messages (chat_id, user_id, text) VALUES (?, ?, ?)`
	_, err := DB.Exec(query, chatID, userID, text)
	return err
}

// GetMessagesByChatID возвращает все сообщения для конкретного чата
func GetMessagesByChatID(chatID int64) ([]models.Message, error) {
	query := `
    SELECT id, chat_id, user_id, text, created_at 
    FROM messages 
    WHERE chat_id = ? 
    ORDER BY created_at ASC`

	rows, err := DB.Query(query, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		err := rows.Scan(&msg.ID, &msg.ChatID, &msg.UserID, &msg.Text, &msg.CreatedAt)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// GetLastMessages возвращает последние N сообщений
func GetLastMessages(chatID int64, limit int) ([]models.Message, error) {
	query := `
    SELECT id, chat_id, user_id, text, created_at 
    FROM messages 
    WHERE chat_id = ? 
    ORDER BY created_at DESC 
    LIMIT ?`

	rows, err := DB.Query(query, chatID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		err := rows.Scan(&msg.ID, &msg.ChatID, &msg.UserID, &msg.Text, &msg.CreatedAt)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	// Переворачиваем массив, чтобы получить хронологический порядок
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// GetMessageCount возвращает количество сообщений в чате
func GetMessageCount(chatID int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM messages WHERE chat_id = ?`
	err := DB.QueryRow(query, chatID).Scan(&count)
	return count, err
}

// DeleteMessage удаляет сообщение по ID
func DeleteMessage(messageID int64) error {
	query := `DELETE FROM messages WHERE id = ?`
	_, err := DB.Exec(query, messageID)
	return err
}
