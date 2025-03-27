package handlers

import (
	"bytes"
	"crypto/md5"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type API struct {
	db *sql.DB
}

func NewAPI(db *sql.DB) *API {
	return &API{db: db}
}

func (a *API) ListBuckets(c echo.Context) error {
	rows, err := a.db.Query("SELECT name, created_at FROM buckets")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to retrieve buckets"})
	}
	defer rows.Close()

	type Bucket struct {
		Name         string `xml:"Name"`
		CreationDate string `xml:"CreationDate"`
	}

	var buckets []Bucket
	for rows.Next() {
		var name string
		var creationDate string
		if err := rows.Scan(&name, &creationDate); err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to scan bucket data"})
		}
		buckets = append(buckets, Bucket{Name: name, CreationDate: creationDate})
	}

	type ListAllMyBucketsResult struct {
		XMLName xml.Name `xml:"ListAllMyBucketsResult"`
		Buckets struct {
			Bucket []Bucket `xml:"Bucket"`
		} `xml:"Buckets"`
	}

	response := ListAllMyBucketsResult{}
	response.Buckets.Bucket = buckets

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationXMLCharsetUTF8)
	return c.XML(http.StatusOK, response)
}

func (a *API) CreateBucket(c echo.Context) error {
	bucketName := c.Param("bucket")

	// Insert the new bucket into the database
	_, err := a.db.Exec("INSERT INTO buckets (name) VALUES (?)", bucketName)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			return c.XML(http.StatusConflict, `<Error><Code>BucketAlreadyExists</Code><Message>The requested bucket name is not available</Message></Error>`)
		}
		return c.XML(http.StatusInternalServerError, `<Error><Code>InternalError</Code><Message>Failed to create bucket</Message></Error>`)
	}

	response := `<CreateBucketResponse><Message>Bucket created successfully</Message></CreateBucketResponse>`
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationXMLCharsetUTF8)
	return c.XML(http.StatusCreated, response)
}

func (a *API) DeleteBucket(c echo.Context) error {
	bucketName := c.Param("bucket")

	// Start a transaction to ensure atomicity
	tx, err := a.db.Begin()
	if err != nil {
		return c.XML(http.StatusInternalServerError, `<Error><Code>InternalError</Code><Message>Failed to start transaction</Message></Error>`)
	}

	// Delete all objects in the bucket
	_, err = tx.Exec("DELETE FROM objects WHERE bucket_id = (SELECT id FROM buckets WHERE name = ?)", bucketName)
	if err != nil {
		tx.Rollback()
		return c.XML(http.StatusInternalServerError, `<Error><Code>InternalError</Code><Message>Failed to delete objects</Message></Error>`)
	}

	// Delete the bucket itself
	_, err = tx.Exec("DELETE FROM buckets WHERE name = ?", bucketName)
	if err != nil {
		tx.Rollback()
		return c.XML(http.StatusInternalServerError, `<Error><Code>InternalError</Code><Message>Failed to delete bucket</Message></Error>`)
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return c.XML(http.StatusInternalServerError, `<Error><Code>InternalError</Code><Message>Failed to commit transaction</Message></Error>`)
	}

	return c.NoContent(http.StatusNoContent)
}

func (a *API) UploadObject(c echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("key")
	fmt.Println("Received request for:", bucket, key)
	fmt.Println("Query Params:", c.QueryParams())

	// Check for the presence of the 'uploads' query parameter
	if c.QueryParams().Has("uploads") {
		fmt.Println("Initiating multipart upload...")
		return a.InitiateMultipartUpload(c)
	}

	// Normal object upload logic
	file, err := c.FormFile("file")
	if err != nil {
		fmt.Println("Error: Invalid file upload")
		return c.XML(http.StatusBadRequest, `<Error><Code>InvalidRequest</Code><Message>Invalid file</Message></Error>`)
	}

	fmt.Println("Uploading file:", file.Filename)

	// Save the file to the database
	src, err := file.Open()
	if err != nil {
		return c.XML(http.StatusInternalServerError, `<Error><Code>InternalError</Code><Message>Failed to open file</Message></Error>`)
	}
	defer src.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, src)
	if err != nil {
		return c.XML(http.StatusInternalServerError, `<Error><Code>InternalError</Code><Message>Failed to read file</Message></Error>`)
	}

	var bucketID int
	err = a.db.QueryRow("SELECT id FROM buckets WHERE name = ?", bucket).Scan(&bucketID)
	if err != nil {
		return c.XML(http.StatusNotFound, `<Error><Code>NoSuchBucket</Code><Message>Bucket not found</Message></Error>`)
	}

	// Get the Content-Type from the multipart form data
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream" // Default content type if not provided
	}

	_, err = a.db.Exec("INSERT INTO objects (bucket_id, key, data, content_type, created_at) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)", bucketID, key, buf.Bytes(), contentType)
	if err != nil {
		return c.XML(http.StatusInternalServerError, `<Error><Code>InternalError</Code><Message>Failed to save file</Message></Error>`)
	}

	// Calculate ETag
	etag := fmt.Sprintf("%x", md5.Sum(buf.Bytes()))

	// Return XML response for successful upload
	response := struct {
		XMLName xml.Name `xml:"PutObjectResult"`
		ETag    string   `xml:"ETag"`
	}{
		ETag: etag,
	}

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationXMLCharsetUTF8)
	return c.XML(http.StatusOK, response)
}

