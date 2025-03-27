package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
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

	// Initialize database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open database")
	}
	defer db.Close()

	e := echo.New()

	// CORS configuration to allow all origins
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"}, // Allow all origins
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	// Middleware
	e.Pre(middleware.RemoveTrailingSlash())
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

	apiGroup := e.Group("/api")
	// apiGroup.GET("/version", handlers.GetVersion)

	e.POST("/api/users.add", func(c echo.Context) error { return createUser(c, db) })

	handlers.SetupReminderApiHandlers(apiGroup, db)
	handlers.SetupChatApiHandlers(apiGroup, db)
	handlers.SetupFileSystemApiHandlers(apiGroup, db)

	// Storage API
	api := handlers.NewAPI(db)
	storageApi := apiGroup.Group("/storage")
	// storageApi := e

	storageApi.GET("", api.ListBuckets)         // s3cmd compatibility
	storageApi.GET("/:bucket", api.ListObjects) // s3cmd compatibility
	storageApi.GET("/buckets", api.ListBuckets)
	storageApi.POST("/buckets/:bucket", api.CreateBucket)
	storageApi.POST("/buckets/:bucket/objects/:key", api.UploadObject)
	storageApi.PUT("/buckets/:bucket/objects/:key", api.UploadPart)
	storageApi.GET("/buckets/:bucket/objects/:key", api.GetObject)
	storageApi.HEAD("/buckets/:bucket/objects/:key", api.GetObject)
	storageApi.GET("/buckets/:bucket/objects", api.ListObjects)
	storageApi.DELETE("/buckets/:bucket/objects/:key", api.DeleteObject)
	storageApi.POST("/buckets/:bucket/objects/:key/complete", api.CompleteMultipartUpload)

	// Catch-all route for S3 compatibility
	storageApi.Match([]string{http.MethodGet, http.MethodHead}, "/:bucket/:key", api.GetObject)
	storageApi.DELETE("/:bucket/:key", api.DeleteObject)

	// Start WebSocket handler
	log.Info().Msg("Starting WebSocket handler")
	wsHandler := handlers.NewWebSocketHandler()
	wsHandler.StartBroadcasting()
	e.GET("/ws", wsHandler.HandleWebSocket)

	// Start reminders agent
	go handlers.StartRemindersAgent(wsHandler)

	e.Logger.Fatal(e.Start(":1323"))
}

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
