package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Kesertki/portal/internal/models"
	"github.com/Kesertki/portal/internal/storage"
	"github.com/labstack/echo/v4"
)

// Returns geolocation information for the client's IP address,
// or for the IP address specified in the `X-Forwarded-For` header,
// or for the IP address specified in the `PORTAL_CLIENT_IP` environment variable.
func GetGeoLocation(c echo.Context) error {
	config := models.Config{
		UsePreconfiguredGeoLocation: os.Getenv("PORTAL_GEO_LOCATION_ENABLED") != "true",
		PreconfiguredGeoLocation: models.GeoLocation{
			IP:       "8.8.8.8",
			City:     "Mountain View",
			Region:   "California",
			Country:  "US",
			Loc:      "37.3860,-122.0840",
			Org:      "Google LLC",
			Postal:   "94035",
			Timezone: "America/Los_Angeles",
		},
	}

	if config.UsePreconfiguredGeoLocation {
		log.Println("Using preconfigured geolocation")
		return c.JSON(http.StatusOK, config.PreconfiguredGeoLocation)
	}

	envIP := os.Getenv("PORTAL_CLIENT_IP")
	ip := envIP
	if ip == "" {
		ip = c.RealIP()
		if ip == "" {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get client IP address"})
		}
	}

	db, err := storage.ConnectToStorage()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Storage connection failed"})
	}
	defer db.Close()

	cached, err := storage.GetFromCache(db, ip)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Storage read failed"})
	}
	if cached != nil {
		return c.JSON(http.StatusOK, cached)
	}

	url := fmt.Sprintf("https://ipinfo.io/%s/json", ip)

	resp, err := http.Get(url)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get geolocation"})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get geolocation"})
	}

	var geoLocation models.GeoLocation
	if err := json.NewDecoder(resp.Body).Decode(&geoLocation); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to parse geolocation data"})
	}

	if err := storage.SaveToCache(db, geoLocation); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Storage write failed"})
	}

	return c.JSON(http.StatusOK, geoLocation)
}
