package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
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

type DeleteChatRequest struct {
	ChatID string `json:"chat_id"`
	UserID string `json:"user_id"`
}

type RenameChatRequest struct {
	ChatID string `json:"chat_id"`
	UserID string `json:"user_id"`
	Title  string `json:"title"`
}

func SetupChatApiHandlers(apiGroup *echo.Group, db *sql.DB) {
	log.Info().Msg("Initializing Chat API")

	apiGroup.POST("/chats.add", CreateChatHandler(db))
	apiGroup.POST("/chats.delete", DeleteChatHandler(db))
	apiGroup.POST("/chats.rename", RenameChatHandler(db))
	apiGroup.GET("/chats.list", GetChatsHandler(db))
	apiGroup.POST("/chats.pin", PinChatHandler(db))
	apiGroup.POST("/chats.unpin", UnpinChatHandler(db))
	apiGroup.GET("/chats.info", GetChatInfoHandler(db))
	apiGroup.POST("/messages.add", CreateChatMessageHandler(db))
	apiGroup.GET("/messages.list", GetChatMessagesHandler(db))
}

func CreateChatHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		chat := new(Chat)
		if err := c.Bind(chat); err != nil {
			return err
		}

		chat.ID = uuid.New().String()
		chat.Timestamp = time.Now().Unix()

		_, err := db.Exec("INSERT INTO chats(id, user_id, title, timestamp) VALUES(?, ?, ?, ?)",
			chat.ID, chat.UserID, chat.Title, chat.Timestamp)
		if err != nil {
			log.Error().Err(err).Msg("Failed to insert chat")
			return err
		}

		return c.JSON(http.StatusCreated, chat)
	}
}

func PinChatHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		chatPin := new(ChatPin)
		if err := c.Bind(chatPin); err != nil {
			log.Error().Err(err).Msg("Failed to bind chat pin")
			return err
		}

		_, err := db.Exec("INSERT INTO chats_pins(chat_id, user_id) VALUES(?, ?) ON CONFLICT(chat_id, user_id) DO NOTHING",
			chatPin.ChatID, chatPin.UserID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to pin chat")
			return err
		}

		return c.NoContent(http.StatusOK)
	}
}

func UnpinChatHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		chatPin := new(ChatPin)
		if err := c.Bind(chatPin); err != nil {
			log.Error().Err(err).Msg("Failed to bind chat pin")
			return err
		}

		_, err := db.Exec("DELETE FROM chats_pins WHERE chat_id = ? AND user_id = ?", chatPin.ChatID, chatPin.UserID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to unpin chat")
			return err
		}

		return c.NoContent(http.StatusOK)
	}
}

func DeleteChatHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		deleteChatRequest := new(DeleteChatRequest)
		if err := c.Bind(deleteChatRequest); err != nil {
			log.Error().Err(err).Msg("Invalid request")
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		}

		result, err := db.Exec("DELETE FROM chats WHERE id = ? AND user_id = ?", deleteChatRequest.ChatID, deleteChatRequest.UserID)
		if err != nil {
			log.Error().Err(err).Msg("Internal server error")
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Error().Err(err).Msg("Internal server error")
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		}

		if rowsAffected == 0 {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Chat not found"})
		}

		return c.NoContent(http.StatusOK)
	}
}

func RenameChatHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		renameChatRequest := new(RenameChatRequest)
		if err := c.Bind(renameChatRequest); err != nil {
			log.Error().Err(err).Msg("Invalid request")
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		}

		result, err := db.Exec("UPDATE chats SET title = ? WHERE id = ? AND user_id = ?", renameChatRequest.Title, renameChatRequest.ChatID, renameChatRequest.UserID)
		if err != nil {
			log.Error().Err(err).Msg("Internal server error")
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Error().Err(err).Msg("Internal server error")
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		}

		if rowsAffected == 0 {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Chat not found"})
		}

		return c.NoContent(http.StatusOK)
	}
}

func GetChatsHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID := c.QueryParam("user_id")

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
			log.Fatal().Err(err).Msg("Failed to query chats")
		}
		defer func() {
			if err := rows.Close(); err != nil {
				log.Error().Err(err).Msg("Error closing rows")
			}
		}()

		chats := []Chat{}
		for rows.Next() {
			var chat Chat
			var isPinnedInt int
			if err := rows.Scan(&chat.ID, &chat.UserID, &chat.Title, &chat.Timestamp, &isPinnedInt); err != nil {
				log.Error().Err(err).Msg("Failed to scan chat")
				return err
			}
			chat.IsPinned = isPinnedInt == 1
			chats = append(chats, chat)
		}

		return c.JSON(http.StatusOK, chats)
	}
}

func CreateChatMessageHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		chatMessage := new(ChatMessage)
		if err := c.Bind(chatMessage); err != nil {
			return err
		}

		chatMessage.ID = uuid.New().String()
		chatMessage.Timestamp = time.Now().Unix()

		_, err := db.Exec("INSERT INTO messages(id, chat_id, sender, sender_role, content, timestamp, tools) VALUES(?, ?, ?, ?, ?, ?, ?)",
			chatMessage.ID, chatMessage.ChatID, chatMessage.Sender, chatMessage.SenderRole, chatMessage.Content, chatMessage.Timestamp, chatMessage.Tools)
		if err != nil {
			log.Error().Err(err).Msg("Failed to insert chat message")
			return err
		}

		// Update chat timestamp
		_, err = db.Exec("UPDATE chats SET timestamp = ? WHERE id = ?", chatMessage.Timestamp, chatMessage.ChatID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to update chat timestamp")
		}

		return c.JSON(http.StatusCreated, chatMessage)
	}
}

func GetChatInfoHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
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
			log.Error().Err(err).Msg("Failed to scan chat info")
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		}

		return c.JSON(http.StatusOK, chat)
	}
}

func GetChatMessagesHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		chatID := c.QueryParam("chat_id")

		rows, err := db.Query("SELECT id, chat_id, sender, sender_role, content, timestamp, tools FROM messages WHERE chat_id = ? ORDER BY timestamp", chatID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to query chat messages")
			return err
		}
		defer func() {
			if err := rows.Close(); err != nil {
				log.Error().Err(err).Msg("Error closing rows")
			}
		}()

		messages := []ChatMessage{}
		for rows.Next() {
			var message ChatMessage
			var tools []byte // Use a byte slice to scan the JSON string
			if err := rows.Scan(&message.ID, &message.ChatID, &message.Sender, &message.SenderRole, &message.Content, &message.Timestamp, &tools); err != nil {
				log.Error().Err(err).Msg("Failed to scan chat message")
				return err
			}
			if tools != nil {
				message.Tools = json.RawMessage(tools) // Convert the byte slice to json.RawMessage
			}
			messages = append(messages, message)
		}

		return c.JSON(http.StatusOK, messages)
	}
}
