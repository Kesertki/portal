package main

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/Kesertki/portal/internal/handlers"
	"github.com/Kesertki/portal/internal/storage"
)

var (
	Version   = "dev" // Default value
	BuildDate = "unknown"
	GitCommit = "unknown"
)

const dbPath = "./data/portal.db"

func main() {
	fmt.Printf("%s v%s (Commit: %s, Built: %s)\n", "[portal]", Version, GitCommit, BuildDate)

	// Configure logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	err := godotenv.Load()
	if err != nil {
		log.Warn().Msg("Error loading .env file")
	}

	storage.ApplyMigrations()

	// Log environment variables
	log.Info().Msgf("DATA_PATH: %s", os.Getenv("DATA_PATH"))

	e := echo.New()

	// CORS configuration to allow all origins
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"}, // Allow all origins
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	// Middleware
	// e.Use(middleware.Logger())
	// Custom logger middleware for Common Log Format (CLF)
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${remote_ip} - - [${time_rfc3339}] \"${method} ${uri} ${protocol}\" ${status} ${bytes_out}\n",
		Output: os.Stderr,
	}))
	e.Use(middleware.Recover())

	e.GET("/api/date.now", handlers.GetCurrentDate)
	e.GET("/api/search.instant", handlers.InstantAnswer)
	e.GET("/api/geolocation", handlers.GetGeoLocation)

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Your portal to the real world :)")
	})

	e.Static("/", "public")

	e.GET("/api/reminders.list", handlers.GetReminders)
	e.POST("/api/reminders.add", handlers.CreateReminder)
	e.POST("/api/reminders.complete", handlers.CompleteReminder)
	e.POST("/api/reminders.delete", handlers.DeleteReminder)
	e.GET("/api/reminders.info", handlers.GetReminderInfo)

	e.POST("/api/chats.add", handlers.CreateChat)
	e.POST("/api/chats.delete", handlers.DeleteChat)
	e.POST("/api/chats.rename", handlers.RenameChat)
	e.GET("/api/chats.list", handlers.GetChats)
	e.POST("/api/chats.pin", handlers.PinChat)
	e.POST("/api/chats.unpin", handlers.UnpinChat)
	e.GET("/api/chats.info", handlers.GetChatInfo)
	e.POST("/api/messages.add", handlers.CreateChatMessage)
	e.GET("/api/messages.list", handlers.GetChatMessages)

	// Initialize database (temporary)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		e.Logger.Fatal(err)
	}
	defer db.Close()

	// File System API
	e.POST("/users", func(c echo.Context) error { return createUser(c, db) })
	e.POST("/files", func(c echo.Context) error { return createFile(c, db) })
	e.GET("/files/*", func(c echo.Context) error { return readFile(c, db) })
	e.PUT("/files/*", func(c echo.Context) error { return updateFile(c, db) })
	e.DELETE("/files/*", func(c echo.Context) error { return deleteFile(c, db) })
	e.GET("/list/*", func(c echo.Context) error { return listDirectory(c, db) })

	// Start WebSocket handler
	log.Info().Msg("Starting WebSocket handler")
	wsHandler := handlers.NewWebSocketHandler()
	wsHandler.StartBroadcasting()
	e.GET("/ws", wsHandler.HandleWebSocket)

	// Start reminders agent
	go handlers.StartRemindersAgent(wsHandler)

	e.Logger.Fatal(e.Start(":1323"))
}

// FS API prototype

// Create a new user
func createUser(c echo.Context, db *sql.DB) error {
	name := c.FormValue("name")
	email := c.FormValue("email")

	id := c.FormValue("id")
	if id == "" {
		id = uuid.New().String()
	}

	_, err := db.Exec("INSERT INTO users (id, name, email) VALUES (?, ?, ?)", id, name, email)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create user")
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}

	return c.JSON(http.StatusCreated, map[string]string{"id": id, "name": name, "email": email})
}

// Create a new file
func createFile(c echo.Context, db *sql.DB) error {
	file, err := c.FormFile("file")
	if err != nil {
		log.Error().Err(err).Msg("Failed to bind file")
		return c.String(http.StatusBadRequest, "Bad Request")
	}

	src, err := file.Open()
	if err != nil {
		log.Error().Err(err).Msg("Failed to open file")
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}
	defer src.Close()

	userID := c.FormValue("user_id")
	filePath := filepath.Join(c.FormValue("path"), file.Filename)
	fileSize := file.Size

	// Ensure the path has a leading slash
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}

	tx, err := db.Begin()
	if err != nil {
		log.Error().Err(err).Msg("Failed to start transaction")
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}
	defer tx.Rollback()

	res, err := tx.Exec("INSERT INTO files (user_id, path, filename, size) VALUES (?, ?, ?, ?)", userID, filePath, file.Filename, fileSize)
	if err != nil {
		log.Error().Err(err).Msg("Failed to insert file metadata")
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}

	fileID, err := res.LastInsertId()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get last insert ID")
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}

	chunkSize := 1024 * 1024 // 1MB chunks
	chunkIndex := 0
	buffer := make([]byte, chunkSize)

	for {
		n, err := src.Read(buffer)
		if err != nil && err != io.EOF {
			log.Error().Err(err).Msg("Failed to read file content")
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}
		if n == 0 {
			break
		}

		_, err = tx.Exec("INSERT INTO file_content (file_id, chunk_index, content) VALUES (?, ?, ?)", fileID, chunkIndex, buffer[:n])
		if err != nil {
			log.Error().Err(err).Msg("Failed to insert file content")
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}
		chunkIndex++
	}

	if err := tx.Commit(); err != nil {
		log.Error().Err(err).Msg("Failed to commit transaction")
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}

	return c.String(http.StatusOK, "File created successfully")
}

