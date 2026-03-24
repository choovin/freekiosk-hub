package services

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/repositories"
)

// MDMTabletService MDM平板设备服务接口
type MDMTabletService interface {
	// 设备管理
	CreateDevice(device *models.MDMTablet) error
	GetDevice(id string) (*models.MDMTablet, error)
	GetDeviceByNumber(number string) (*models.MDMTablet, error)
	UpdateDevice(device *models.MDMTablet) error
	DeleteDevice(id string) error
	ListDevices(tenantID string, limit, offset int) ([]*models.MDMTablet, int64, error)
	SearchDevices(filter *models.DeviceSearchFilter) ([]*models.MDMTablet, int64, error)
	UpdateDeviceStatus(id string, status models.MDMTabletStatus) error

	// 设备分组管理
	CreateGroup(group *models.MDMTabletGroup) error
	UpdateGroup(group *models.MDMTabletGroup) error
	DeleteGroup(id string) error
	ListGroups(tenantID string) ([]*models.MDMTabletGroup, error)
	AssignDeviceToGroup(deviceID, groupID string) error
	UnassignDeviceFromGroup(deviceID string) error

	// 设备标签管理
	AddTag(tag *models.MDMTabletTag) error
	RemoveTag(deviceID, tagName string) error
	GetDeviceTags(deviceID string) ([]*models.MDMTabletTag, error)

	// 设备位置管理
	UpdateLocation(deviceID string, lat, lng float64, timestamp int64) error
	GetDeviceLocation(deviceID string) (*models.GPSData, error)

	// 设备事件管理
	RecordEvent(event *models.MDMTabletEvent) error
	GetDeviceEvents(deviceID string, limit int) ([]*models.MDMTabletEvent, error)

	// 批量操作
	BulkUpdateStatus(deviceIDs []string, status models.MDMTabletStatus) error
	BulkAssignGroup(deviceIDs []string, groupID string) error
}

// mdmTabletServiceImpl MDM平板设备服务实现
type mdmTabletServiceImpl struct {
	repo repositories.MDMTabletRepository
}

// NewMDMTabletService 创建MDM平板设备服务
func NewMDMTabletService(repo repositories.MDMTabletRepository) MDMTabletService {
	return &mdmTabletServiceImpl{repo: repo}
}

// CreateDevice 创建设备
func (s *mdmTabletServiceImpl) CreateDevice(device *models.MDMTablet) error {
	// 生成ID如果未提供
	if device.ID == "" {
		device.ID = generateDeviceID()
	}
	device.Status = string(models.MDMTabletStatusActive)
	device.CreatedAt = time.Now().Unix()
	device.UpdatedAt = time.Now().Unix()

	slog.Info("创建设备", "id", device.ID, "name", device.Name, "tenant", device.TenantID)
	return s.repo.CreateDevice(device)
}

// GetDevice 获取设备
func (s *mdmTabletServiceImpl) GetDevice(id string) (*models.MDMTablet, error) {
	device, err := s.repo.GetDeviceByID(id)
	if err != nil {
		slog.Warn("获取设备失败", "id", id, "error", err)
		return nil, repositories.ErrDeviceNotFound
	}
	return device, nil
}

// GetDeviceByNumber 根据编号获取设备
func (s *mdmTabletServiceImpl) GetDeviceByNumber(number string) (*models.MDMTablet, error) {
	device, err := s.repo.GetDeviceByNumber(number)
	if err != nil {
		slog.Warn("根据编号获取设备失败", "number", number, "error", err)
		return nil, repositories.ErrDeviceNotFound
	}
	return device, nil
}

// UpdateDevice 更新设备
func (s *mdmTabletServiceImpl) UpdateDevice(device *models.MDMTablet) error {
	device.UpdatedAt = time.Now().Unix()
	slog.Info("更新设备", "id", device.ID, "name", device.Name)
	return s.repo.UpdateDevice(device)
}

// DeleteDevice 删除设备（软删除）
func (s *mdmTabletServiceImpl) DeleteDevice(id string) error {
	slog.Info("删除设备", "id", id)
	return s.repo.DeleteDevice(id)
}

// ListDevices 获取设备列表
func (s *mdmTabletServiceImpl) ListDevices(tenantID string, limit, offset int) ([]*models.MDMTablet, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListDevices(tenantID, limit, offset)
}

// SearchDevices 搜索设备
func (s *mdmTabletServiceImpl) SearchDevices(filter *models.DeviceSearchFilter) ([]*models.MDMTablet, int64, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}
	slog.Debug("搜索设备", "tenant", filter.TenantID, "status", filter.Status, "search", filter.Search)
	return s.repo.SearchDevices(filter)
}

// UpdateDeviceStatus 更新设备状态
func (s *mdmTabletServiceImpl) UpdateDeviceStatus(id string, status models.MDMTabletStatus) error {
	device, err := s.repo.GetDeviceByID(id)
	if err != nil {
		return repositories.ErrDeviceNotFound
	}
	device.Status = string(status)
	device.UpdatedAt = time.Now().Unix()
	slog.Info("更新设备状态", "id", id, "status", status)
	return s.repo.UpdateDevice(device)
}

