package handler

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/johnquangdev/meeting-assistant/errors"
	"github.com/johnquangdev/meeting-assistant/internal/infrastructure/storage"
	"github.com/johnquangdev/meeting-assistant/pkg/config"
)

// StorageTest handles storage testing endpoints
type StorageTest struct {
	minioClient *storage.MinIOClient
	logger      *zap.Logger
}

// NewStorageTest creates a new storage test handler
func NewStorageTest(cfg *config.Config, logger *zap.Logger) (*StorageTest, error) {
	minioClient, err := storage.NewMinIOClient(&cfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	return &StorageTest{
		minioClient: minioClient,
		logger:      logger,
	}, nil
}

// TestUpload tests uploading a file to MinIO
// @Summary      Test MinIO upload
// @Description  Test uploading a text file to MinIO to verify connection and credentials
// @Tags         Storage Test
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "Upload successful"
// @Failure      500  {object}  map[string]interface{}  "Upload failed"
// @Router       /test/storage/upload [post]
func (h *StorageTest) TestUpload(c echo.Context) error {
	ctx := c.Request().Context()

	// Create test file content
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	content := fmt.Sprintf(`MinIO Connection Test
Timestamp: %s
Server: Meeting Assistant API
Status: Connection successful
`, timestamp)

	// Upload test file
	objectName := fmt.Sprintf("test/connection-test-%s.txt", timestamp)
	err := h.minioClient.UploadText(ctx, objectName, content)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("failed to upload test file",
				zap.String("object_name", objectName),
				zap.Error(err))
		}
		return HandleError(h.logger, c, errors.ErrInternal(err))
	}

	// Get presigned URL for verification
	url, err := h.minioClient.GetFileURL(ctx, objectName, 1*time.Hour)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("uploaded but failed to generate URL",
				zap.String("object_name", objectName),
				zap.Error(err))
		}
		url = "failed to generate URL"
	}

	if h.logger != nil {
		h.logger.Info("test file uploaded successfully",
			zap.String("object_name", objectName),
			zap.String("url", url))
	}

	return HandleSuccess(h.logger, c, map[string]interface{}{
		"message":     "Upload successful",
		"object_name": objectName,
		"url":         url,
		"timestamp":   timestamp,
	})
}

// TestBucketInfo tests bucket connection and returns info
// @Summary      Test MinIO bucket info
// @Description  Get information about the MinIO bucket and connection status
// @Tags         Storage Test
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "Bucket info"
// @Failure      500  {object}  map[string]interface{}  "Failed to get bucket info"
// @Router       /test/storage/info [get]
func (h *StorageTest) TestBucketInfo(c echo.Context) error {
	ctx := c.Request().Context()

	info, err := h.minioClient.GetBucketInfo(ctx)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("failed to get bucket info", zap.Error(err))
		}
		return HandleError(h.logger, c, errors.ErrInternal(err))
	}

	if h.logger != nil {
		h.logger.Info("bucket info retrieved", zap.Any("info", info))
	}

	return HandleSuccess(h.logger, c, info)
}

// TestListFiles lists all files in the bucket
// @Summary      List files in MinIO bucket
// @Description  List all files in the MinIO bucket with optional prefix filter
// @Tags         Storage Test
// @Produce      json
// @Param        prefix  query  string  false  "File prefix filter"
// @Success      200     {object}  map[string]interface{}  "File list"
// @Failure      500     {object}  map[string]interface{}  "Failed to list files"
// @Router       /test/storage/files [get]
func (h *StorageTest) TestListFiles(c echo.Context) error {
	ctx := c.Request().Context()
	prefix := c.QueryParam("prefix")

	files, err := h.minioClient.ListFiles(ctx, prefix)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("failed to list files", zap.Error(err))
		}
		return HandleError(h.logger, c, errors.ErrInternal(err))
	}

	if h.logger != nil {
		h.logger.Info("files listed",
			zap.String("prefix", prefix),
			zap.Int("count", len(files)))
	}

	return HandleSuccess(h.logger, c, map[string]interface{}{
		"files":  files,
		"count":  len(files),
		"prefix": prefix,
	})
}

// TestDownloadURL generates a download URL for a file
// @Summary      Generate download URL
// @Description  Generate a presigned URL for downloading a file from MinIO
// @Tags         Storage Test
// @Produce      json
// @Param        file  query  string  true  "File path/name in bucket"
// @Success      200   {object}  map[string]interface{}  "Download URL"
// @Failure      400   {object}  map[string]interface{}  "Missing file parameter"
// @Failure      500   {object}  map[string]interface{}  "Failed to generate URL"
// @Router       /test/storage/download-url [get]
func (h *StorageTest) TestDownloadURL(c echo.Context) error {
	ctx := c.Request().Context()
	filePath := c.QueryParam("file")

	if filePath == "" {
		return HandleError(h.logger, c, errors.ErrInvalidArgument("Missing file parameter"))
	}

	url, err := h.minioClient.GetFileURL(ctx, filePath, 1*time.Hour)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("failed to generate download URL",
				zap.String("file", filePath),
				zap.Error(err))
		}
		return HandleError(h.logger, c, errors.ErrInternal(err))
	}

	return HandleSuccess(h.logger, c, map[string]interface{}{
		"file":       filePath,
		"url":        url,
		"expires_in": "1 hour",
	})
}
