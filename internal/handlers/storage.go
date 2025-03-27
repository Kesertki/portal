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
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid file"})
	}

	fmt.Println("Uploading file:", file.Filename)

	// Save the file to the database
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to open file"})
	}
	defer src.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, src)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to read file"})
	}

	var bucketID int
	err = a.db.QueryRow("SELECT id FROM buckets WHERE name = ?", bucket).Scan(&bucketID)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "Bucket not found"})
	}

	_, err = a.db.Exec("INSERT INTO objects (bucket_id, key, data) VALUES (?, ?, ?)", bucketID, key, buf.Bytes())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to save file"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "File uploaded successfully"})
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

func (a *API) ListObjects(c echo.Context) error {
	bucketName := c.Param("bucket")
	delimiter := c.QueryParam("delimiter")

	// Get bucket ID from name
	var bucketID int
	err := a.db.QueryRow("SELECT id FROM buckets WHERE name = ?", bucketName).Scan(&bucketID)
	if err != nil {
		return c.XML(http.StatusNotFound, `<Error><Code>NoSuchBucket</Code><Message>The specified bucket does not exist</Message></Error>`)
	}

	// Query objects for the given bucket
	query := "SELECT key, created_at, LENGTH(data) as size FROM objects WHERE bucket_id = ?"
	args := []interface{}{bucketID}

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
				prefix := key[:strings.Index(key, delimiter)+len(delimiter)]
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
	uploadID := c.QueryParam("uploadId")

	if uploadID != "" {
		// Abort multipart upload
		_, _ = a.db.Exec("DELETE FROM multipart_parts WHERE upload_id = ?", uploadID)
		_, _ = a.db.Exec("DELETE FROM multipart_uploads WHERE upload_id = ?", uploadID)
		return c.XML(http.StatusOK, `<AbortMultipartUploadResult/>`)
	}

	// Delete completed object
	_, err := a.db.Exec(`
		DELETE FROM objects
		WHERE key = ? AND bucket_id = (SELECT id FROM buckets WHERE name = ?)`, key, bucket)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to delete object"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "Object deleted"})
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
