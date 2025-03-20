package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/Kesertki/portal/internal/handlers"
	"github.com/Kesertki/portal/internal/storage"
)

func main() {
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

	// e.GET("/users/:id", handlers.GetUser)
	/*
		curl -X POST http://localhost:1323/users \
		     -H "Content-Type: application/json" \
		     -d '{"name": "John Doe", "email": "john.doe@example.com"}'
	*/
	// e.POST("/users", handlers.CreateUser)

	e.POST("/api/reminders", handlers.CreateReminder)
	e.GET("/api/reminders", handlers.GetReminders)

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