func (a *API) GetObject(c echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("key")

	var data []byte
	var contentType sql.NullString
	var lastModified time.Time

	err := a.db.QueryRow(`
		SELECT o.data, o.content_type, o.created_at
		FROM objects o
		JOIN buckets b ON o.bucket_id = b.id
		WHERE b.name = ? AND o.key = ?`, bucket, key).Scan(&data, &contentType, &lastModified)

	if err != nil {
		fmt.Println("Error retrieving object:", err)
		return c.XML(http.StatusNotFound, `<Error><Code>NoSuchKey</Code><Message>The specified key does not exist</Message></Error>`)
	}

	finalContentType := "application/octet-stream"
	if contentType.Valid {
		finalContentType = contentType.String
	}

	fmt.Println("Retrieved object data:", string(data))
	fmt.Println("Content-Type:", finalContentType)

	// Calculate ETag
	etag := fmt.Sprintf("%x", md5.Sum(data))

	// Set the Content-Length, ETag, and Last-Modified headers
	contentLength := len(data)
	c.Response().Header().Set(echo.HeaderContentLength, fmt.Sprintf("%d", contentLength))
	c.Response().Header().Set("ETag", etag)
	c.Response().Header().Set("Last-Modified", lastModified.Format(http.TimeFormat))

	if c.Request().Method == http.MethodHead {
		// For HEAD requests, return headers without the body
		return c.NoContent(http.StatusOK)
	}

	return c.Blob(http.StatusOK, finalContentType, data)
}

