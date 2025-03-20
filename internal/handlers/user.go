package handlers

import (
	"net/http"

	"github.com/Kesertki/portal/internal/models"
	"github.com/labstack/echo/v4"
)

// e.GET("/users/:id", getUser)
func GetUser(c echo.Context) error {
	id := c.Param("id")
	return c.String(http.StatusOK, id)
}

func CreateUser(c echo.Context) error {
	u := new(models.User)
	if err := c.Bind(u); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, u)
	// or
	// return c.XML(http.StatusCreated, u)
}
