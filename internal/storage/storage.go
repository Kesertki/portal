package storage

import (
	"database/sql"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

func ApplyMigrations() {
	log.Println("Applying database migrations...")
	m, err := migrate.New(
		"file://db/migrations",
		"sqlite3://data/portal.db",
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal(err)
	}
	log.Println("Migrations applied successfully")
}

func ConnectToStorage() (*sql.DB, error) {
	dataPath := os.Getenv("DATA_PATH")
	if dataPath == "" {
		dataPath = "."
	}

	db, err := sql.Open("sqlite3", dataPath+"/portal.db")
	return db, err
}
