package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"

	"github.com/Kesertki/portal/internal/handlers"
	"github.com/Kesertki/portal/internal/storage"
)

var (
	Version   = "dev" // Default value
	BuildDate = "unknown"
	GitCommit = "unknown"
)

func main() {
	fmt.Printf("%s v%s (Commit: %s, Built: %s)\n", "[portal]", Version, GitCommit, BuildDate)

	// Configure zerolog
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

	log.Info().Msg("Starting File System API")
	// Create a new Afero file system (you can replace this with any other Afero-supported file system)
	// fs := afero.NewOsFs()
	fs := afero.NewMemMapFs()

	// Create test files and directories
	err = fs.MkdirAll("src/a", 0755)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create directory")
	}
	err = afero.WriteFile(fs, "src/a/b", []byte("file b"), 0644)
	if err != nil {
		log.Error().Err(err).Msg("Failed to write file")
	}
	err = afero.WriteFile(fs, "src/c", []byte("file c"), 0644)
	if err != nil {
		log.Error().Err(err).Msg("Failed to write file")
	}

	afero.Walk(fs, "", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			e.Logger.Error(err)
			return nil
		}
		log.Info().Msgf("File: %s", path)
		return nil
	})

	httpFs := afero.NewHttpFs(fs)
	fileServer := http.FileServer(httpFs.Dir(""))
	e.GET("/api/fs/*", echo.WrapHandler(http.StripPrefix("/api/fs/", fileServer)))

	// Start WebSocket handler
	log.Info().Msg("Starting WebSocket handler")
	wsHandler := handlers.NewWebSocketHandler()
	wsHandler.StartBroadcasting()
	e.GET("/ws", wsHandler.HandleWebSocket)

	// Start reminders agent
	go handlers.StartRemindersAgent(wsHandler)

	e.Logger.Fatal(e.Start(":1323"))
}