func (a *API) ListObjects(c echo.Context) error {
	bucketName := c.Param("bucket")
	delimiter := c.QueryParam("delimiter")
	location := c.QueryParam("location")

	if location != "" {
		// Return a default location for the bucket
		response := struct {
			XMLName xml.Name `xml:"LocationConstraint"`
			Value   string   `xml:",chardata"`
		}{
			Value: "us-east-1", // Default region
		}
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationXMLCharsetUTF8)
		return c.XML(http.StatusOK, response)
	}

	// Get bucket ID from name
	var bucketID int
	err := a.db.QueryRow("SELECT id FROM buckets WHERE name = ?", bucketName).Scan(&bucketID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.XML(http.StatusNotFound, `<Error><Code>NoSuchBucket</Code><Message>The specified bucket does not exist</Message></Error>`)
		}
		return c.XML(http.StatusInternalServerError, `<Error><Code>InternalError</Code><Message>Failed to retrieve bucket information</Message></Error>`)
	}

	// Query objects for the given bucket
	query := "SELECT key, created_at, LENGTH(data) as size FROM objects WHERE bucket_id = ?"
	args := []interface{}{bucketID}

	if delimiter != "" {
		query += " AND key LIKE ?"
		args = append(args, "%"+delimiter+"%")
	}

	rows, err := a.db.Query(query, args...)
	if err != nil {
		return c.XML(http.StatusInternalServerError, `<Error><Code>InternalError</Code><Message>Failed to retrieve objects</Message></Error>`)
	}
	defer rows.Close()

	type Object struct {
		Key          string `xml:"Key"`
		LastModified string `xml:"LastModified"`
		ETag         string `xml:"ETag"`
		Size         int64  `xml:"Size"`
		StorageClass string `xml:"StorageClass"`
	}

	var objects []Object
	var commonPrefixes []string
	for rows.Next() {
		var key string
		var createdAt string
		var size int64
		if err := rows.Scan(&key, &createdAt, &size); err != nil {
			return c.XML(http.StatusInternalServerError, `<Error><Code>InternalError</Code><Message>Failed to scan object data</Message></Error>`)
		}

		if delimiter != "" {
			// Check if the key contains the delimiter and add the prefix
			if strings.Contains(key, delimiter) {
				prefix := key[:strings.Index(key, delimiter)+1]
				if !contains(commonPrefixes, prefix) {
					commonPrefixes = append(commonPrefixes, prefix)
				}
				// Skip adding the object itself if it matches the delimiter
				continue
			}
		}

		// For simplicity, using a dummy ETag and StorageClass
		etag := "dummy-etag" // Replace with actual ETag calculation if available
		storageClass := "STANDARD"
		objects = append(objects, Object{Key: key, LastModified: createdAt, ETag: etag, Size: size, StorageClass: storageClass})
	}

	type ListBucketResult struct {
		XMLName        xml.Name `xml:"ListBucketResult"`
		Name           string   `xml:"Name"`
		Prefix         string   `xml:"Prefix"`
		Marker         string   `xml:"Marker"`
		MaxKeys        int      `xml:"MaxKeys"`
		IsTruncated    bool     `xml:"IsTruncated"`
		Contents       []Object `xml:"Contents"`
		CommonPrefixes []string `xml:"CommonPrefixes>Prefix"`
	}

	response := ListBucketResult{
		Name:           bucketName,
		Prefix:         "",
		Marker:         "",
		MaxKeys:        1000,
		IsTruncated:    false,
		Contents:       objects,
		CommonPrefixes: commonPrefixes,
	}

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationXMLCharsetUTF8)
	return c.XML(http.StatusOK, response)
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (a *API) DeleteObject(c echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("key")

	var bucketID int
	err := a.db.QueryRow("SELECT id FROM buckets WHERE name = ?", bucket).Scan(&bucketID)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "Bucket not found"})
	}

	_, err = a.db.Exec("DELETE FROM objects WHERE bucket_id = ? AND key = ?", bucketID, key)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to delete object"})
	}

	// Return a 204 No Content response to indicate successful deletion
	return c.NoContent(http.StatusNoContent)
}

func (a *API) InitiateMultipartUpload(c echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("key")
	uploadID := uuid.New().String() // Generate a unique Upload ID

	fmt.Println("Multipart Upload Initiated for:", bucket, key, "UploadID:", uploadID)

	// Store the upload ID in the database
	var bucketID int
	err := a.db.QueryRow("SELECT id FROM buckets WHERE name = ?", bucket).Scan(&bucketID)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "Bucket not found"})
	}

	_, err = a.db.Exec("INSERT INTO multipart_uploads (bucket_id, key, upload_id) VALUES (?, ?, ?)", bucketID, key, uploadID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to initiate multipart upload"})
	}

	return c.XML(http.StatusOK, struct {
		XMLName  xml.Name `xml:"InitiateMultipartUploadResult"`
		Bucket   string   `xml:"Bucket"`
		Key      string   `xml:"Key"`
		UploadID string   `xml:"UploadId"`
	}{
		Bucket:   bucket,
		Key:      key,
		UploadID: uploadID,
	})
}

func (a *API) UploadPart(c echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("key")
	uploadID := c.QueryParam("uploadId")
	partNumber := c.QueryParam("partNumber")

	// Log the received parameters
	fmt.Println("UploadPart - Bucket:", bucket)
	fmt.Println("UploadPart - Key:", key)
	fmt.Println("UploadPart - UploadID:", uploadID)
	fmt.Println("UploadPart - PartNumber:", partNumber)

	// Validate bucket and key exist
	var bucketID int
	err := a.db.QueryRow("SELECT id FROM buckets WHERE name = ?", bucket).Scan(&bucketID)
	if err != nil {
		return c.XML(http.StatusNotFound, `<Error><Code>NoSuchBucket</Code></Error>`)
	}

	var exists bool
	err = a.db.QueryRow("SELECT EXISTS (SELECT 1 FROM multipart_uploads WHERE upload_id = ? AND key = ? AND bucket_id = ?)",
		uploadID, key, bucketID).Scan(&exists)
	if err != nil || !exists {
		return c.XML(http.StatusNotFound, `<Error><Code>NoSuchUpload</Code></Error>`)
	}

	// Read part data
	var buf bytes.Buffer
	_, err = io.Copy(&buf, c.Request().Body)
	if err != nil {
		return c.XML(http.StatusInternalServerError, `<Error><Code>InternalError</Code></Error>`)
	}

	etag := fmt.Sprintf("%x", md5.Sum(buf.Bytes()))
	fmt.Println("Calculated ETag:", etag)

	// Store the part data
	_, err = a.db.Exec("INSERT INTO multipart_parts (upload_id, part_number, data) VALUES (?, ?, ?)",
		uploadID, partNumber, buf.Bytes())
	if err != nil {
		return c.XML(http.StatusConflict, `<Error><Code>PartAlreadyExists</Code></Error>`)
	}

	// Set the ETag in the response header
	c.Response().Header().Set("ETag", etag)

	// Construct the XML response
	response := struct {
		XMLName xml.Name `xml:"UploadPartResult"`
		ETag    string   `xml:"ETag"`
	}{
		ETag: etag,
	}
	fmt.Println("Response:", response)

	return c.XML(http.StatusOK, response)
}

