package handlers

import (
	"bytes"
	"database/sql"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
)

type API struct {
	db *sql.DB
}

func NewAPI(db *sql.DB) *API {
	return &API{db: db}
}

// Create a new bucket
func (a *API) CreateBucket(c echo.Context) error {
	name := c.Param("bucket")

	_, err := a.db.Exec("INSERT INTO buckets (name) VALUES (?)", name)
	if err != nil {
		return c.JSON(http.StatusConflict, echo.Map{"error": "Bucket already exists"})
	}
	return c.JSON(http.StatusCreated, echo.Map{"message": "Bucket created"})
}

// Upload an object
func (a *API) UploadObject(c echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("key")

	// Get bucket ID
	var bucketID int
	err := a.db.QueryRow("SELECT id FROM buckets WHERE name = ?", bucket).Scan(&bucketID)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "Bucket not found"})
	}

	// Read file
	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid file"})
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, src)
	if err != nil {
		return err
	}

	_, err = a.db.Exec("INSERT INTO objects (bucket_id, key, data, content_type) VALUES (?, ?, ?, ?)",
		bucketID, key, buf.Bytes(), file.Header.Get("Content-Type"))
	if err != nil {
		return c.JSON(http.StatusConflict, echo.Map{"error": "Object already exists"})
	}

	return c.JSON(http.StatusCreated, echo.Map{"message": "Object uploaded"})
}

// Get an object
func (a *API) GetObject(c echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("key")

	var data []byte
	var contentType string

	err := a.db.QueryRow(`
		SELECT o.data, o.content_type
		FROM objects o
		JOIN buckets b ON o.bucket_id = b.id
		WHERE b.name = ? AND o.key = ?`, bucket, key).Scan(&data, &contentType)

	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "Object not found"})
	}

	return c.Blob(http.StatusOK, contentType, data)
}

// List objects in a bucket
func (a *API) ListObjects(c echo.Context) error {
	bucket := c.Param("bucket")

	rows, err := a.db.Query(`
		SELECT o.key FROM objects o
		JOIN buckets b ON o.bucket_id = b.id
		WHERE b.name = ?`, bucket)
	if err != nil {
		return err
	}
	defer rows.Close()

	var objects []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return err
		}
		objects = append(objects, key)
	}

	return c.JSON(http.StatusOK, echo.Map{"objects": objects})
}

// Delete an object
func (a *API) DeleteObject(c echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("key")

	_, err := a.db.Exec(`
		DELETE FROM objects
		WHERE key = ? AND bucket_id = (SELECT id FROM buckets WHERE name = ?)`, key, bucket)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to delete object"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "Object deleted"})
}
