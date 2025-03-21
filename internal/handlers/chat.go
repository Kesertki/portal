package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Kesertki/portal/internal/storage"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type Chat struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Title     string `json:"title"`
	Timestamp int    `json:"timestamp"`
}

type ChatMessage struct {
	ID         string          `json:"id"`
	ChatID     string          `json:"chat_id"`
	Sender     string          `json:"sender"`
	SenderRole string          `json:"sender_role"`
	Content    string          `json:"content"`
	Timestamp  int             `json:"timestamp"`
	Feedback   int             `json:"feedback"`
	Tools      json.RawMessage `json:"tools"`
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

	_, err = db.Exec("INSERT INTO chats(id, user_id, title, timestamp) VALUES(?, ?, ?, ?)",
		chat.ID, chat.UserID, chat.Title, chat.Timestamp)
	if err != nil {
		log.Println(err)
		return err
	}

	return c.JSON(http.StatusCreated, chat)
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
