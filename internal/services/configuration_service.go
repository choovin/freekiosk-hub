package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/repositories"
)

// ConfigurationService 配置档案服务接口
type ConfigurationService interface {
	// 创键配置档案
	Create(cfg *models.ConfigurationProfile) error
	// 获取配置档案
	GetByID(id string) (*models.ConfigurationProfile, error)
	// 更新配置档案
	Update(cfg *models.ConfigurationProfile) error
	// 删除配置档案
	Delete(id string) error
	// 获取配置档案列表
	List(tenantID string, limit, offset int) ([]*models.ConfigurationProfile, int64, error)
	// 分配配置到设备
	AssignToDevice(deviceID, configID, tenantID, assignedBy string) error
	// 取消分配
	UnassignFromDevice(deviceID, configID string) error
	// 获取设备当前配置
	GetDeviceConfiguration(deviceID string) (*models.ConfigurationProfile, error)
	// 获取设备所有配置
	GetDeviceConfigurations(deviceID string) ([]*models.ConfigurationProfile, error)
}

// configurationServiceImpl 配置档案服务实现
type configurationServiceImpl struct {
	repo repositories.ConfigurationRepository
}

// NewConfigurationService 创建配置档案服务
func NewConfigurationService(repo repositories.ConfigurationRepository) ConfigurationService {
	return &configurationServiceImpl{repo: repo}
}

// Create 创键配置档案
func (s *configurationServiceImpl) Create(cfg *models.ConfigurationProfile) error {
	cfg.ID = fmt.Sprintf("cfg-%s", uuid.New().String()[:8])
	now := time.Now().Unix()
	cfg.CreatedAt = now
	cfg.UpdatedAt = now
	return s.repo.Create(cfg)
}

// GetByID 获取配置档案
func (s *configurationServiceImpl) GetByID(id string) (*models.ConfigurationProfile, error) {
	return s.repo.GetByID(id)
}

// Update 更新配置档案
func (s *configurationServiceImpl) Update(cfg *models.ConfigurationProfile) error {
	cfg.UpdatedAt = time.Now().Unix()
	return s.repo.Update(cfg)
}

// Delete 删除配置档案
func (s *configurationServiceImpl) Delete(id string) error {
	return s.repo.Delete(id)
}

// List 获取配置档案列表
func (s *configurationServiceImpl) List(tenantID string, limit, offset int) ([]*models.ConfigurationProfile, int64, error) {
	return s.repo.List(tenantID, limit, offset)
}

// AssignToDevice 分配配置到设备
func (s *configurationServiceImpl) AssignToDevice(deviceID, configID, tenantID, assignedBy string) error {
	// 验证配置存在
	cfg, err := s.repo.GetByID(configID)
	if err != nil {
		return fmt.Errorf("configuration not found: %w", err)
	}
	if cfg.TenantID != tenantID {
		return fmt.Errorf("configuration does not belong to this tenant")
	}
	return s.repo.AssignToDevice(deviceID, configID, tenantID, assignedBy)
}

// UnassignFromDevice 取消分配
func (s *configurationServiceImpl) UnassignFromDevice(deviceID, configID string) error {
	return s.repo.UnassignFromDevice(deviceID, configID)
}

// GetDeviceConfiguration 获取设备当前配置
func (s *configurationServiceImpl) GetDeviceConfiguration(deviceID string) (*models.ConfigurationProfile, error) {
	return s.repo.GetDeviceConfiguration(deviceID)
}

// GetDeviceConfigurations 获取设备所有配置
func (s *configurationServiceImpl) GetDeviceConfigurations(deviceID string) ([]*models.ConfigurationProfile, error) {
	return s.repo.GetDeviceConfigurations(deviceID)
}
