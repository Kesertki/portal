package handlers

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type DateResponse struct {
	Date string `json:"date"`
}

// Returns the current date and time in RFC3339 format
func GetCurrentDate(c echo.Context) error {
	currentTime := time.Now()
	response := DateResponse{
		Date: currentTime.Format(time.RFC3339),
	}
	return c.JSON(http.StatusOK, response)
}