func (a *API) CompleteMultipartUpload(c echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("key")
	uploadID := c.QueryParam("uploadId")

	// Validate upload ID
	var bucketID int
	err := a.db.QueryRow("SELECT bucket_id FROM multipart_uploads WHERE upload_id = ?", uploadID).Scan(&bucketID)
	if err != nil {
		return c.XML(http.StatusNotFound, `<Error><Code>NoSuchUpload</Code></Error>`)
	}

	// Retrieve all parts
	rows, err := a.db.Query("SELECT data FROM multipart_parts WHERE upload_id = ? ORDER BY part_number ASC", uploadID)
	if err != nil {
		return c.XML(http.StatusInternalServerError, `<Error><Code>InternalError</Code></Error>`)
	}
	defer rows.Close()

	var finalData bytes.Buffer
	for rows.Next() {
		var partData []byte
		if err := rows.Scan(&partData); err != nil {
			return err
		}
		finalData.Write(partData)
	}

	// Store the final object
	_, err = a.db.Exec("INSERT INTO objects (bucket_id, key, data) VALUES (?, ?, ?)", bucketID, key, finalData.Bytes())
	if err != nil {
		return c.XML(http.StatusInternalServerError, `<Error><Code>InternalError</Code></Error>`)
	}

	// Cleanup
	_, _ = a.db.Exec("DELETE FROM multipart_parts WHERE upload_id = ?", uploadID)
	_, _ = a.db.Exec("DELETE FROM multipart_uploads WHERE upload_id = ?", uploadID)

	etag := fmt.Sprintf("%x", md5.Sum(finalData.Bytes()))

	// Construct the XML response
	xmlResponse := struct {
		XMLName  xml.Name `xml:"CompleteMultipartUploadResult"`
		Location string   `xml:"Location"`
		Bucket   string   `xml:"Bucket"`
		Key      string   `xml:"Key"`
		ETag     string   `xml:"ETag"`
	}{
		Location: fmt.Sprintf("http://localhost:1323/buckets/%s/objects/%s", bucket, key),
		Bucket:   bucket,
		Key:      key,
		ETag:     etag,
	}

	// Set the Content-Type header to application/xml
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationXML)

	return c.XML(http.StatusOK, xmlResponse)
}

func (a *API) AbortMultipartUpload(c echo.Context) error {
	bucket := c.Param("bucket")
	key := c.Param("key")
	uploadID := c.QueryParam("uploadId")

	// Start a transaction to ensure atomicity
	tx, err := a.db.Begin()
	if err != nil {
		return c.XML(http.StatusInternalServerError, `<Error><Code>InternalError</Code><Message>Failed to start transaction</Message></Error>`)
	}

	// Delete all parts associated with the upload ID
	_, err = tx.Exec("DELETE FROM multipart_parts WHERE upload_id = ?", uploadID)
	if err != nil {
		tx.Rollback()
		return c.XML(http.StatusInternalServerError, `<Error><Code>InternalError</Code><Message>Failed to delete parts</Message></Error>`)
	}

	// Delete the multipart upload record
	_, err = tx.Exec("DELETE FROM multipart_uploads WHERE upload_id = ?", uploadID)
	if err != nil {
		tx.Rollback()
		return c.XML(http.StatusInternalServerError, `<Error><Code>InternalError</Code><Message>Failed to delete upload record</Message></Error>`)
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return c.XML(http.StatusInternalServerError, `<Error><Code>InternalError</Code><Message>Failed to commit transaction</Message></Error>`)
	}

	// Return a successful response
	return c.NoContent(http.StatusNoContent)
}
