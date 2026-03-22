package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// OTAHandler handles OTA (Over-The-Air) update endpoints
type OTAHandler struct {
	APKDir string
}

// NewOTAHandler creates a new OTAHandler
func NewOTAHandler(apkDir string) *OTAHandler {
	return &OTAHandler{
		APKDir: apkDir,
	}
}

// extractVersion extracts version from OTA filename
// e.g., "ota_12345_com.example.app_v1.2.3.apk" -> "1.2.3"
func extractVersion(filename string) string {
	// Find the _v marker and extract version before .apk
	if idx := strings.Index(filename, "_v"); idx >= 0 {
		versionPart := filename[idx+2:]
		if endIdx := strings.LastIndex(versionPart, ".apk"); endIdx >= 0 {
			return versionPart[:endIdx]
		}
	}
	return ""
}

// UploadOTA handles APK file upload
// POST /api/v2/fieldtrip/ota/upload
func (h *OTAHandler) UploadOTA(c echo.Context) error {
	// Get the APK file from the multipart form
	file, err := c.FormFile("apk")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "apk file is required")
	}

	// Validate file extension
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".apk") {
		return echo.NewHTTPError(http.StatusBadRequest, "only .apk files are allowed")
	}

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to read uploaded file")
	}
	defer src.Close()

	// Ensure APK directory exists
	if err := os.MkdirAll(h.APKDir, 0755); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create APK directory")
	}

	// Generate filename with timestamp to avoid collisions
	filename := fmt.Sprintf("ota_%d_%s", time.Now().Unix(), filepath.Base(file.Filename))
	dstPath := filepath.Join(h.APKDir, filename)

	// Create destination file
	dst, err := os.Create(dstPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create destination file")
	}
	defer dst.Close()

	// Copy uploaded file content to destination
	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(dstPath) // Clean up on error
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save APK file")
	}

	// Get file size
	info, err := os.Stat(dstPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get file info")
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"filename": filename,
		"version":  extractVersion(filename),
		"size":     info.Size(),
		"url":      fmt.Sprintf("/apk/%s", filename),
	})
}

// ListOTA lists available OTA APK files
// GET /api/v2/fieldtrip/ota/list
func (h *OTAHandler) ListOTA(c echo.Context) error {
	// Ensure APK directory exists
	if _, err := os.Stat(h.APKDir); os.IsNotExist(err) {
		return c.JSON(http.StatusOK, []map[string]interface{}{})
	}

	// Read directory contents
	entries, err := os.ReadDir(h.APKDir)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to read APK directory")
	}

	var files []map[string]interface{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".apk") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, map[string]interface{}{
			"name":    entry.Name(),
			"version": extractVersion(entry.Name()),
			"size":    info.Size(),
			"url":     fmt.Sprintf("/apk/%s", entry.Name()),
		})
	}

	return c.JSON(http.StatusOK, files)
}