// CreateGroup 创建设备分组
func (s *mdmTabletServiceImpl) CreateGroup(group *models.MDMTabletGroup) error {
	if group.ID == "" {
		group.ID = generateGroupID()
	}
	group.CreatedAt = time.Now().Unix()
	group.UpdatedAt = time.Now().Unix()
	slog.Info("创建设备分组", "id", group.ID, "name", group.Name, "tenant", group.TenantID)
	return s.repo.CreateGroup(group)
}

// UpdateGroup 更新设备分组
func (s *mdmTabletServiceImpl) UpdateGroup(group *models.MDMTabletGroup) error {
	group.UpdatedAt = time.Now().Unix()
	slog.Info("更新设备分组", "id", group.ID, "name", group.Name)
	return s.repo.UpdateGroup(group)
}

// DeleteGroup 删除设备分组
func (s *mdmTabletServiceImpl) DeleteGroup(id string) error {
	slog.Info("删除设备分组", "id", id)
	return s.repo.DeleteGroup(id)
}

// ListGroups 获取设备分组列表
func (s *mdmTabletServiceImpl) ListGroups(tenantID string) ([]*models.MDMTabletGroup, error) {
	return s.repo.ListGroups(tenantID)
}

// AssignDeviceToGroup 分配设备到分组
func (s *mdmTabletServiceImpl) AssignDeviceToGroup(deviceID, groupID string) error {
	device, err := s.repo.GetDeviceByID(deviceID)
	if err != nil {
		return repositories.ErrDeviceNotFound
	}
	device.GroupID = &groupID
	device.UpdatedAt = time.Now().Unix()
	slog.Info("分配设备到分组", "device", deviceID, "group", groupID)
	return s.repo.UpdateDevice(device)
}

// UnassignDeviceFromGroup 从分组移除设备
func (s *mdmTabletServiceImpl) UnassignDeviceFromGroup(deviceID string) error {
	device, err := s.repo.GetDeviceByID(deviceID)
	if err != nil {
		return repositories.ErrDeviceNotFound
	}
	device.GroupID = nil
	device.UpdatedAt = time.Now().Unix()
	slog.Info("从分组移除设备", "device", deviceID)
	return s.repo.UpdateDevice(device)
}

// AddTag 添加设备标签
func (s *mdmTabletServiceImpl) AddTag(tag *models.MDMTabletTag) error {
	if tag.ID == "" {
		tag.ID = generateTagID()
	}
	tag.CreatedAt = time.Now().Unix()
	slog.Debug("添加设备标签", "device", tag.DeviceID, "tag", tag.Tag, "value", tag.Value)
	return s.repo.AddTag(tag)
}

// RemoveTag 移除设备标签
func (s *mdmTabletServiceImpl) RemoveTag(deviceID, tagName string) error {
	slog.Debug("移除设备标签", "device", deviceID, "tag", tagName)
	return s.repo.RemoveTag(deviceID, tagName)
}

// GetDeviceTags 获取设备标签
func (s *mdmTabletServiceImpl) GetDeviceTags(deviceID string) ([]*models.MDMTabletTag, error) {
	return s.repo.GetDeviceTags(deviceID)
}

// UpdateLocation 更新设备位置
func (s *mdmTabletServiceImpl) UpdateLocation(deviceID string, lat, lng float64, timestamp int64) error {
	slog.Debug("更新设备位置", "device", deviceID, "lat", lat, "lng", lng)
	return s.repo.UpdateLocation(deviceID, lat, lng, timestamp)
}

// GetDeviceLocation 获取设备位置
func (s *mdmTabletServiceImpl) GetDeviceLocation(deviceID string) (*models.GPSData, error) {
	return s.repo.GetDeviceLocation(deviceID)
}

// RecordEvent 记录设备事件
func (s *mdmTabletServiceImpl) RecordEvent(event *models.MDMTabletEvent) error {
	if event.ID == "" {
		event.ID = generateEventID()
	}
	event.CreatedAt = time.Now().Unix()
	slog.Debug("记录设备事件", "device", event.DeviceID, "type", event.EventType)
	return s.repo.RecordEvent(event)
}

// GetDeviceEvents 获取设备事件
func (s *mdmTabletServiceImpl) GetDeviceEvents(deviceID string, limit int) ([]*models.MDMTabletEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	return s.repo.GetDeviceEvents(deviceID, limit)
}

// BulkUpdateStatus 批量更新设备状态
func (s *mdmTabletServiceImpl) BulkUpdateStatus(deviceIDs []string, status models.MDMTabletStatus) error {
	slog.Info("批量更新设备状态", "count", len(deviceIDs), "status", status)
	return s.repo.BulkUpdateStatus(deviceIDs, string(status))
}

// BulkAssignGroup 批量分配设备到分组
func (s *mdmTabletServiceImpl) BulkAssignGroup(deviceIDs []string, groupID string) error {
	slog.Info("批量分配设备到分组", "count", len(deviceIDs), "group", groupID)
	return s.repo.BulkAssignGroup(deviceIDs, groupID)
}

// ID生成函数
func generateDeviceID() string {
	return fmt.Sprintf("mdm-%s", uuid.New().String()[:8])
}

func generateGroupID() string {
	return fmt.Sprintf("grp-%s", uuid.New().String()[:8])
}

func generateTagID() string {
	return fmt.Sprintf("tag-%s", uuid.New().String()[:8])
}

func generateEventID() string {
	return fmt.Sprintf("evt-%s", uuid.New().String()[:8])
}
