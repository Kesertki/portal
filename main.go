package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

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

	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	storage.ApplyMigrations()

	// Log environment variables
	log.Println("DATA_PATH:", os.Getenv("DATA_PATH"))

	e := echo.New()

	// CORS configuration to allow all origins
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"}, // Allow all origins
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	// Middleware
	e.Use(middleware.Logger())
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

	// Create a new WebSocket handler
	wsHandler := handlers.NewWebSocketHandler()

	// Start the broadcasting process
	wsHandler.StartBroadcasting()

	// Routes
	e.GET("/ws", wsHandler.HandleWebSocket)

	// go handlers.HandleMessages(clients, broadcast)
	go handlers.StartRemindersAgent(wsHandler)

	e.Logger.Fatal(e.Start(":1323"))
}
