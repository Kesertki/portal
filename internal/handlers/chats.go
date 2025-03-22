package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Kesertki/portal/internal/storage"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type Chat struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Title     string `json:"title"`
	Timestamp int64  `json:"timestamp"`
	IsPinned  bool   `json:"is_pinned"`
}

type ChatMessage struct {
	ID         string          `json:"id"`
	ChatID     string          `json:"chat_id"`
	Sender     string          `json:"sender"`
	SenderRole string          `json:"sender_role"`
	Content    string          `json:"content"`
	Timestamp  int64           `json:"timestamp"`
	Tools      json.RawMessage `json:"tools,omitempty"`
}

type ChatPin struct {
	ChatID string `json:"chat_id"`
	UserID string `json:"user_id"`
}

func CreateChat(c echo.Context) error {
	db, err := storage.ConnectToStorage()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Storage connection failed"})
	}
	defer db.Close()

	chat := new(Chat)
	if err := c.Bind(chat); err != nil {
		return err
	}

	chat.ID = uuid.New().String()
	chat.Timestamp = time.Now().Unix()

	_, err = db.Exec("INSERT INTO chats(id, user_id, title, timestamp) VALUES(?, ?, ?, ?)",
		chat.ID, chat.UserID, chat.Title, chat.Timestamp)
	if err != nil {
		log.Println(err)
		return err
	}

	return c.JSON(http.StatusCreated, chat)
}

func PinChat(c echo.Context) error {
	db, err := storage.ConnectToStorage()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Storage connection failed"})
	}
	defer db.Close()

	chatPin := new(ChatPin)
	if err := c.Bind(chatPin); err != nil {
		log.Println(err)
		return err
	}

	_, err = db.Exec("INSERT INTO chats_pins(chat_id, user_id) VALUES(?, ?) ON CONFLICT(chat_id, user_id) DO NOTHING",
		chatPin.ChatID, chatPin.UserID)
	if err != nil {
		log.Println(err)
		return err
	}

	return c.NoContent(http.StatusOK)
}

func UnpinChat(c echo.Context) error {
	db, err := storage.ConnectToStorage()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Storage connection failed"})
	}
	defer db.Close()

	chatPin := new(ChatPin)
	if err := c.Bind(chatPin); err != nil {
		log.Panicln(err)
		return err
	}

	_, err = db.Exec("DELETE FROM chats_pins WHERE chat_id = ? AND user_id = ?", chatPin.ChatID, chatPin.UserID)
	if err != nil {
		log.Println(err)
		return err
	}

	return c.NoContent(http.StatusOK)
}

func DeleteChat(c echo.Context) error {
	db, err := storage.ConnectToStorage()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Storage connection failed"})
	}
	defer db.Close()

	chatID := c.QueryParam("id")

	_, err = db.Exec("DELETE FROM chats WHERE id = ?", chatID)
	if err != nil {
		log.Println(err)
		return err
	}

	return c.NoContent(http.StatusOK)
}

func GetChats(c echo.Context) error {
	db, err := storage.ConnectToStorage()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Storage connection failed"})
	}
	defer db.Close()

	userID := c.QueryParam("user_id")

	// rows, err := db.Query("SELECT id, user_id, title, timestamp FROM chats WHERE user_id = ? ORDER BY timestamp DESC", userID)
	// if err != nil {
	// 	log.Println(err)
	// 	return err
	// }
	// defer rows.Close()

	rows, err := db.Query(`
		SELECT
			c.id,
			c.user_id,
			c.title,
			c.timestamp,
			CASE
				WHEN cp.id IS NOT NULL THEN 1
				ELSE 0
			END AS is_pinned
		FROM
			chats c
		LEFT JOIN
			chats_pins cp ON c.id = cp.chat_id AND c.user_id = cp.user_id
		WHERE
			c.user_id = ?;
	`, userID)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	chats := []Chat{}
	for rows.Next() {
		var chat Chat
		var isPinnedInt int
		if err := rows.Scan(&chat.ID, &chat.UserID, &chat.Title, &chat.Timestamp, &isPinnedInt); err != nil {
			log.Println(err)
			return err
		}
		chat.IsPinned = isPinnedInt == 1
		chats = append(chats, chat)
	}

	return c.JSON(http.StatusOK, chats)
}

func CreateChatMessage(c echo.Context) error {
	db, err := storage.ConnectToStorage()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Storage connection failed"})
	}
	defer db.Close()

	chatMessage := new(ChatMessage)
	if err := c.Bind(chatMessage); err != nil {
		return err
	}

	chatMessage.ID = uuid.New().String()
	chatMessage.Timestamp = time.Now().Unix()

	_, err = db.Exec("INSERT INTO messages (id, chat_id, sender, sender_role, content, timestamp, tools) VALUES(?, ?, ?, ?, ?, ?, ?)",
		chatMessage.ID, chatMessage.ChatID, chatMessage.Sender, chatMessage.SenderRole, chatMessage.Content, chatMessage.Timestamp, chatMessage.Tools)
	if err != nil {
		log.Println(err)
		return err
	}

	// Update chat timestamp
	_, err = db.Exec("UPDATE chats SET timestamp = ? WHERE id = ?", chatMessage.Timestamp, chatMessage.ChatID)
	if err != nil {
		log.Println(err)
	}

	return c.JSON(http.StatusCreated, chatMessage)
}

func GetChatInfo(c echo.Context) error {
	db, err := storage.ConnectToStorage()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Storage connection failed"})
	}
	defer db.Close()

	chatID := c.QueryParam("chat_id")
	userID := c.QueryParam("user_id")

	row := db.QueryRow(`
		SELECT
			c.id,
			c.user_id,
			c.title,
			c.timestamp,
			CASE
				WHEN cp.id IS NOT NULL THEN 1
				ELSE 0
			END AS is_pinned
		FROM
			chats c
		LEFT JOIN
			chats_pins cp ON c.id = cp.chat_id AND c.user_id = cp.user_id
		WHERE
			c.user_id = ? AND c.id = ?;
	`, userID, chatID)

	var chat Chat
	if err := row.Scan(&chat.ID, &chat.UserID, &chat.Title, &chat.Timestamp, &chat.IsPinned); err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Chat not found"})
		}
		log.Println("Error scanning row:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	return c.JSON(http.StatusOK, chat)
}

func GetChatMessages(c echo.Context) error {
	db, err := storage.ConnectToStorage()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Storage connection failed"})
	}
	defer db.Close()

	chatID := c.QueryParam("chat_id")
	rows, err := db.Query("SELECT id, chat_id, sender, sender_role, content, timestamp, tools FROM messages WHERE chat_id = ? ORDER BY timestamp DESC", chatID)
	if err != nil {
		log.Println(err)
		return err
	}
	defer rows.Close()

	messages := []ChatMessage{}
	for rows.Next() {
		var message ChatMessage
		var tools []byte // Use a byte slice to scan the JSON string
		if err := rows.Scan(&message.ID, &message.ChatID, &message.Sender, &message.SenderRole, &message.Content, &message.Timestamp, &tools); err != nil {
			log.Println(err)
			return err
		}
		if tools != nil {
			message.Tools = json.RawMessage(tools) // Convert the byte slice to json.RawMessage
		}
		messages = append(messages, message)
	}

	return c.JSON(http.StatusOK, messages)
}