// Read a file
func readFile(c echo.Context, db *sql.DB) error {
	userID := c.QueryParam("user_id")
	filePath := c.Param("*")

	// Ensure the path has a leading slash
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}

	log.Info().Msgf("Reading file for userID: %s, filePath: %s", userID, filePath)

	var fileID int64
	err := db.QueryRow("SELECT id FROM files WHERE user_id = ? AND path = ?", userID, filePath).Scan(&fileID)
	if err != nil {
		log.Error().Err(err).Msg("File not found")
		return c.String(http.StatusNotFound, "File not found")
	}
	log.Info().Msgf("File ID: %d", fileID)

	rows, err := db.Query("SELECT content FROM file_content WHERE file_id = ? ORDER BY chunk_index", fileID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get file content")
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}
	defer rows.Close()

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEOctetStream)
	for rows.Next() {
		var chunk []byte
		if err := rows.Scan(&chunk); err != nil {
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}
		if _, err := c.Response().Write(chunk); err != nil {
			log.Error().Err(err).Msg("Failed to write file content")
			return err
		}
	}

	return nil
}

// Update a file
func updateFile(c echo.Context, db *sql.DB) error {
	userID := c.FormValue("user_id")
	filePath := c.Param("*")

	// Ensure the path has a leading slash
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.String(http.StatusBadRequest, "Bad Request")
	}

	src, err := file.Open()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}
	defer src.Close()

	var fileID int64
	err = db.QueryRow("SELECT id FROM files WHERE user_id = ? AND path = ?", userID, filePath).Scan(&fileID)
	if err != nil {
		return c.String(http.StatusNotFound, "File not found")
	}

	// Delete existing content
	_, err = db.Exec("DELETE FROM file_content WHERE file_id = ?", fileID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}

	// Insert new file content in chunks
	chunkSize := 1024 * 1024 // 1MB chunks
	chunkIndex := 0
	buffer := make([]byte, chunkSize)

	for {
		n, err := src.Read(buffer)
		if err != nil && err != io.EOF {
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}
		if n == 0 {
			break
		}

		_, err = db.Exec("INSERT INTO file_content (file_id, chunk_index, content) VALUES (?, ?, ?)", fileID, chunkIndex, buffer[:n])
		if err != nil {
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}
		chunkIndex++
	}

	// Update file metadata
	_, err = db.Exec("UPDATE files SET size = ?, created_at = ? WHERE id = ?", file.Size, time.Now(), fileID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}

	return c.String(http.StatusOK, "File updated successfully")
}

// Delete a file
func deleteFile(c echo.Context, db *sql.DB) error {
	userID := c.QueryParam("user_id")
	filePath := c.Param("*")

	// Ensure the path has a leading slash
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}

	var fileID int64
	err := db.QueryRow("SELECT id FROM files WHERE user_id = ? AND path = ?", userID, filePath).Scan(&fileID)
	if err != nil {
		return c.String(http.StatusNotFound, "File not found")
	}

	_, err = db.Exec("DELETE FROM file_content WHERE file_id = ?", fileID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}

	_, err = db.Exec("DELETE FROM files WHERE id = ?", fileID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}

	return c.String(http.StatusOK, "File deleted successfully")
}

// List directory contents
func listDirectory(c echo.Context, db *sql.DB) error {
	userID := c.QueryParam("user_id")
	dirPath := c.Param("*")

	// Ensure the path has a leading slash
	if !strings.HasPrefix(dirPath, "/") {
		dirPath = "/" + dirPath
	}

	// Ensure the path ends with a slash to denote a directory
	if !strings.HasSuffix(dirPath, "/") {
		dirPath += "/"
	}

	rows, err := db.Query("SELECT filename FROM files WHERE user_id = ? AND path LIKE ?", userID, dirPath+"%")
	if err != nil {
		return c.String(http.StatusInternalServerError, "Internal Server Error")
	}
	defer rows.Close()

	var files []string
	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}
		files = append(files, filename)
	}

	return c.JSON(http.StatusOK, files)
}
