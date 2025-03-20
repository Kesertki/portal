package storage

import (
	"database/sql"

	"github.com/Kesertki/portal/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

func SaveToCache(db *sql.DB, gl models.GeoLocation) error {
	_, err := db.Exec(`
        INSERT OR REPLACE INTO cache_geolocation
        (ip, city, region, country, loc, org, postal, timezone)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		gl.IP, gl.City, gl.Region, gl.Country, gl.Loc, gl.Org, gl.Postal, gl.Timezone)
	return err
}

func GetFromCache(db *sql.DB, ip string) (*models.GeoLocation, error) {
	row := db.QueryRow(`SELECT ip, city, region, country, loc, org, postal, timezone FROM cache_geolocation WHERE ip = ?`, ip)
	var gl models.GeoLocation
	err := row.Scan(&gl.IP, &gl.City, &gl.Region, &gl.Country, &gl.Loc, &gl.Org, &gl.Postal, &gl.Timezone)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &gl, nil
}
