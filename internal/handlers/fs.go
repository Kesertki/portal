package handlers

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// CreateFileHandler handles file creation
func CreateFileHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		file, err := c.FormFile("file")
		if err != nil {
			log.Error().Err(err).Msg("Failed to bind file")
			return c.String(http.StatusBadRequest, "Bad Request")
		}

		src, err := file.Open()
		if err != nil {
			log.Error().Err(err).Msg("Failed to open file")
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}
		defer func() {
			if cerr := src.Close(); cerr != nil {
				fmt.Printf("Error closing file: %v\n", cerr)
			}
		}()

		userID := c.FormValue("user_id")
		filePath := filepath.Join(c.FormValue("path"), file.Filename)
		fileSize := file.Size

		// Ensure the path has a leading slash
		if !strings.HasPrefix(filePath, "/") {
			filePath = "/" + filePath
		}

		tx, err := db.Begin()
		if err != nil {
			log.Error().Err(err).Msg("Failed to start transaction")
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}
		rollback(tx)

		res, err := tx.Exec("INSERT INTO files (user_id, path, filename, size) VALUES (?, ?, ?, ?)", userID, filePath, file.Filename, fileSize)
		if err != nil {
			log.Error().Err(err).Msg("Failed to insert file metadata")
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}

		fileID, err := res.LastInsertId()
		if err != nil {
			log.Error().Err(err).Msg("Failed to get last insert ID")
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}

		chunkSize := 1024 * 1024 // 1MB chunks
		chunkIndex := 0
		buffer := make([]byte, chunkSize)

		for {
			n, err := src.Read(buffer)
			if err != nil && err != io.EOF {
				log.Error().Err(err).Msg("Failed to read file content")
				return c.String(http.StatusInternalServerError, "Internal Server Error")
			}
			if n == 0 {
				break
			}

			_, err = tx.Exec("INSERT INTO file_content (file_id, chunk_index, content) VALUES (?, ?, ?)", fileID, chunkIndex, buffer[:n])
			if err != nil {
				log.Error().Err(err).Msg("Failed to insert file content")
				return c.String(http.StatusInternalServerError, "Internal Server Error")
			}
			chunkIndex++
		}

		if err := tx.Commit(); err != nil {
			log.Error().Err(err).Msg("Failed to commit transaction")
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}

		return c.String(http.StatusOK, "File created successfully")
	}
}

// ReadFileHandler handles file reading
func ReadFileHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID := c.QueryParam("user_id")
		filePath := c.Param("*")

		// Ensure the path has a leading slash
		if !strings.HasPrefix(filePath, "/") {
			filePath = "/" + filePath
		}

		log.Info().Msgf("Reading file for userID: %s, filePath: %s", userID, filePath)

		var fileID int64
		err := db.QueryRow("SELECT id FROM files WHERE user_id = ? AND path = ?", userID, filePath).Scan(&fileID)
		if err != nil {
			log.Error().Err(err).Msg("File not found")
			return c.String(http.StatusNotFound, "File not found")
		}
		log.Info().Msgf("File ID: %d", fileID)

		rows, err := db.Query("SELECT content FROM file_content WHERE file_id = ? ORDER BY chunk_index", fileID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get file content")
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}
		defer func() {
			if err := rows.Close(); err != nil {
				log.Error().Err(err).Msg("Error closing rows")
			}
		}()

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEOctetStream)
		for rows.Next() {
			var chunk []byte
			if err := rows.Scan(&chunk); err != nil {
				log.Error().Err(err).Msg("Failed to scan file content")
				return c.String(http.StatusInternalServerError, "Internal Server Error")
			}
			if _, err := c.Response().Write(chunk); err != nil {
				log.Error().Err(err).Msg("Failed to write file content")
				return err
			}
		}

		return nil
	}
}

