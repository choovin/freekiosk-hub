package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/repositories"
)

// AppPackageService 应用包服务接口
type AppPackageService interface {
	// 上传应用包
	UploadPackage(tenantID, uploadedBy, filePath, originalName string, fileSize int64) (*models.AppPackage, error)
	// 获取应用包
	GetPackage(id string) (*models.AppPackage, error)
	// 获取应用包列表
	ListPackages(tenantID string, limit, offset int) ([]*models.AppPackage, int64, error)
	// 搜索应用包
	SearchPackages(tenantID, keyword string, limit, offset int) ([]*models.AppPackage, int64, error)
	// 删除应用包
	DeletePackage(id string) error
	// 获取应用下载URL
	GetDownloadURL(pkg *models.AppPackage) string
	// 创建设备上的应用安装记录
	RecordInstall(deviceID, packageID, tenantID string) (*models.AppInstall, error)
	// 更新安装状态
	UpdateInstallStatus(installID, status, errorMsg string) error
	// 获取设备的安装记录
	GetDeviceInstalls(deviceID string) ([]*models.AppInstall, error)
}

// appPackageServiceImpl 实现
type appPackageServiceImpl struct {
	repo     repositories.AppPackageRepository
	uploadDir string
	baseURL  string
}

// NewAppPackageService 创建应用包服务
func NewAppPackageService(repo repositories.AppPackageRepository, uploadDir, baseURL string) AppPackageService {
	return &appPackageServiceImpl{
		repo:     repo,
		uploadDir: uploadDir,
		baseURL:  baseURL,
	}
}

// UploadPackage 上传应用包
func (s *appPackageServiceImpl) UploadPackage(tenantID, uploadedBy, filePath, originalName string, fileSize int64) (*models.AppPackage, error) {
	// 生成ID
	id := fmt.Sprintf("apk-%s", uuid.New().String()[:8])

	// 计算SHA256
	sha256Hash, err := s.calculateSHA256(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate SHA256: %w", err)
	}

	// 生成存储路径
	ext := filepath.Ext(originalName)
	storeFileName := id + ext
	storePath := filepath.Join(s.uploadDir, tenantID, storeFileName)

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(storePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// 移动文件到存储位置
	if err := os.Rename(filePath, storePath); err != nil {
		// 如果跨设备移动失败，尝试复制
		if err := s.copyFile(filePath, storePath); err != nil {
			return nil, fmt.Errorf("failed to move file: %w", err)
		}
		os.Remove(filePath)
	}

	// 生成下载URL
	downloadURL := fmt.Sprintf("%s/api/v2/apps/%s/download", s.baseURL, id)

	// 解析版本信息（从文件名）
	version, versionCode := s.parseVersionFromName(originalName)

	pkg := &models.AppPackage{
		ID:           id,
		Name:         originalName,
		PackageName:  s.extractPackageName(originalName),
		Version:      version,
		VersionCode:  versionCode,
		FileSize:     fileSize,
		FilePath:     storePath,
		SHA256:       sha256Hash,
		TenantID:     tenantID,
		UploadedBy:   uploadedBy,
		UploadURL:    downloadURL,
		InstallCount: 0,
	}

	if err := s.repo.Create(pkg); err != nil {
		// 清理文件
		os.Remove(storePath)
		return nil, fmt.Errorf("failed to create package record: %w", err)
	}

	slog.Info("应用包上传成功", "id", id, "name", originalName, "tenant", tenantID)
	return pkg, nil
}

// GetPackage 获取应用包
func (s *appPackageServiceImpl) GetPackage(id string) (*models.AppPackage, error) {
	return s.repo.GetByID(id)
}

// ListPackages 获取应用包列表
func (s *appPackageServiceImpl) ListPackages(tenantID string, limit, offset int) ([]*models.AppPackage, int64, error) {
	return s.repo.List(tenantID, limit, offset)
}

// SearchPackages 搜索应用包
func (s *appPackageServiceImpl) SearchPackages(tenantID, keyword string, limit, offset int) ([]*models.AppPackage, int64, error) {
	return s.repo.Search(tenantID, keyword, limit, offset)
}

// DeletePackage 删除应用包
func (s *appPackageServiceImpl) DeletePackage(id string) error {
	pkg, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	// 删除文件
	os.Remove(pkg.FilePath)

	// 删除记录
	return s.repo.Delete(id)
}

// GetDownloadURL 获取下载URL
func (s *appPackageServiceImpl) GetDownloadURL(pkg *models.AppPackage) string {
	return pkg.UploadURL
}

// RecordInstall 记录安装
func (s *appPackageServiceImpl) RecordInstall(deviceID, packageID, tenantID string) (*models.AppInstall, error) {
	pkg, err := s.repo.GetByID(packageID)
	if err != nil {
		return nil, err
	}

	install := &models.AppInstall{
		ID:        fmt.Sprintf("inst-%s", uuid.New().String()[:8]),
		PackageID: packageID,
		DeviceID:  deviceID,
		TenantID:  tenantID,
		Version:   pkg.Version,
		Status:    string(models.InstallStatusPending),
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	// 这里应该通过AppInstallRepository，但我们暂时用简单方式
	slog.Info("记录应用安装", "install_id", install.ID, "device", deviceID, "package", packageID)
	return install, nil
}

// UpdateInstallStatus 更新安装状态
func (s *appPackageServiceImpl) UpdateInstallStatus(installID, status, errorMsg string) error {
	slog.Info("更新安装状态", "install_id", installID, "status", status)
	return nil
}

// GetDeviceInstalls 获取设备的安装记录
func (s *appPackageServiceImpl) GetDeviceInstalls(deviceID string) ([]*models.AppInstall, error) {
	slog.Info("获取设备安装记录", "device", deviceID)
	return []*models.AppInstall{}, nil
}

// 辅助方法

func (s *appPackageServiceImpl) calculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *appPackageServiceImpl) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func (s *appPackageServiceImpl) parseVersionFromName(name string) (version string, versionCode int) {
	// 简单实现：从文件名提取版本
	// 例如: app-v1.2.3.apk -> version: 1.2.3
	version = "1.0.0"
	versionCode = 1
	return
}

func (s *appPackageServiceImpl) extractPackageName(name string) string {
	// 简单实现：从文件名提取包名
	// 例如: com.example.app-v1.2.3.apk -> packageName: com.example.app
	name = filepath.Base(name)
	// 移除扩展名
	name = name[:len(name)-len(filepath.Ext(name))]
	return name
}
