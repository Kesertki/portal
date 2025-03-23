package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Kesertki/portal/internal/storage"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
)

type Reminder struct {
	ID          string    `json:"id"`
	Message     string    `json:"message"`
	Description string    `json:"description"`
	DueTime     time.Time `json:"due_time"`
	Completed   bool      `json:"completed"`
	WebhookURL  string    `json:"webhook_url"`
}

func SetupReminderApiHandlers(apiGroup *echo.Group, db *sql.DB) {
	log.Info().Msg("Initializing Reminders API")

	apiGroup.POST("/reminders.add", CreateReminderHandler(db))
	apiGroup.GET("/reminders.list", ListRemindersHandler(db))
	apiGroup.POST("/reminders.complete", CompleteReminderHandler(db))
	apiGroup.POST("/reminders.delete", DeleteReminderHandler(db))
	apiGroup.GET("/reminders.info", GetReminderInfoHandler(db))
}

func CreateReminderHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		reminder := new(Reminder)
		if err := c.Bind(reminder); err != nil {
			return err
		}

		reminder.ID = uuid.New().String()

		_, err := db.Exec("INSERT INTO reminders(id, message, description, due_time, completed, webhook_url) VALUES(?, ?, ?, ?, ?, ?)",
			reminder.ID, reminder.Message, reminder.Description, reminder.DueTime, reminder.Completed, reminder.WebhookURL)
		if err != nil {
			log.Error().Err(err).Msg("Failed to insert reminder")
			return err
		}

		return c.JSON(http.StatusCreated, reminder)
	}
}

func ListRemindersHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		rows, err := db.Query("SELECT id, message, description, due_time, completed, webhook_url FROM reminders")
		if err != nil {
			log.Error().Err(err).Msg("Failed to query reminders")
			return err
		}
		defer rows.Close()

		reminders := []Reminder{}
		for rows.Next() {
			var r Reminder
			if err := rows.Scan(&r.ID, &r.Message, &r.Description, &r.DueTime, &r.Completed, &r.WebhookURL); err != nil {
				log.Error().Err(err).Msg("Failed to scan reminder")
				return err
			}
			reminders = append(reminders, r)
		}
		return c.JSON(http.StatusOK, reminders)
	}
}

func CompleteReminderHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.QueryParam("id")
		_, err := db.Exec("UPDATE reminders SET completed = TRUE WHERE id = ?", id)
		if err != nil {
			log.Error().Err(err).Msg("Failed to update reminder")
			return err
		}

		return c.NoContent(http.StatusOK)
	}
}

func DeleteReminderHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.QueryParam("id")
		_, err := db.Exec("DELETE FROM reminders WHERE id = ?", id)
		if err != nil {
			log.Error().Err(err).Msg("Failed to delete reminder")
			return err
		}

		return c.NoContent(http.StatusOK)
	}
}

func GetReminderInfoHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.QueryParam("id")
		row := db.QueryRow("SELECT id, message, description, due_time, completed, webhook_url FROM reminders WHERE id = ?", id)

		var r Reminder
		if err := row.Scan(&r.ID, &r.Message, &r.Description, &r.DueTime, &r.Completed, &r.WebhookURL); err != nil {
			log.Error().Err(err).Msg("Failed to scan reminder")
			return err
		}

		return c.JSON(http.StatusOK, r)
	}
}

func notifyWebhook(reminder Reminder, webhookURL string) error {
	jsonData, err := json.Marshal(reminder)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal reminder")
		return err
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Error().Err(err).Msg("Failed to send webhook")
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send webhook: %s", resp.Status)
	}

	log.Info().Msgf("Webhook sent for reminder %s", reminder.ID)
	return nil
}

func StartRemindersAgent(wsHandler *WebSocketHandler) {
	db, err := storage.ConnectToStorage()
	if err != nil {
		log.Fatal().Msg("Storage connection failed")
	}
	defer db.Close()

	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		now := time.Now()
		truncatedNow := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())
		log.Info().Msgf("Checking for reminders at %s", truncatedNow.Format("2006-01-02 15:04:05"))

		tx, err := db.Begin()
		if err != nil {
			log.Error().Err(err).Msg("Error starting transaction")
			continue
		}

		rows, err := tx.Query("SELECT id, message, webhook_url FROM reminders WHERE due_time BETWEEN ? AND ? AND completed = FALSE",
			truncatedNow.Format("2006-01-02 15:04:05"),
			truncatedNow.Add(time.Minute).Format("2006-01-02 15:04:05"))
		if err != nil {
			log.Error().Err(err).Msg("Error querying reminders")
			tx.Rollback()
			continue
		}

		for rows.Next() {
			var r Reminder
			if err := rows.Scan(&r.ID, &r.Message, &r.WebhookURL); err != nil {
				log.Error().Err(err).Msg("Error scanning reminder")
				tx.Rollback()
				continue
			}
			log.Info().Msgf("Reminder %s: %s", r.ID, r.Message)

			// Notify via WebSocket
			wsHandler.BroadcastMessage("api.reminders", r.Message)

			// Send webhook
			if r.WebhookURL != "" {
				log.Info().Msgf("Sending webhook for reminder %s to %s", r.ID, r.WebhookURL)
				if err := notifyWebhook(r, r.WebhookURL); err != nil {
					log.Error().Err(err).Msg("Error sending webhook")
				}
			}

			if _, err := tx.Exec("UPDATE reminders SET completed = TRUE WHERE id = ?", r.ID); err != nil {
				log.Error().Err(err).Msg("Error marking reminder as completed")
				tx.Rollback()
				continue
			}
		}
		rows.Close()

		if err := tx.Commit(); err != nil {
			log.Error().Err(err).Msg("Error committing transaction")
		}
	}
}
