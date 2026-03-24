package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
	"github.com/wared2003/freekiosk-hub/internal/services"

	"github.com/labstack/echo/v4"
)

// AppPackageHandler 应用包HTTP处理器
type AppPackageHandler struct {
	svc      services.AppPackageService
	uploadDir string
}

// NewAppPackageHandler 创建应用包处理器
func NewAppPackageHandler(svc services.AppPackageService, uploadDir string) *AppPackageHandler {
	return &AppPackageHandler{svc: svc, uploadDir: uploadDir}
}

// HandleListPackages 获取应用包列表
func (h *AppPackageHandler) HandleListPackages(c echo.Context) error {
	tenantID := c.QueryParam("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	packages, total, err := h.svc.ListPackages(tenantID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"packages": packages,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// HandleSearchPackages 搜索应用包
func (h *AppPackageHandler) HandleSearchPackages(c echo.Context) error {
	tenantID := c.QueryParam("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	keyword := c.QueryParam("keyword")
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	packages, total, err := h.svc.SearchPackages(tenantID, keyword, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"packages": packages,
		"total":    total,
	})
}

// HandleGetPackage 获取单个应用包
func (h *AppPackageHandler) HandleGetPackage(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "package id is required"})
	}

	pkg, err := h.svc.GetPackage(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "package not found"})
	}

	return c.JSON(http.StatusOK, pkg)
}

// HandleUploadPackage 上传应用包
func (h *AppPackageHandler) HandleUploadPackage(c echo.Context) error {
	tenantID := c.FormValue("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	uploadedBy := c.FormValue("uploaded_by")

	// 获取上传的文件
	file, err := c.FormFile("apk_file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "apk_file is required"})
	}

	// 打开上传的文件
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to open file"})
	}
	defer src.Close()

	// 创建临时文件
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, fmt.Sprintf("upload-%s-%s", uuid.New().String()[:8], file.Filename))
	dst, err := os.Create(tempFile)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create temp file"})
	}
	defer os.Remove(tempFile)
	defer dst.Close()

	// 复制内容
	if _, err := io.Copy(dst, src); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save file"})
	}

	// 创建应用包
	pkg, err := h.svc.UploadPackage(tenantID, uploadedBy, tempFile, file.Filename, file.Size)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, pkg)
}

// HandleDeletePackage 删除应用包
func (h *AppPackageHandler) HandleDeletePackage(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "package id is required"})
	}

	if err := h.svc.DeletePackage(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "package deleted"})
}

// HandleDownloadPackage 下载应用包
func (h *AppPackageHandler) HandleDownloadPackage(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "package id is required"})
	}

	pkg, err := h.svc.GetPackage(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "package not found"})
	}

	// 检查文件是否存在
	if _, err := os.Stat(pkg.FilePath); os.IsNotExist(err) {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "file not found"})
	}

	return c.File(pkg.FilePath)
}

// HandleGetDownloadURL 获取下载URL
func (h *AppPackageHandler) HandleGetDownloadURL(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "package id is required"})
	}

	pkg, err := h.svc.GetPackage(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "package not found"})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"download_url": pkg.UploadURL,
		"sha256":      pkg.SHA256,
	})
}