// UpdateFileHandler handles file updating
func UpdateFileHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID := c.FormValue("user_id")
		filePath := c.Param("*")

		// Ensure the path has a leading slash
		if !strings.HasPrefix(filePath, "/") {
			filePath = "/" + filePath
		}

		file, err := c.FormFile("file")
		if err != nil {
			log.Error().Err(err).Msg("Failed to bind file")
			return c.String(http.StatusBadRequest, "Bad Request")
		}

		src, err := file.Open()
		if err != nil {
			log.Error().Err(err).Msg("Failed to open file")
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}
		defer func() {
			if cerr := src.Close(); cerr != nil {
				fmt.Printf("Error closing file: %v\n", cerr)
			}
		}()

		var fileID int64
		err = db.QueryRow("SELECT id FROM files WHERE user_id = ? AND path = ?", userID, filePath).Scan(&fileID)
		if err != nil {
			log.Error().Err(err).Msg("File not found")
			return c.String(http.StatusNotFound, "File not found")
		}

		// Delete existing content
		_, err = db.Exec("DELETE FROM file_content WHERE file_id = ?", fileID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to delete file content")
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}

		// Insert new file content in chunks
		chunkSize := 1024 * 1024 // 1MB chunks
		chunkIndex := 0
		buffer := make([]byte, chunkSize)

		for {
			n, err := src.Read(buffer)
			if err != nil && err != io.EOF {
				log.Error().Err(err).Msg("Failed to read file content")
				return c.String(http.StatusInternalServerError, "Internal Server Error")
			}
			if n == 0 {
				break
			}

			_, err = db.Exec("INSERT INTO file_content (file_id, chunk_index, content) VALUES (?, ?, ?)", fileID, chunkIndex, buffer[:n])
			if err != nil {
				log.Error().Err(err).Msg("Failed to insert file content")
				return c.String(http.StatusInternalServerError, "Internal Server Error")
			}
			chunkIndex++
		}

		// Update file metadata
		_, err = db.Exec("UPDATE files SET size = ?, created_at = ? WHERE id = ?", file.Size, time.Now(), fileID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to update file metadata")
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}

		return c.String(http.StatusOK, "File updated successfully")
	}
}

// DeleteFileHandler handles file deletion
func DeleteFileHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID := c.QueryParam("user_id")
		filePath := c.Param("*")

		// Ensure the path has a leading slash
		if !strings.HasPrefix(filePath, "/") {
			filePath = "/" + filePath
		}

		var fileID int64
		err := db.QueryRow("SELECT id FROM files WHERE user_id = ? AND path = ?", userID, filePath).Scan(&fileID)
		if err != nil {
			log.Error().Err(err).Msg("File not found")
			return c.String(http.StatusNotFound, "File not found")
		}

		_, err = db.Exec("DELETE FROM file_content WHERE file_id = ?", fileID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to delete file content")
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}

		_, err = db.Exec("DELETE FROM files WHERE id = ?", fileID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to delete file metadata")
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}

		return c.String(http.StatusOK, "File deleted successfully")
	}
}

// ListDirectoryHandler handles directory listing
func ListDirectoryHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID := c.QueryParam("user_id")
		dirPath := c.Param("*")

		// Ensure the path has a leading slash
		if !strings.HasPrefix(dirPath, "/") {
			dirPath = "/" + dirPath
		}

		// Ensure the path ends with a slash to denote a directory
		if !strings.HasSuffix(dirPath, "/") {
			dirPath += "/"
		}

		rows, err := db.Query("SELECT filename FROM files WHERE user_id = ? AND path LIKE ?", userID, dirPath+"%")
		if err != nil {
			log.Error().Err(err).Msg("Failed to list directory")
			return c.String(http.StatusInternalServerError, "Internal Server Error")
		}
		defer func() {
			if err := rows.Close(); err != nil {
				log.Error().Err(err).Msg("Error closing rows")
			}
		}()

		var files []string
		for rows.Next() {
			var filename string
			if err := rows.Scan(&filename); err != nil {
				return c.String(http.StatusInternalServerError, "Internal Server Error")
			}
			files = append(files, filename)
		}

		return c.JSON(http.StatusOK, files)
	}
}

// SetupFileSystemApiHandlers sets up the file system API handlers
func SetupFileSystemApiHandlers(apiGroup *echo.Group, db *sql.DB) {
	log.Info().Msg("Initializing File System API")

	fsGroup := apiGroup.Group("/fs")
	fsGroup.POST("/files", CreateFileHandler(db))
	fsGroup.GET("/files/*", ReadFileHandler(db))
	fsGroup.PUT("/files/*", UpdateFileHandler(db))
	fsGroup.DELETE("/files/*", DeleteFileHandler(db))
	fsGroup.GET("/list/*", ListDirectoryHandler(db))
}
